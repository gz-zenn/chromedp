package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate(`http://localhost:8080/`),
		chromedp.FullScreenshot(&buf, 90), // 90 = JPEG quality
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("screenshot.jpg", buf, 0644); err != nil {
		log.Fatal(err)
	}
	log.Println("wrote screenshot.jpg", len(buf), "bytes")
}
