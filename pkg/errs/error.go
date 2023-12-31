package errs

import "errors"

var (
	ErrorInvalidField = errors.New("invalid field")
	ErrorInvalidInput = errors.New("invalid input")
	ErrorNotFound     = errors.New("not found")
	ErrorInternal     = errors.New("internal error")
	ErrorVerification = errors.New("verification error")
	ErrorMismatch     = errors.New("mismatch error")
)
