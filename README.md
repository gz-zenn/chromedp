# Browser Automation in Go with chromedp

Go doesn't have a built-in browser automation library, but [chromedp](https://github.com/chromedp/chromedp) fills that gap nicely. It drives Chrome (or any Chromium-based browser) via the Chrome DevTools Protocol (CDP) — no Selenium, no WebDriver, no external binaries beyond Chrome itself. This makes it a popular choice for scraping JavaScript-heavy sites, taking screenshots, generating PDFs, and running end-to-end browser tests, all from pure Go.

## Installation

```bash
go get github.com/chromedp/chromedp
```

chromedp needs a local Chrome or Chromium install. It will try to find one automatically, but you can also point it at a specific binary (shown later).

## The core concepts

Everything in chromedp revolves around two ideas:

- **Context** — chromedp uses Go's `context.Context` to represent a browser tab. `chromedp.NewContext` creates one; canceling it closes the browser.
- **Actions and Tasks** — you build a list of `chromedp.Action` values (navigate, click, wait, extract text, etc.) and run them with `chromedp.Run`. A `chromedp.Tasks` value is just `[]Action` run in sequence.

## A minimal example

This navigates to a page and grabs its title:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var title string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://example.com`),
		chromedp.Title(&title),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Page title:", title)
}
```

Note the two contexts: one from `chromedp.NewContext` (the browser session) wrapped in a `context.WithTimeout` (so a hung page doesn't run forever).

## Extracting text and attributes

```go
var heading string
var href string

err := chromedp.Run(ctx,
	chromedp.Navigate(`https://example.com`),
	chromedp.Text(`h1`, &heading, chromedp.NodeVisible),
	chromedp.AttributeValue(`a`, "href", &href, nil),
)
```

`chromedp.Text` pulls the rendered text of the first matching node; `chromedp.AttributeValue` reads an HTML attribute. Both take a CSS selector as the first argument.

## Waiting for dynamic content

Since many pages render via JavaScript, you often need to wait for an element to appear before interacting with it:

```go
err := chromedp.Run(ctx,
	chromedp.Navigate(`https://example.com/dashboard`),
	chromedp.WaitVisible(`#results`, chromedp.ByID),
	chromedp.Text(`#results`, &output),
)
```

`WaitVisible`, `WaitReady`, and `WaitNotPresent` cover most timing issues without resorting to `time.Sleep`.

## Clicking, typing, and forms

```go
err := chromedp.Run(ctx,
	chromedp.Navigate(`https://example.com/login`),
	chromedp.SendKeys(`#username`, "myuser"),
	chromedp.SendKeys(`#password`, "mypassword"),
	chromedp.Click(`#submit`, chromedp.NodeVisible),
	chromedp.WaitVisible(`#dashboard`),
)
```

Actions run in order, so this fills in both fields, clicks submit, and waits for the resulting page to load.

## Taking a screenshot

```go
var buf []byte

err := chromedp.Run(ctx,
	chromedp.Navigate(`https://example.com`),
	chromedp.FullScreenshot(&buf, 90), // 90 = JPEG quality
)
if err != nil {
	log.Fatal(err)
}
if err := os.WriteFile("screenshot.jpg", buf, 0644); err != nil {
	log.Fatal(err)
}
```

Use `chromedp.CaptureScreenshot` for just the viewport, or `chromedp.Screenshot(selector, &buf)` to capture a single element.

## Running JavaScript directly

Sometimes it's easier to just evaluate JS in the page context:

```go
var result int
err := chromedp.Run(ctx,
	chromedp.Navigate(`https://example.com`),
	chromedp.Evaluate(`document.querySelectorAll("a").length`, &result),
)
fmt.Println("Number of links:", result)
```

## Configuring Chrome options

By default chromedp launches headless Chrome. You can customize this with `chromedp.NewExecAllocator`:

```go
opts := append(chromedp.DefaultExecAllocatorOptions[:],
	chromedp.Flag("headless", false), // show the browser window
	chromedp.Flag("disable-gpu", true),
	chromedp.UserAgent("MyCustomAgent/1.0"),
	chromedp.ExecPath("/usr/bin/chromium-browser"), // custom Chrome path
)

allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
defer cancel()

