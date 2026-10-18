package nas_storage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/containers"
)

func startNasStorage(t *testing.T) *NASStorage {
	t.Helper()

	endpoint := containers.StartSamba(t)

	return &NASStorage{
		StorageID: uuid.New(),
		Host:      endpoint.Host,
		Port:      endpoint.Port,
		Share:     containers.SambaShare,
		Username:  containers.SambaUsername,
		Password:  containers.SambaPassword,
		Path:      "cleanup-" + uuid.New().String(),
	}
}

// A non-empty directory is the one refusal the provider's own API can set up.
func Test_DeleteFile_WhenTheServerRefusesTheRemoval_ReturnsTheError(t *testing.T) {
	storage := startNasStorage(t)
	encryptor := encryption.GetFieldEncryptor()
	directory := "refused-" + uuid.New().String()

	session, err := storage.dialSession(t.Context(), encryptor)
	require.NoError(t, err)
	t.Cleanup(func() { closeSession(session) })

	require.NoError(t, session.getShare(t.Context()).MkdirAll(storage.getFilePath(directory), 0o755))

	require.NoError(t, storage.SaveFile(
		t.Context(), encryptor, logger.GetLogger(), directory+"/artifact", strings.NewReader("payload"),
	))

	err = storage.DeleteFile(t.Context(), encryptor, logger.GetLogger(), directory)

	assert.Error(t, err, "a removal the server refuses must not count as a completed deletion")
}

// The configured path is a directory that was never created, so this also covers a
// removal whose parent is missing.
func Test_DeleteFile_WhenFileIsAbsent_Succeeds(t *testing.T) {
	storage := startNasStorage(t)

	err := storage.DeleteFile(
		t.Context(), encryption.GetFieldEncryptor(), logger.GetLogger(), "never-written-"+uuid.New().String(),
	)

	assert.NoError(t, err)
}
