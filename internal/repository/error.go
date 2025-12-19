package repository

import (
	"errors"
)

// DB errors
var ErrConflict = errors.New("conflict")
var ErrInternal = errors.New("internal")
var ErrNotExists = errors.New("not exists")

// File errors
