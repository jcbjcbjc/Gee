package main

import (
	"Gee/geeregistry"
	"sync"
)

func main() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go geeregistry.StartRegistry("9999", &wg)
	wg.Wait()
}
