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

	var url string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`http://localhost:8080/login`),
		chromedp.SendKeys(`#username`, "myuser"),
		chromedp.SendKeys(`#password`, "mypassword"),
		chromedp.Click(`#submit`, chromedp.NodeVisible),
		chromedp.WaitVisible(`#dashboard`),
		chromedp.Location(&url),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("After login, at:", url)
}
