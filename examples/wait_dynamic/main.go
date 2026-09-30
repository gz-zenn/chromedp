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

	var output string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`http://localhost:8080/dashboard`),
		chromedp.WaitVisible(`#results`, chromedp.ByID),
		chromedp.Text(`#results`, &output),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Results:", output)
}
