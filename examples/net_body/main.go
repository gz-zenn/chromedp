package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var mu sync.Mutex
	pending := make(map[network.RequestID]string) // requestID -> MIME type

	// A chromedp context only gains a CDP executor once it is inside a
	// chromedp.Run task, so we capture the live task context to make
	// network.GetResponseBody work from the listener goroutine.
	var taskCtx context.Context

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventResponseReceived:
			if e.Response.MimeType == "application/json" {
				mu.Lock()
				pending[e.RequestID] = e.Response.MimeType
				mu.Unlock()
			}

		case *network.EventLoadingFinished:
			mu.Lock()
			_, ok := pending[e.RequestID]
			delete(pending, e.RequestID)
			mu.Unlock()
			if !ok {
				return
			}

			go func(reqID network.RequestID) {
				body, err := network.GetResponseBody(reqID).Do(taskCtx)
				if err != nil {
					log.Println("body fetch failed:", err)
					return
				}
				fmt.Println("JSON body:", string(body))
			}(e.RequestID)
		}
	})

	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(c context.Context) error {
			taskCtx = c
			return nil
		}),
		network.Enable(),
		chromedp.Navigate(`http://localhost:8080/`),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}
}
