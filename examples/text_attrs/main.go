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

	var heading string
	var href string

	err := chromedp.Run(ctx,
		chromedp.Navigate(`http://localhost:8080/`),
		chromedp.Text(`h1`, &heading, chromedp.NodeVisible),
		chromedp.AttributeValue(`a`, "href", &href, nil),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Heading:", heading)
	fmt.Println("First link href:", href)
}
