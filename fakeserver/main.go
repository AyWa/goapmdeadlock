package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var StartReq int64
var EndReq int64

func main() {
	go func() {
		for {
			// we just print time to time the state of current request
			fmt.Printf("Open request %d Done request %d\n", StartReq-EndReq, EndReq)
			time.Sleep(time.Second * 2)
		}
	}()
	http.HandleFunc("/2", HelloServer)
	http.ListenAndServe(":8081", nil)
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&StartReq, 1)
	defer func() {
		atomic.AddInt64(&EndReq, 1)
	}()
	time.Sleep(time.Second * 2)
	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}
