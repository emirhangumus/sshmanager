package prompt

import (
	"errors"
	"io"
	"testing"
)

func TestIsCancelError(t *testing.T) {
	if !IsCancelError(ErrCancelled) {
		t.Fatal("expected ErrCancelled to be treated as cancel")
	}
	if !IsCancelError(io.EOF) {
		t.Fatal("expected EOF to be treated as cancel")
	}
	if IsCancelError(errors.New("boom")) {
		t.Fatal("unexpected cancel classification for regular error")
	}
	if IsCancelError(nil) {
		t.Fatal("nil error must not be treated as cancel")
	}
}
