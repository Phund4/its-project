package postgres

import "errors"

var (
	ErrFailedToOpenDB    = errors.New("failed to open database")
	ErrFailedToPingDB    = errors.New("failed to ping database")
	ErrFailedToCloseDB   = errors.New("failed to close database")
	ErrFailedToQueryDB   = errors.New("failed to query database")
	ErrFailedToScanDB    = errors.New("failed to scan database")
	ErrFailedToExecDB    = errors.New("failed to exec database")
	ErrFailedToPrepareDB = errors.New("failed to prepare database")
)
