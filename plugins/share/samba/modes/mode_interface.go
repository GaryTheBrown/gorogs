package modes

const (
	Name string = "Samba"
)

type System interface {
	Setup() error
	NotifyCreate(shareName string, path string) error
	NotifyRemove(shareName string) error
	NotifyCommentUpdate(shareName, comment string) error
}
