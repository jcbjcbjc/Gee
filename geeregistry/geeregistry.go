package geeregistry

import (
	. "github.com/jcbjcbjc/Gee/geeregistry/gtree"
	"github.com/jcbjcbjc/Gee/geeweb"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GeeRegistry is a simple register center, provide following functions.
// add a server and receive heartbeat to keep it alive.
// returns all alive servers and delete dead servers sync simultaneously.
type GeeRegistry struct {
	gTree   *GTree
	timeout time.Duration
	mu      sync.Mutex // protect following

}

const (
	defaultPath    = "/_geerpc_/registry"
	DefaultTimeout = time.Minute * 5
	defaultRoot    = "Gee"
)

func New(timeout time.Duration) *GeeRegistry {
	return &GeeRegistry{
		gTree: NewGTree(),

		timeout: timeout,
	}
}

var DefaultGeeRegister = New(DefaultTimeout)

func (r *GeeRegistry) addServiceProvider(groupPath, service string, addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.gTree.AddProvider(groupPath, service, addr); err != nil {
		log.Println(err)
	}
}
func (r *GeeRegistry) aliveServers(groupPath, service string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	providers, err := r.gTree.GetProviders(groupPath, service)
	if err != nil {
		log.Println(err)
	}
	return providers
}

/*func (r *GeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		service := req.Header.Get("Service")
		// keep it simple, server is in req.Header
		w.Header().Set("X-Geerpc-Servers", strings.Join(r.aliveServers(service), ","))

	case "POST":
		service := req.Header.Get("Service")
		addr := req.Header.Get("X-Geerpc-serverapi")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println("GeeRegistry Post:addService" + service + addr)

		//TODO
		r.addService(service, addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}*/

func StartRegistry(addr string, wg *sync.WaitGroup) {
	r := geeweb.Default()
	r.GET(defaultPath, geeweb.HandlerFunc(func(c *geeweb.Context) {
		service := c.Req.Header.Get("Service")
		// keep it simple, server is in req.Header
		c.Writer.Header().Set("X-Geerpc-Servers", strings.Join(DefaultGeeRegister.aliveServers("", service), ","))
	}))

	r.POST(defaultPath, geeweb.HandlerFunc(func(c *geeweb.Context) {
		service := c.Req.Header.Get("Service")
		addr := c.Req.Header.Get("X-Geerpc-serverapi")
		if addr == "" {
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println("GeeRegistry Post:addService" + service + addr)

		//TODO
		DefaultGeeRegister.addServiceProvider("", service, addr)
	}))

	r.Run(addr)
	wg.Done()
}
