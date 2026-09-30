package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // show the browser window
		chromedp.Flag("disable-gpu", true),
		chromedp.UserAgent("MyCustomAgent/1.0"),
		// chromedp.ExecPath("/usr/bin/chromium-browser"), // custom Chrome path
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var title string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`http://localhost:8080/`),
		chromedp.Title(&title),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Page title:", title)
}
