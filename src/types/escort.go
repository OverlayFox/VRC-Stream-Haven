package types

type Escort interface {
	Connection
	GetLocation() Location
}
