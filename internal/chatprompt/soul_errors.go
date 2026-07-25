package chatprompt

import "errors"

var (
	errSoulEmpty    = errors.New("SOUL cannot be empty")
	errSoulTooLarge = errors.New("SOUL exceeds size limit")
)

// ErrSoulEmpty reports an empty SOUL save attempt.
func ErrSoulEmpty() error { return errSoulEmpty }

// ErrSoulTooLarge reports SOUL content over SoulMaxBytes.
func ErrSoulTooLarge() error { return errSoulTooLarge }
