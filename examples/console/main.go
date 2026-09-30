package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			var args []string
			for _, a := range e.Args {
				args = append(args, string(a.Value))
			}
			fmt.Printf("console.%s: %v\n", e.Type, args)
		case *runtime.EventExceptionThrown:
			fmt.Println("JS error:", e.ExceptionDetails.Error())
		}
	})

	err := chromedp.Run(ctx,
		runtime.Enable(),
		chromedp.Navigate(`http://localhost:8080/console`),
	)
	if err != nil {
		log.Fatal(err)
	}
}
