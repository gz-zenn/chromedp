package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var session string
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return network.SetCookie("session_id", "abc123").
				WithDomain("localhost").
				WithPath("/").
				WithHTTPOnly(true).
				Do(ctx)
		}),
		chromedp.Navigate(`http://localhost:8080/account`),
		chromedp.Text(`#session`, &session, chromedp.NodeVisible),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Account page shows:", session)
}
