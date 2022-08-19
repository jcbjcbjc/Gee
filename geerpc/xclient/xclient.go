package xclient

import (
	"context"
	"errors"
	"github.com/jcbjcbjc/Gee/Util/cache/singleflight"
	. "github.com/jcbjcbjc/Gee/geerpc"
	"github.com/jcbjcbjc/Gee/geerpc/discovery"

	"io"
	"reflect"
	"sync"
)

var OnHandlerMap = make(map[string]OnHandler)

func GetOnHandler(serviceMethod string) (OnHandler, error) {
	var err error
	if onHandler, ok := OnHandlerMap[serviceMethod]; ok {
		return onHandler, nil
	}
	err = errors.New("GetOnHandler:lack service")
	return nil, err
}

type XClient struct {
	d       discovery.Discovery
	mode    discovery.SelectMode
	opt     *Option
	mu      sync.Mutex
	clients map[string]*Client

	loader *singleflight.Group
}

var _ io.Closer = (*XClient)(nil)

func NewXClient(d discovery.Discovery, mode discovery.SelectMode, opt *Option) *XClient {
	return &XClient{d: d, mode: mode, opt: opt, clients: make(map[string]*Client), loader: &singleflight.Group{}}
}

func (xc *XClient) Close() error {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	for key, client := range xc.clients {
		// I have no idea how to deal with error, just ignore it.
		_ = client.Close()
		delete(xc.clients, key)
	}
	return nil
}

//TODO add cache
func (xc *XClient) dial(rpcaddr string) (*Client, error) {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	client, ok := xc.clients[rpcaddr]
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(xc.clients, rpcaddr)
		client = nil
	}
	if client == nil {
		var err error
		client, err = XDial(rpcaddr, xc.opt)
		if err != nil {
			return nil, err
		}
		xc.clients[rpcaddr] = client
	}
	return client, nil
}
func (xc *XClient) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}, Async bool) error {
	client, err := xc.dial(rpcAddr)

	if err != nil {
		return err
	}
	return client.Call(ctx, serviceMethod, args, reply, Async)
}

//TODO mod
func RegisterOnHandler(serviceMethod string, f OnHandler) error {
	var err error
	_, ok := OnHandlerMap[serviceMethod]
	if ok {
		return errors.New("sss")
	}
	OnHandlerMap[serviceMethod] = f
	return err
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.
// xc will choose a proper server.

//use singleFlight to ensure for the same call

func (xc *XClient) Call(ctx context.Context, service string, serviceMethod string, args, reply interface{}, Async bool) error {
	rpcAddr, err := xc.d.Get(service, xc.mode)
	if err != nil {
		return err
	}
	return xc.call(rpcAddr, ctx, serviceMethod, args, reply, Async)
}

// Broadcast invokes the named function for every server registered in clientDiscovery
func (xc *XClient) Broadcast(ctx context.Context, service string, serviceMethod string, args, reply interface{}) error {
	servers, err := xc.d.GetAll(service)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex // protect e and replyDone
	var e error
	replyDone := reply == nil // if reply is nil, don't need to set value
	ctx, cancel := context.WithCancel(ctx)
	for _, rpcAddr := range servers {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var clonedReply interface{}
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			err := xc.call(rpcAddr, ctx, serviceMethod, args, clonedReply, false)
			mu.Lock()
			if err != nil && e == nil {
				e = err
				cancel() // if any call failed, cancel unfinished calls
			}
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	return e
}
