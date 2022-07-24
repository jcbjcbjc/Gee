package gtree

type GTree struct {
	root *GNode
	//mu   sync.Mutex
}

func (g *GTree) createRoot(name string) error {
	return nil
}

func (g *GTree) createService(root string, name string) error {
	return nil
}
func (g *GTree) removeService(root string, name string) error {
	return nil
}
func (g *GTree) AddProvider(path string) error {
	return nil
}
func (g *GTree) RemoveProvider(path string) error {
	return nil
}
func (g *GTree) GetProviders(path string) ([]string, error) {
	return nil, nil

}

func NewGTree() *GTree {
	root := &GNode{
		data:  "/",
		child: make(map[string]*GNode),
		path:  "/",
	}
	return &GTree{root: root}
}

func (g *GTree) find(path string) (*GNode, error) {
	return nil, nil
}
