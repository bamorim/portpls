package app

import "errors"

var (
	ErrNoFreePorts        = errors.New("no free ports available")
	ErrAllocationNotFound = errors.New("allocation not found")
	ErrMissingFlags       = errors.New("must specify --name or --all")
	ErrConfirmDeclined    = errors.New("cancelled")
	ErrInvalidConfigKey   = errors.New("invalid configuration key")
	ErrInvalidConfigValue = errors.New("invalid configuration value")
	ErrInvalidPortRange   = errors.New("invalid port range")
	ErrUnknownFormat      = errors.New("unknown format")
)

type CodeError struct {
	Code int
	Err  error
}

func (e CodeError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e CodeError) Unwrap() error {
	return e.Err
}

func NewCodeError(code int, err error) error {
	return CodeError{Code: code, Err: err}
}
