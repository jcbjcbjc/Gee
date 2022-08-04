package registry

import (
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	defaultPath    = "/_geerpc_/registry"
	DefaultTimeout = time.Minute * 5
	defaultRoot    = "Gee"
)

type GeeRegister struct {
	mu              sync.Mutex
	registerService string
	registry        string
	addr            string
	duration        time.Duration
}

func NewGeeRegister(registry, addr string, duration time.Duration) *GeeRegister {
	return &GeeRegister{

		registry: registry,
		addr:     addr,
		duration: duration,
	}
}
func (g *GeeRegister) StartGeeRegister() error {
	return g.heartbeat()
}
func (g *GeeRegister) Register(service string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.registerService = service
	return nil
}

// Heartbeat send a heartbeat message every once in a while
// it's a helper function for a server to register or send heartbeat
func (g *GeeRegister) heartbeat() error {
	if g.duration == 0 {
		// make sure there is enough time to send heart beat
		// before it's removed from registry
		g.duration = DefaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = g.sendHeartbeat()
	go func() {
		t := time.NewTicker(g.duration)
		for err == nil {
			<-t.C
			err = g.sendHeartbeat()
		}
		//TODO  throw exception
	}()
	return err
}

func (g *GeeRegister) sendHeartbeat() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	log.Println(g.addr, "send heart beat to registry", g.registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", g.registry, nil)
	// TODO  //////
	req.Header.Set("Service", g.registerService)
	req.Header.Set("X-Geerpc-serverapi", g.addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
