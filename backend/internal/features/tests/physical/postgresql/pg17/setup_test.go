package pg17

import (
	"os"
	"testing"

	physicaltesting "databasus-backend/internal/features/tests/physical/postgresql/shared"
)

func TestMain(m *testing.M) {
	teardown := physicaltesting.Setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}
