package flags

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

// Existing command tests use an in-memory keyring, never the developer's OS secrets.
func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}
