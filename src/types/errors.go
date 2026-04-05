package types

import "errors"

var (
	ErrBufferNotReady = errors.New("buffer not ready")

	ErrHavenNotFound = errors.New("haven not found")

	ErrPublisherAlreadyExists = errors.New("flagship already exists")
	ErrPublisherNotFound      = errors.New("flagship not found")

	ErrEscortNotFound      = errors.New("escort not found")
	ErrEscortsNotAvailable = errors.New("no escorts available")
)
