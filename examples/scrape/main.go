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
		chromedp.Navigate(`http://localhost:8080/news`),
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