ctx, cancel := chromedp.NewContext(allocCtx)
defer cancel()
```

This is useful for debugging (watching the browser act in real time), setting a custom user agent to avoid bot detection, or running in restricted environments where Chrome lives at a non-default path.

## A complete scraping example

Putting it together — scraping a list of headlines:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var headlines []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://news.ycombinator.com`),
		chromedp.WaitVisible(`.athing`),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll(".titleline > a"))
				.map(a => a.textContent)
		`, &headlines),
	)
	if err != nil {
		log.Fatal(err)
	}

	for i, h := range headlines {
		fmt.Printf("%d. %s\n", i+1, h)
	}
}
```

Here `chromedp.Evaluate` returns a JS array directly into a Go `[]string` slice — handy for bulk extraction instead of chaining many individual `Text` calls.

## Working with cookies

chromedp exposes the CDP `Network` and `Storage` domains for cookie access via the `network` and `storage` subpackages.

Reading cookies after a page loads:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var cookies []*network.Cookie
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://example.com`),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range cookies {
		fmt.Printf("%s=%s (domain=%s)\n", c.Name, c.Value, c.Domain)
	}
}
```

Setting a cookie before navigating (useful for injecting an auth session):

```go
err := chromedp.Run(ctx,
	chromedp.ActionFunc(func(ctx context.Context) error {
		return network.SetCookie("session_id", "abc123").
			WithDomain("example.com").
			WithPath("/").
			WithHTTPOnly(true).
			Do(ctx)
	}),
	chromedp.Navigate(`https://example.com/account`),
)
```

## Network interception and monitoring

chromedp can listen to raw CDP events, which is how you tap into network traffic. Register a listener with `chromedp.ListenTarget`.

Logging every request URL and status code:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			fmt.Println("request:", e.Request.Method, e.Request.URL)
		case *network.EventResponseReceived:
			fmt.Println("response:", e.Response.Status, e.Response.URL)
		}
	})

	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(`https://example.com`),
	)
	if err != nil {
		log.Fatal(err)
	}
}
```

Blocking specific requests (e.g. images or ad domains) to speed up scraping:

```go
err := chromedp.Run(ctx,
	network.Enable(),
	network.SetBlockedURLs().WithURLPatterns([]*network.BlockPattern{
		{URLPattern: "*://*:*/*.png", Block: true},
		{URLPattern: "*://*:*/*.jpg", Block: true},
		{URLPattern: "*://*doubleclick.net/*", Block: true},
	}),
	chromedp.Navigate(`https://example.com`),
)
```

`network.SetBlockedURLs` no longer takes a `[]string` of glob patterns — `cdproto` dropped the deprecated `urls` parameter from its generated command in favour of `urlPatterns`, so you now build a slice of `*network.BlockPattern` with `WithURLPatterns`. Pinning the version matters here: the change landed in `cdproto` [`v0.0.0-20260321001828-e3e3800016bc`](https://github.com/chromedp/cdproto/commit/e3e3800016bc8b62edcf5607ca2dfb3880e6a1fe) (2026-03-21), the first version where `SetBlockedURLs` became a zero-argument builder rather than `SetBlockedURLs(urls []string)`. Since **`chromedp` v0.15.1 is the first release to require that `cdproto`**, anyone on v0.15.0 or earlier is still on the old signature and `WithUrls`/`WithURLPatterns` won't compile there. Two details matter here:

- **The pattern must be a proper URLPattern, in absolute form.** A bare glob like `*.png` compiles fine but fails at runtime with `Pattern "*.png" failed to parse as a URLPattern. (-32602)`. Use a full pattern like `*://*:*/*.png` instead.
- **Set `Block: true` explicitly.** It's the zero value's opposite — an omitted `Block` field defaults to `false`, meaning "don't block." Since that fails silently (no error, nothing blocked), it's easy to miss.

