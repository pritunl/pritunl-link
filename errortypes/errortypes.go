package errortypes

import (
	"github.com/dropbox/godropbox/errors"
)

type ParseError struct {
	errors.DropboxError
}

type ReadError struct {
	errors.DropboxError
}

type WriteError struct {
	errors.DropboxError
}

type UnknownError struct {
	errors.DropboxError
}

type ExecError struct {
	errors.DropboxError
}

type RequestError struct {
	errors.DropboxError
}
