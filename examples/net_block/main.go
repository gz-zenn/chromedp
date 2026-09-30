package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {
	http.Post(`http://localhost:8080/debug/counters`, "text/plain", nil)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	err := chromedp.Run(ctx,
		network.Enable(),
		network.SetBlockedURLs().WithURLPatterns([]*network.BlockPattern{
			{URLPattern: "*://*:*/*.png", Block: true},
			{URLPattern: "*://*:*/*.jpg", Block: true},
			{URLPattern: "*://*doubleclick.net/*", Block: true},
		}),
		chromedp.Navigate(`http://localhost:8080/`),
		chromedp.WaitVisible(`img`),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}

	var imgWidth int64
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector("img").naturalWidth`, &imgWidth),
	); err != nil {
		log.Fatal(err)
	}

	resp, err := http.Get(`http://localhost:8080/debug/counters`)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if imgWidth == 0 {
		fmt.Println("image blocked (naturalWidth=0)")
	} else {
		fmt.Println("image was NOT blocked (naturalWidth>0)")
	}
	fmt.Println("server-side counter:", string(body))
}