Capturing a response body is trickier than it looks. `EventResponseReceived` fires as soon as headers arrive — the body hasn't been buffered by Chrome yet, so calling `network.GetResponseBody` from that handler reliably fails with `No data found for resource with given identifier (-32000)` (see chromedp issues [#1130](https://github.com/chromedp/chromedp/issues/1130), [#1317](https://github.com/chromedp/chromedp/issues/1317), and [#1408](https://github.com/chromedp/chromedp/issues/1408)). The body is only guaranteed to be available once `EventLoadingFinished` fires. The fix is to record the request ID and MIME type on `responseReceived`, then fetch the body when `loadingFinished` arrives for that same request ID. Target listeners are dispatched sequentially from a single event goroutine, so blocking inside one stalls every event queued behind it — the mutex on the map is there not because callbacks overlap, but so the map stays safe once you read it from your own goroutines:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var mu sync.Mutex
	pending := make(map[network.RequestID]string) // requestID -> MIME type

	// A chromedp context only gains a CDP executor once it is inside a
	// chromedp.Run task, so we capture the live task context to make
	// network.GetResponseBody work from the listener goroutine.
	var taskCtx context.Context

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventResponseReceived:
			if e.Response.MimeType == "application/json" {
				mu.Lock()
				pending[e.RequestID] = e.Response.MimeType
				mu.Unlock()
			}

		case *network.EventLoadingFinished:
			mu.Lock()
			_, ok := pending[e.RequestID]
			delete(pending, e.RequestID)
			mu.Unlock()
			if !ok {
				return
			}

			go func(reqID network.RequestID) {
				body, err := network.GetResponseBody(reqID).Do(taskCtx)
				if err != nil {
					log.Println("body fetch failed:", err)
					return
				}
				fmt.Println("JSON body:", string(body))
			}(e.RequestID)
		}
	})

	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(c context.Context) error {
			taskCtx = c
			return nil
		}),
		network.Enable(),
		chromedp.Navigate(`http://localhost:8080/`),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}
}
```

The difference isn't marginal: fetching the body from `EventResponseReceived` fails essentially every time — testing against a page issuing 40 JSON responses gave 0 successes out of 40, even with small (~1 KB) bodies and no artificial delay. Moving the fetch to `EventLoadingFinished` gave 40 out of 40.

## Performance metrics

CDP's `Performance` domain reports metrics like layout count, JS heap size, and timestamps. Enable it, then query metrics at any point:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/chromedp/cdproto/performance"
	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var metrics []*performance.Metric
	err := chromedp.Run(ctx,
		performance.Enable(),
		chromedp.Navigate(`https://example.com`),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			metrics, err = performance.GetMetrics().Do(ctx)
			return err
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, m := range metrics {
		fmt.Printf("%s = %.2f\n", m.Name, m.Value)
	}
}
```

For page-load timing specifically (navigation start, DOM content loaded, load event), pull the Navigation Timing API via `Evaluate`:

```go
var timing map[string]float64
err := chromedp.Run(ctx,
	chromedp.Navigate(`https://example.com`),
	chromedp.Evaluate(`(() => {
		const t = performance.timing;
		return {
			dns: t.domainLookupEnd - t.domainLookupStart,
			connect: t.connectEnd - t.connectStart,
			ttfb: t.responseStart - t.requestStart,
			domContentLoaded: t.domContentLoadedEventEnd - t.navigationStart,
			load: t.loadEventEnd - t.navigationStart,
		};
	})()`, &timing),
)
fmt.Println(timing)
```

## Capturing console messages and JS errors

Console output and uncaught exceptions arrive as CDP runtime events. Listen for `runtime.EventConsoleAPICalled` and `runtime.EventExceptionThrown`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			var args []string
			for _, a := range e.Args {
				args = append(args, string(a.Value))
			}
			fmt.Printf("console.%s: %v\n", e.Type, args)
		case *runtime.EventExceptionThrown:
			fmt.Println("JS error:", e.ExceptionDetails.Error())
		}
	})

	err := chromedp.Run(ctx,
		runtime.Enable(),
		chromedp.Navigate(`https://example.com`),
	)
	if err != nil {
		log.Fatal(err)
	}
}
```

This is especially useful in test pipelines: fail a test if any `EventExceptionThrown` fires, or assert that an expected `console.log` message appeared during a user flow.

## Common gotchas

- **Always set a timeout.** Without one, a stalled page can hang your program indefinitely.
- **Selectors must match visible/ready elements** for actions like `Click` and `SendKeys`; pair them with a `WaitVisible` or `WaitReady` first.
- **Headless vs. headed rendering can differ** slightly (fonts, viewport size). If a site behaves oddly only in headless mode, try running headed to debug.
- **One browser context per goroutine.** Don't share a single `chromedp.Run` context across concurrent tasks — create a new tab context (`chromedp.NewContext(allocCtx)`) per goroutine instead.
- **Respect robots.txt and terms of service** when scraping, and consider rate-limiting requests to avoid overloading target servers.

## When to reach for chromedp

chromedp shines when you need real browser rendering — JavaScript-heavy single-page apps, sites behind client-side auth flows, or anything requiring visual output like screenshots and PDFs. If a site's content is available via plain HTML or a public API, a simpler HTTP client with `net/http` and `goquery` will be faster and lighter-weight. Reach for chromedp when the browser's rendering itself is part of the requirement.

