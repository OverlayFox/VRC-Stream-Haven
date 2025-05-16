package types

type Governor interface {
	AddHaven(Haven)
	RemoveHaven(Haven) error
	GetHaven(string) (Haven, error)
}
