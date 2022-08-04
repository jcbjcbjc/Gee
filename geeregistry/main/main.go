package main

import (
	. "Gee/geeregistry"
	"net"
	"net/http"
	"sync"
)

func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	HandleHTTP()

	_ = http.Serve(l, nil)
}
func main() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()
}
