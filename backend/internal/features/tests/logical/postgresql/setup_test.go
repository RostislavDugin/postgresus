package postgresql_logical

import (
	"os"
	"testing"

	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
)

func TestMain(m *testing.M) {
	teardown := logicaltesting.Setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}
