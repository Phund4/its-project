package coordinator

import "errors"

var ErrWorkerStatusTimeoutSecRequired = errors.New("worker status timeout sec must be greater than 0")

// Config controls coordinator domain/service behavior.
type Config struct {
	WorkerStatusTimeoutSec int
}

func (c Config) Validate() error {
	if c.WorkerStatusTimeoutSec <= 0 {
		return ErrWorkerStatusTimeoutSecRequired
	}
	return nil
}
