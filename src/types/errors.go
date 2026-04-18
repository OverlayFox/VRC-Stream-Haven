package types

import "errors"

var (
	ErrBufferNotReady = errors.New("buffer not ready")

	ErrHavenNotFound = errors.New("haven not found")

	ErrPublisherAlreadyExists = errors.New("publisher already exists")
	ErrPublisherNotFound      = errors.New("publisher not found")

	ErrEscortNotFound      = errors.New("escort not found")
	ErrEscortsNotAvailable = errors.New("no escorts available")

	ErrViewerNotFound = errors.New("viewer not found")
)
