package shares

type StorageShare interface {
	Setup() error
	Start() error
	Healthcheck() error
	IsCritical() bool // Tells the supervisor if this component can trigger a container failure
	Stop() error
}
