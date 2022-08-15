package gtree

import "time"

type Ggroup struct {
	prefix       string
	data         string
	ServiceNodes []*ServiceNode
	Children     []*Ggroup
	parent       *Ggroup
}

func newGroup() *Ggroup {
	return &Ggroup{
		prefix:       "",
		data:         "",
		ServiceNodes: make([]*ServiceNode, 0),
		Children:     make([]*Ggroup, 0),
		parent:       nil,
	}
}

// Group is defined to create a new RouterGroup
// remember all groups share the same Engine instance
func (group *Ggroup) Group(data string) *Ggroup {
	newGroup := &Ggroup{
		prefix:       group.prefix + "/" + data,
		data:         data,
		parent:       group,
		ServiceNodes: make([]*ServiceNode, 0),
		Children:     make([]*Ggroup, 0),
	}
	group.Children = append(group.Children, newGroup)
	return newGroup
}
func (group *Ggroup) findOrAddService(name string) *ServiceNode {
	for _, child := range group.ServiceNodes {
		if child.ServiceName == name {
			return child
		}
	}
	service := newServiceNode(name)
	group.ServiceNodes = append(group.ServiceNodes, service)
	return service
}

type providerItem struct {
	addr  string
	start time.Time
}
type consumerItem struct {
	addr string
}
type ServiceNode struct {
	ServiceName string
	Providers   []*providerItem
	Consumers   []*consumerItem
}

func newServiceNode(name string) *ServiceNode {
	return &ServiceNode{
		ServiceName: name,
		Providers:   make([]*providerItem, 0),
		Consumers:   make([]*consumerItem, 0),
	}
}
