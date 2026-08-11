package domain

import "errors"

var (
	ErrInvalidROI       = errors.New("invalid ROI dimensions")
	ErrInvalidThreshold = errors.New("threshold must be between 0 and 1")
	ErrInvalidFSM       = errors.New("invalid FSM parameters")
	ErrInvalidMode      = errors.New("mode must be textil")
	ErrConfigNotFound   = errors.New("configuration not found")
	ErrInvalidOEE       = errors.New("invalid OEE parameters")
	ErrInvalidCloud     = errors.New("invalid cloud parameters")
)
