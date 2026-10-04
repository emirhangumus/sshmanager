package progress

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type notifyingWriter struct{ lines chan string }

func (w notifyingWriter) Write(data []byte) (int, error) {
	w.lines <- string(data)
	return len(data), nil
}

func TestStagesAndHeartbeatWhileOperationBlocks(t *testing.T) {
	w := notifyingWriter{lines: make(chan string, 20)}
	finish := make(chan struct{})
	result := make(chan error, 1)
	expected := errors.New("keyring unavailable")
	go func() {
		result <- run(w, func(report func(string)) error {
			report("Accessing OS keyring")
			<-finish
			report("Storage change failed")
			return expected
		}, 10*time.Millisecond)
	}()
	defer close(finish)
	for _, expectedText := range []string{"Accessing OS keyring", "Still working: Accessing OS keyring"} {
		select {
		case line := <-w.lines:
			if !strings.Contains(line, expectedText) {
				t.Fatalf("got %q, want %q", line, expectedText)
			}
		case <-time.After(time.Second):
			t.Fatal("progress was not displayed while operation blocked")
		}
	}
	finish <- struct{}{}
	select {
	case err := <-result:
		if !errors.Is(err, expected) {
			t.Fatalf("operation error lost: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("progress worker did not stop")
	}
}
