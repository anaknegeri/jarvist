package interfaces

// ServerController defines methods needed to control a server
type ServerController interface {
	Start(port int) error
	Stop() error
}

// ServiceComponent defines a component that can be started and stopped
type ServiceComponent interface {
	Start() error
	Stop() error
}
