package magic

type Store interface {
	Set(magic uint64) error
	Get() (uint64, error)
	Clear() error
}
