package prompt

import (
	"errors"
	"testing"

	"github.com/manifoldco/promptui"
)

func TestIsCancelError(t *testing.T) {
	if !IsCancelError(promptui.ErrInterrupt) {
		t.Fatal("expected ErrInterrupt to be treated as cancel")
	}
	if !IsCancelError(promptui.ErrEOF) {
		t.Fatal("expected ErrEOF to be treated as cancel")
	}
	if !IsCancelError(promptui.ErrAbort) {
		t.Fatal("expected ErrAbort to be treated as cancel")
	}
	if IsCancelError(errors.New("boom")) {
		t.Fatal("unexpected cancel classification for regular error")
	}
	if IsCancelError(nil) {
		t.Fatal("nil error must not be treated as cancel")
	}
}
