package port

import "errors"

var ErrNoFreePorts = errors.New("no free ports available")
