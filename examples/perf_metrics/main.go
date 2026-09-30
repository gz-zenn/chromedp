package main

import (
	"context"
	"fmt"
	"log"

	"github.com/chromedp/cdproto/performance"
	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var metrics []*performance.Metric
	err := chromedp.Run(ctx,
		performance.Enable(),
		chromedp.Navigate(`http://localhost:8080/`),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			metrics, err = performance.GetMetrics().Do(ctx)
			return err
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, m := range metrics {
		fmt.Printf("%s = %.2f\n", m.Name, m.Value)
	}
}
