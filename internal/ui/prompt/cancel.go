package prompt

import (
	"errors"
	"io"
	"os"
)

var ErrCancelled = errors.New("prompt cancelled")

// IsCancelError returns true when an error means the user cancelled input.
func IsCancelError(err error) bool {
	return errors.Is(err, ErrCancelled) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, os.ErrClosed)
}
