package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

var StartReq int64
var EndReq int64
var StartGoroutineReq int64
var EndGoroutineReq int64

type MyHandler struct{}

// The goal of this main function is to simulate a server with APM trace client and APM wrap server
// where request can be in "deadlock" states.
// to reproduce: `go run fakeserver/main.go`
// to reproduce: `go run main.go`
// then do a request one at a times: example:
// for i in {1..1000}; do curl localhost:8082; done
// at some point you can see that a request never return
// you can see log in main server looking like:
// Open request 1 Done request 45 (stay like that)
// Open goroutine req 2912 Done goroutine req 159088 (stay like that)
func main() {
	go func() {
		for {
			// we just print time to time the state of current request
			fmt.Printf("Open request %d Done request %d\n", StartReq-EndReq, EndReq)
			fmt.Printf("Open goroutine req %d Done goroutine req %d\n", StartGoroutineReq-EndGoroutineReq, EndGoroutineReq)
			time.Sleep(time.Second * 2)
		}
	}()
	srv := &http.Server{
		Addr:         ":8082",
		Handler:      apmhttp.Wrap(&MyHandler{}),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	// if this line is remove -> everything is fine
	http.DefaultClient = apmhttp.WrapClient(http.DefaultClient, apmhttp.WithClientTrace())
	defaultTransportPtr := http.DefaultTransport.(*http.Transport)
	defaultTransportPtr.MaxIdleConns = 10
	defaultTransportPtr.MaxIdleConnsPerHost = 10
	log.Fatal(srv.ListenAndServe())
}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&StartReq, 1)
	defer func() {
		atomic.AddInt64(&EndReq, 1)
	}()
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()
	span, reqCtx := apm.StartSpan(ctx, "random", "custom")
	span.Context.SetLabel("user", 1)
	r = r.WithContext(reqCtx)
	defer span.End()

	for a := 0; a < 100; a++ {
		wg := sync.WaitGroup{}
		for i := 0; i < 50; i++ { // only 50 concurrent go routine per request
			wg.Add(1)
			go func() {
				atomic.AddInt64(&StartGoroutineReq, 1)
				defer func() {
					atomic.AddInt64(&EndGoroutineReq, 1)
				}()
				defer wg.Done()
				// I think the bug happen when we timeout
				ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*10)
				defer cancel()
				req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:8081/2", nil)
				http.DefaultClient.Do(req)
			}()
		}
		wg.Wait()
	}
	fmt.Fprint(w, "Hello!")
}
