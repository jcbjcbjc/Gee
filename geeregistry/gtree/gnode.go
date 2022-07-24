package gtree

import "time"

type gNodeType int

const (
	Root gNodeType = iota // 开始生成枚举值, 默认为0
	Service
	Type
	URL
)
const (
	Providers = "providers"
	consumers = "consumers"
)

type ProviderItem struct {
	addr  string
	start time.Time
}
type GNode struct {
	data         string
	nodeType     gNodeType
	child        map[string]*GNode
	path         string
	providerItem *ProviderItem
}
