package gtree

import "time"

type GTree struct {
	root *Ggroup
	//mu   sync.Mutex
}

func (g *GTree) findGroup(groupPath string) (*Ggroup, error) {
	return nil, nil
}
func (g *GTree) AddProvider(groupPath, service string, addr string) error {
	group, err := g.findGroup(groupPath)
	if err != nil {
		return err
	}
	serviceNode := group.findOrAddService(service)
	for _, child := range serviceNode.Providers {
		if child.addr == addr {
			child.start = time.Now()
			return nil
		}
	}
	serviceNode.Providers = append(serviceNode.Providers, &providerItem{addr: addr, start: time.Now()})
	return nil
}
func (g *GTree) RemoveProvider(groupPath, service string, provider *providerItem) error {
	return nil
}
func (g *GTree) GetProviders(groupPath, service string) ([]string, error) {
	return nil, nil
}

func NewGTree() *GTree {
	return &GTree{root: newGroup()}
}
