package errors

import "errors"

var (
	ErrImageNotFound  = errors.New("image does not exists")
	ErrEmptyImageName = errors.New("image name is empty")
)
