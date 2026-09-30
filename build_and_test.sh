#!/usr/bin/env bash
#
# Build and test everything in this repo.
#
#   ./build_and_test.sh              build, vet, fmt, README check, then run all examples
#   ./build_and_test.sh --no-run     skip the browser tests (build/lint/README checks only)
#   ./build_and_test.sh --timeout 60 per-example browser timeout in seconds (default 45)
#
# Every browser example is wrapped in a hard timeout. Four of them
# (cookies_read, net_body, net_log, perf_metrics) have no context.WithTimeout
# of their own, so if Chrome cannot start they would block forever and this
# script would never reach the summary.

set -uo pipefail

cd "$(dirname "$0")"

PER_EXAMPLE_TIMEOUT=45
RUN_TESTS=1

while [ $# -gt 0 ]; do
	case "$1" in
	--no-run)
		RUN_TESTS=0
		shift
		;;
	--timeout)
		PER_EXAMPLE_TIMEOUT="${2:?--timeout needs a value}"
		shift 2
		;;
	-h | --help)
		sed -n '2,14p' "$0"
		exit 0
		;;
	*)
		echo "unknown flag: $1" >&2
		exit 2
		;;
	esac
done

RED=$'\033[31m'
GREEN=$'\033[32m'
YELLOW=$'\033[33m'
BOLD=$'\033[1m'
OFF=$'\033[0m'

FAILURES=()
STEP=0

step() {
	STEP=$((STEP + 1))
	printf '\n%s[%d] %s%s\n' "$BOLD" "$STEP" "$1" "$OFF"
}

fail() {
	FAILURES+=("$1")
	printf '%sFAIL%s  %s\n' "$RED" "$OFF" "$1"
}

ok() {
	printf '%sOK%s    %s\n' "$GREEN" "$OFF" "$1"
}

warn() {
	printf '%sWARN%s  %s\n' "$YELLOW" "$OFF" "$1"
}

# ---------------------------------------------------------------- gofmt
step "Checking formatting (gofmt -l)"
unformatted=$(gofmt -l . 2>/dev/null)
if [ -n "$unformatted" ]; then
	fail "gofmt: these files are not formatted:"
	printf '        %s\n' $unformatted
else
	ok "all files gofmt-clean"
fi

# ---------------------------------------------------------------- build + vet
step "Building all packages"
if build_out=$(go build ./... 2>&1); then
	ok "go build ./..."
else
	fail "go build ./..."
	printf '%s\n' "$build_out"
fi

step "Running go vet"
if vet_out=$(go vet ./... 2>&1); then
	ok "go vet ./..."
else
	fail "go vet ./..."
	printf '%s\n' "$vet_out"
fi

# ------------------------------------------------- README vs source parity
# Every full `package main` program in the README is expected to correspond to
# an example in examples/. The only permitted differences are:
#   - the Navigate() URL (README uses public URLs, examples use localhost:8080)
#   - example-only additions the README deliberately elides (e.g. timeouts)
step "Checking README code blocks against examples/"
if [ ! -f README.md ]; then
	warn "no README.md, skipping"
else
	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT

	# Extract each ```go fenced block into its own file.
	awk -v dir="$tmp" '
		/^```go$/ { n++; out = sprintf("%s/block%02d.go", dir, n); inblock = 1; next }
		/^```$/    { if (inblock) { close(out); inblock = 0 } ; next }
		inblock    { print > out }
	' README.md

	full_blocks=$(grep -l '^package main' "$tmp"/block*.go 2>/dev/null | sort)
	if [ -z "$full_blocks" ]; then
		warn "no full programs found in README"
	else
		for block in $full_blocks; do
			name=$(basename "$block" .go)
			best=""
			best_score=-1
			for ex in examples/*/main.go; do
				[ -f "$ex" ] || continue
				# 0 differing lines means byte-identical. Everything else is
				# scored by differing-line count, so lower is a closer match.
				score=$(diff "$block" "$ex" | grep -c '^[<>]' || true)
				[ -n "$score" ] || score=0
				if [ "$best_score" -lt 0 ] || [ "$score" -lt "$best_score" ]; then
					best="$ex"
					best_score="$score"
				fi
			done

			if [ "$best_score" -eq 0 ]; then
				ok "$name is byte-identical to $best"
			else
				# Show only the differing lines so drift is reviewable at a glance.
				diffout=$(diff "$best" "$block" | grep '^[<>]' || true)
				printf '%sWARN%s  %s differs from %s (%s lines):\n' \
					"$YELLOW" "$OFF" "$name" "$best" "$best_score"
				printf '%s\n' "$diffout" | sed 's/^/          /'
			fi
		done
	fi

	# Fragments (blocks that are not standalone programs) are partial excerpts of
	# an example. Compare them line-by-line against the closest example after
	# normalising URLs and string literals, so that a stale snippet is caught.
	step "Checking README snippets against examples/ (normalised)"
	python3 - "$tmp" <<'PY'
import glob, os, re, sys

tmp = sys.argv[1]

def norm(text):
    out = []
    for line in text.split("\n"):
        line = re.sub(r"(?<!:)//.*$", "", line)   # strip comments, keep https://
        line = re.sub(r"`[^`]*`", "`LIT`", line)   # backticked literals
        line = re.sub(r'"[^"]*"', '"LIT"', line)   # quoted literals
        line = line.strip()
        if line:
            out.append(line)
    return out

examples = {p: norm(open(p).read()) for p in sorted(glob.glob("examples/*/main.go"))}
blocks = sorted(glob.glob(os.path.join(tmp, "block*.go")))
fragments = [b for b in blocks if not open(b).read().startswith("package main")]

