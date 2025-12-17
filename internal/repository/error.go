package repository

import "errors"

var ConflictErr = errors.New("conflict")
var InternalErr = errors.New("internal")
var NotExistsErr = errors.New("not exists")
