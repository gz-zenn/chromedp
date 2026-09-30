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
		chromedp.Navigate(`http://localhost:8080/`),
	)
	if err != nil {
		log.Fatal(err)
	}
}
