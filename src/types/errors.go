package types

import "fmt"

var (
	ErrBufferNotReady = fmt.Errorf("buffer not ready")

	ErrHavenNotFound = fmt.Errorf("haven not found")

	ErrFlagshipAlreadyExists = fmt.Errorf("flagship already exists")
	ErrFlagshipNotFound      = fmt.Errorf("flagship not found")

	ErrEscortNotFound      = fmt.Errorf("escort not found")
	ErrEscortsNotAvailable = fmt.Errorf("no escorts available")
)
