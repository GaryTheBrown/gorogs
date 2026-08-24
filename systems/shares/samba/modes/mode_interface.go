package modes

const (
	Name = "Samba"
)

type System interface {
	Setup() error
	NotifyCreate(shareName string, path string) error
	NotifyRemove(shareName string) error
}
