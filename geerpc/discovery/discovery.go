package discovery

type SelectMode int

const (
	RandomSelect     SelectMode = iota // select randomly
	RoundRobinSelect                   // select using Robbin algorithm
)

type Discovery interface {
	Refresh(service string) error
	//TODO
	Update(service string, servers []string) error
	Get(service string, mode SelectMode) (string, error)
	GetAll(service string) ([]string, error)
}
