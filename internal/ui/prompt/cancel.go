package prompt

import (
	"errors"

	"github.com/manifoldco/promptui"
)

// IsCancelError returns true when an error means the user cancelled input.
func IsCancelError(err error) bool {
	return errors.Is(err, promptui.ErrInterrupt) ||
		errors.Is(err, promptui.ErrEOF) ||
		errors.Is(err, promptui.ErrAbort)
}
