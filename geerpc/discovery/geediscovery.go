package discovery

import (
	"errors"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

type GeeDiscovery struct {
	//TODO
	//serviceProviders map[string][]providerItem
	serviceProviders map[string][]string
	index            int
	r                *rand.Rand
	mu               sync.RWMutex
	registry         string
	timeout          time.Duration
	lastUpdate       time.Time
}
type providerItem struct {
	addr string
}

var _ Discovery = (*GeeDiscovery)(nil)

const defaultUpdateTimeout = time.Second * 10

func NewGeeRegistryDiscovery(registerAddr string, timeout time.Duration) *GeeDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	d := &GeeDiscovery{
		serviceProviders: make(map[string][]string),
		r:                rand.New(rand.NewSource(time.Now().UnixNano())),
		registry:         registerAddr,
		timeout:          timeout,
	}
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}
func (d *GeeDiscovery) Update(service string, servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.serviceProviders[service] = servers
	d.lastUpdate = time.Now()
	return nil
}
func (d *GeeDiscovery) Refresh(service string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lastUpdate.Add(d.timeout).After(time.Now()) {
		return nil
	}
	log.Println("rpc registry: refresh servers from registry", d.registry)
	// TODO
	resp, err := http.Get(d.registry)
	if err != nil {
		log.Println("rpc registry refresh err:", err)
		return nil
	}
	services := strings.Split(resp.Header.Get("X-Geerpc-Servers"), ",")
	d.serviceProviders[service] = make([]string, 0, len(services))
	for _, service := range services {
		if strings.TrimSpace(service) != "" {
			d.serviceProviders[service] = append(d.serviceProviders[service], strings.TrimSpace(service))
		}
	}
	d.lastUpdate = time.Now()
	return nil
}

func (d *GeeDiscovery) Get(service string, mode SelectMode) (string, error) {
	if err := d.Refresh(service); err != nil {
		return "", err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.serviceProviders[service])
	if n == 0 {
		return "", errors.New("rpc clientdiscovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return d.serviceProviders[service][d.r.Intn(n)], nil
	case RoundRobinSelect:
		s := d.serviceProviders[service][d.index%n] // servers could be updated, so mode n to ensure safety
		d.index = (d.index + 1) % n
		return s, nil
	default:
		return "", errors.New("rpc clientdiscovery: not supported select mode")
	}
}
func (d *GeeDiscovery) GetAll(service string) ([]string, error) {
	if err := d.Refresh(service); err != nil {
		return nil, err
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	// return a copy of d.servers
	servers := make([]string, len(d.serviceProviders[service]), len(d.serviceProviders[service]))
	copy(servers, d.serviceProviders[service])
	return servers, nil
}
