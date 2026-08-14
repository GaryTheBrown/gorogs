package shares

type StorageShare interface {
	Setup() error
	Start() error
	Healthcheck() error
	IsCritical() bool
	Stop() error
}
