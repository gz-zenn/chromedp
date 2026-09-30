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

	ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var timing map[string]float64
	err := chromedp.Run(ctx,
		chromedp.Navigate(`http://localhost:8080/`),
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
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(timing)
}
