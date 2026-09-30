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

	var result int
	err := chromedp.Run(ctx,
		chromedp.Navigate(`http://localhost:8080/`),
		chromedp.Evaluate(`document.querySelectorAll("a").length`, &result),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Number of links:", result)
}
