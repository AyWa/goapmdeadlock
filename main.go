package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

var StartReq int64
var EndReq int64
var StartGoroutineReq int64
var EndGoroutineReq int64

func main() {
	go func() {
		for {
			// we just print time to time the state of current request
			fmt.Printf("Open request %d Done request %d\n", StartReq-EndReq, EndReq)
			fmt.Printf("Open goroutine req %d Done goroutine req %d\n", StartGoroutineReq-EndGoroutineReq, EndGoroutineReq)
			time.Sleep(time.Second * 2)
		}
	}()

	r := gin.Default()
	srv := &http.Server{
		Addr:         ":8082",
		Handler:      apmhttp.Wrap(r),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	http.DefaultClient = apmhttp.WrapClient(http.DefaultClient, apmhttp.WithClientTrace())
	defaultTransportPtr := http.DefaultTransport.(*http.Transport)
	defaultTransportPtr.MaxIdleConns = 10
	defaultTransportPtr.MaxIdleConnsPerHost = 10

	r.GET("/", func(c *gin.Context) {
		atomic.AddInt64(&StartReq, 1)
		defer func() {
			atomic.AddInt64(&EndReq, 1)
		}()
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second*5)
		defer cancel()
		span, reqCtx := apm.StartSpan(ctx, "random", "custom")
		span.Context.SetLabel("user", 1)
		c.Request = c.Request.WithContext(reqCtx)
		defer span.End()

		wg := sync.WaitGroup{}
		for i := 0; i < 3000; i++ {
			wg.Add(1)
			go func() {
				atomic.AddInt64(&StartGoroutineReq, 1)
				defer func() {
					atomic.AddInt64(&EndGoroutineReq, 1)
				}()
				defer wg.Done()
				ctx, cancel := context.WithTimeout(reqCtx, time.Millisecond*500)
				defer cancel()
				req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:8081/2", nil)
				http.DefaultClient.Do(req)
				if span := apm.SpanFromContext(ctx); span != nil {
					traceID := span.TraceContext().Trace
					fmt.Println(traceID)
				}
			}()
		}
		wg.Wait()
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	log.Fatal(srv.ListenAndServe())
}