if not fragments:
    print("no snippet blocks found")
    sys.exit(0)

drift = 0
for block in fragments:
    lines = norm(open(block).read())
    best_ratio, best_path = max(
        (sum(1 for l in lines if l in body) / len(lines), p)
        for p, body in examples.items()
    )
    label = os.path.basename(block)[:-3]
    if best_ratio == 1.0:
        print(f"  \033[32mOK\033[0m    {label} matches {best_path}")
    else:
        drift += 1
        print(f"  \033[33mWARN\033[0m  {label} vs {best_path} ({best_ratio*100:.0f}%):")
        for l in lines:
            if l not in examples[best_path]:
                print(f"          only in README: {l}")

sys.exit(1 if drift else 0)
PY
	if [ $? -eq 0 ]; then
		ok "all README snippets trace back to an example"
	else
		warn "some README snippets diverge from their example (see above)"
	fi

	# Compile the README's full programs exactly as written. This catches a
	# snippet that was edited without the matching example being updated.
	step "Compiling README programs verbatim"
	checkdir="$PWD/.readmecheck"
	rm -rf "$checkdir"
	mkdir -p "$checkdir"
	compiled=0
	verbatim_failed=0
	for block in "$tmp"/block*.go; do
		if head -1 "$block" | grep -q '^package main'; then
			name=$(basename "$block" .go)
			mkdir -p "$checkdir/$name"
			cp "$block" "$checkdir/$name/main.go"
			if out=$(cd "$checkdir/$name" && go build -o /dev/null . 2>&1); then
				ok "$name compiles verbatim"
				compiled=$((compiled + 1))
			else
				fail "$name does not compile verbatim from the README"
				printf '%s\n' "$out" | sed 's/^/        /'
				verbatim_failed=$((verbatim_failed + 1))
			fi
		fi
	done
	rm -rf "$checkdir"
	[ "$compiled" -gt 0 ] || warn "no full README programs were found"
fi

# ---------------------------------------------------------------- example builds
step "Compiling every example"
for dir in examples/*/; do
	name=$(basename "$dir")
	if out=$(cd "$dir" && go build -o /dev/null . 2>&1); then
		ok "$name"
	else
		fail "$name failed to compile"
		printf '%s\n' "$out" | sed 's/^/        /'
	fi
done

# ---------------------------------------------------------------- browser tests
if [ "$RUN_TESTS" -eq 0 ]; then
	step "Skipping browser tests (--no-run)"
else
	step "Running browser examples against the local test server"

	server_bin=$(mktemp -d)/cdp_server
	server_pid=""
	cleanup() {
		[ -n "$server_pid" ] && kill "$server_pid" 2>/dev/null
		return 0
	}
	trap 'cleanup; rm -rf "$tmp"' EXIT

	if ! go build -o "$server_bin" ./server 2>/dev/null; then
		fail "could not build ./server"
	else
		"$server_bin" >/tmp/cdp_server.log 2>&1 &
		server_pid=$!

		up=0
		for _ in $(seq 1 50); do
			if curl -s -o /dev/null --max-time 2 "http://localhost:8080/"; then
				up=1
				break
			fi
			sleep 0.2
		done

		if [ "$up" -ne 1 ]; then
			fail "test server did not come up on :8080"
			tail -5 /tmp/cdp_server.log | sed 's/^/        /'
		else
			ok "test server listening on :8080"

			run_example() {
				local name="$1" expect="$2"
				local out code
				out=$(cd "examples/$name" && timeout "$PER_EXAMPLE_TIMEOUT" go run . 2>&1)
				code=$?
				if [ "$code" -eq 124 ]; then
					fail "$name timed out after ${PER_EXAMPLE_TIMEOUT}s (no context.WithTimeout in the example?)"
				elif [ "$code" -ne 0 ]; then
					fail "$name exited $code"
					printf '%s\n' "$out" | sed 's/^/        /'
				elif printf '%s' "$out" | grep -qE "$expect"; then
					ok "$name"
				else
					fail "$name ran but output did not match /$expect/"
					printf '%s\n' "$out" | sed 's/^/        /'
				fi
			}

			run_example title "Page title: Example Domain"
			run_example text_attrs "Heading: Example Domain"
			run_example wait_dynamic "Results: Data loaded at "
			run_example forms "After login, at: http://localhost:8080/dashboard"
			run_example screenshot "wrote screenshot.jpg"
			run_example evaluate "Number of links: 5"
			run_example chrome_opts "Page title: Example Domain"
			run_example scrape "^1\. Go 1\.26 released"
			run_example cookies_read "session_id=server-set-cookie"
			run_example cookies_set "Account page shows: Cookie value: abc123"
			run_example net_log "request: GET http://localhost:8080/"
			run_example net_block "image blocked"
			run_example net_body "JSON body:"
			run_example perf_metrics "= "
			run_example perf_timing "domContentLoaded:"
			run_example console "console.log"
		fi
	fi
fi

# ---------------------------------------------------------------- summary
printf '\n%s────────────────────────────────────────%s\n' "$BOLD" "$OFF"
if [ ${#FAILURES[@]} -eq 0 ]; then
	printf '%sALL CHECKS PASSED%s\n' "$GREEN$BOLD" "$OFF"
	exit 0
fi

printf '%s%d FAILURE(S):%s\n' "$RED$BOLD" "${#FAILURES[@]}" "$OFF"
for f in "${FAILURES[@]}"; do
	printf '  - %s\n' "$f"
done
exit 1
