#!/usr/bin/env bash
set -u

cd "$(dirname "$0")"

PASS=0
FAIL=0

server_pid=""
cleanup() {
  [ -n "$server_pid" ] && kill "$server_pid" 2>/dev/null
}
trap cleanup EXIT

go build -o /tmp/cdp_server ./server || exit 1
/tmp/cdp_server > /tmp/cdp_server.log 2>&1 &
server_pid=$!

for i in $(seq 1 50); do
  if curl -s -o /dev/null "http://localhost:8080/"; then break; fi
  sleep 0.2
done

run_example() {
  local name="$1"
  local expect="$2"
  local dir="examples/$name"
  local out
  out=$(cd "$dir" && go run . 2>&1)
  local code=$?
  if [ $code -ne 0 ]; then
    echo "FAIL  $name (exit $code)"
    echo "      $out"
    FAIL=$((FAIL+1))
  elif echo "$out" | grep -qE "$expect"; then
    echo "PASS  $name"
    PASS=$((PASS+1))
  else
    echo "FAIL  $name (no match for /$expect/)"
    echo "      $out"
    FAIL=$((FAIL+1))
  fi
}

run_example title           "Page title: Example Domain"
run_example text_attrs      "Heading: Example Domain"
run_example wait_dynamic    "Results: Data loaded at "
run_example forms           "After login, at: http://localhost:8080/dashboard"
run_example screenshot      "wrote screenshot.jpg"
run_example evaluate        "Number of links: 5"
run_example chrome_opts     "Page title: Example Domain"
run_example scrape          "^1. Go 1.26 released"
run_example cookies_read    "session_id=server-set-cookie"
run_example cookies_set     "Account page shows: Cookie value: abc123"
run_example net_log         "request: GET http://localhost:8080/"
run_example net_block       "image blocked"
run_example net_body        "JSON body:"
run_example perf_metrics    "= "
run_example perf_timing     "domContentLoaded:"
run_example console         "console.log"

echo "----"
echo "PASS=$PASS FAIL=$FAIL"
[ $FAIL -eq 0 ]
