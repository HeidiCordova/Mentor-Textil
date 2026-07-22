package domain

import "errors"

var (
	ErrMissingDeviceID    = errors.New("device_id is required")
	ErrMissingTimestamp   = errors.New("timestamp is required")
	ErrInvalidStopType    = errors.New("invalid stop_type")
	ErrStopNotFound       = errors.New("stop not found")
	ErrCommandNotFound    = errors.New("command not found")
	ErrDuplicateCommand   = errors.New("command already exists (idempotency_key)")
	ErrConfigNotFound     = errors.New("config not found for device")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrServiceUnavailable = errors.New("internal service unavailable")
)
