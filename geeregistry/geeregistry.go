package registry

import (
	. "geeregistry/gtree"
	"log"
	"net"
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

func (r *GeeRegistry) addService(service string, addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.gTree.AddProvider("/" + defaultRoot + "/" + service + "/" + Providers + "/" + addr); err != nil {
		log.Println(err)
	}
}
func (r *GeeRegistry) aliveServers(service string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	providers, err := r.gTree.GetProviders("/" + defaultRoot + "/" + service + "/" + Providers)
	if err != nil {
		log.Println(err)
	}
	return providers
}

func (r *GeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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
		r.addService(service, addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *GeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(defaultPath, r)
	log.Println("rpc registry path:", registryPath)
}
func HandleHTTP() {
	DefaultGeeRegister.HandleHTTP(defaultPath)
}

func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":8848")
	HandleHTTP()
	wg.Done()
	_ = http.Serve(l, nil)
}
