package registry

type Register interface {
	Register(service string) error
	StartRegister() error
}
