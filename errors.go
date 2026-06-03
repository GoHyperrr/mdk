package mdk

import "errors"

var (
	ErrModuleNotFound    = errors.New("mdk: module not found")
	ErrWorkflowNotFound  = errors.New("mdk: workflow not found")
	ErrRunNotFound       = errors.New("mdk: run not found")
	ErrAlreadyRegistered = errors.New("mdk: already registered")
)
