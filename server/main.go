package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

var (
	imgMu   sync.Mutex
	imgHits int
)

const page = `<!DOCTYPE html>
<html>
<head>
	<title>Example Domain</title>
</head>
<body>
	<h1>Example Domain</h1>
	<p>This domain is for use in illustrative examples in documents.</p>
	<p><a href="https://example.com">More information...</a></p>
	<ul>
		<li><a href="/login">Log in</a></li>
		<li><a href="/dashboard">Dashboard</a></li>
		<li><a href="/news">News</a></li>
		<li><a href="/account">Account</a></li>
	</ul>
	<img src="/img.png" alt="blocked image" width="10" height="10">
	<script>
		fetch("/api/data?n=1");
		fetch("/api/data?n=2");
	</script>
</body>
</html>`

const dashboard = `<!DOCTYPE html>
<html>
<head>
	<title>Dashboard</title>
</head>
<body id="dashboard">
	<h1>Dashboard</h1>
	<p>Welcome back! Content is loading...</p>
	<div id="results" style="display:none"></div>
	<script>
		setTimeout(function () {
			var el = document.getElementById("results");
			el.style.display = "block";
			el.textContent = "Data loaded at " + new Date().toISOString();
		}, 1500);
	</script>
</body>
</html>`

const login = `<!DOCTYPE html>
<html>
<head>
	<title>Log in</title>
</head>
<body>
	<h1>Log in</h1>
	<form action="/login" method="post">
		<label>Username <input id="username" name="username" type="text"></label>
		<label>Password <input id="password" name="password" type="password"></label>
		<button id="submit" type="submit">Log in</button>
	</form>
</body>
</html>`

const news = `<!DOCTYPE html>
<html>
<head>
	<title>News</title>
</head>
<body>
	<center><h1>News</h1></center>
	<table class="itemlist">
		<tr class="athing" id="1">
			<td class="title"><span class="titleline"><a href="https://go.dev/blog">Go 1.26 released</a></span></td>
		</tr>
		<tr class="athing" id="2">
			<td class="title"><span class="titleline"><a href="https://chromedp.io">chromedp drives Chrome over CDP</a></span></td>
		</tr>
		<tr class="athing" id="3">
			<td class="title"><span class="titleline"><a href="https://example.com">Example domain for documentation</a></span></td>
		</tr>
	</table>
</body>
</html>`

const consolePage = `<!DOCTYPE html>
<html>
<head>
	<title>Console</title>
</head>
<body>
	<h1>Console test page</h1>
	<script>
		console.log("hello from the page");
		console.info("an info message");
		console.warn("a warning");
		console.error("an error message");
		setTimeout(function () {
			throw new Error("boom: something broke");
		}, 500);
	</script>
</body>
</html>`

func imgHandler(w http.ResponseWriter, r *http.Request) {
	imgMu.Lock()
	imgHits++
	imgMu.Unlock()
	w.Header().Set("Content-Type", "image/png")
	w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
}

func countersHandler(w http.ResponseWriter, r *http.Request) {
	imgMu.Lock()
	defer imgMu.Unlock()
	if r.Method == http.MethodPost {
		imgHits = 0
		fmt.Fprintln(w, "reset")
		return
	}
	fmt.Fprintf(w, "img_requests=%d\n", imgHits)
}

func accountHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	value := "none"
	if err == nil {
		value = cookie.Value
	}
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Account</title></head><body>
	<h1>Account</h1><p id="session">Cookie value: %s</p></body></html>`, value)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "server-set-cookie", Path: "/"})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, page)
	})

	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, dashboard)
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, login)
	})

	mux.HandleFunc("/news", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, news)
	})

	mux.HandleFunc("/account", accountHandler)

	mux.HandleFunc("/console", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, consolePage)
	})

	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"n":%q,"ts":"%s"}`, r.URL.Query().Get("n"), "some-json-body")
	})

	mux.HandleFunc("/img.png", imgHandler)
	mux.HandleFunc("/debug/counters", countersHandler)

	addr := ":8080"
	log.Printf("test webserver listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
