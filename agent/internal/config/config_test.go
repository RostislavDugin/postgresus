package config

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LoadFromJSONAndArgs_ValuesLoadedFromJSON(t *testing.T) {
	dir := setupTempDir(t)
	writeConfigJSON(t, dir, Config{
		DatabasusHost: "http://json-host:4005",
		DbID:          "json-db-id",
		Token:         "json-token",
	})

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{})

	assert.Equal(t, "http://json-host:4005", cfg.DatabasusHost)
	assert.Equal(t, "json-db-id", cfg.DbID)
	assert.Equal(t, "json-token", cfg.Token)
}

func Test_LoadFromJSONAndArgs_ValuesLoadedFromArgs_WhenNoJSON(t *testing.T) {
	setupTempDir(t)

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{
		"--databasus-host", "http://arg-host:4005",
		"--db-id", "arg-db-id",
		"--token", "arg-token",
	})

	assert.Equal(t, "http://arg-host:4005", cfg.DatabasusHost)
	assert.Equal(t, "arg-db-id", cfg.DbID)
	assert.Equal(t, "arg-token", cfg.Token)
}

func Test_LoadFromJSONAndArgs_ArgsOverrideJSON(t *testing.T) {
	dir := setupTempDir(t)
	writeConfigJSON(t, dir, Config{
		DatabasusHost: "http://json-host:4005",
		DbID:          "json-db-id",
		Token:         "json-token",
	})

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{
		"--databasus-host", "http://arg-host:9999",
		"--db-id", "arg-db-id-override",
		"--token", "arg-token-override",
	})

	assert.Equal(t, "http://arg-host:9999", cfg.DatabasusHost)
	assert.Equal(t, "arg-db-id-override", cfg.DbID)
	assert.Equal(t, "arg-token-override", cfg.Token)
}

func Test_LoadFromJSONAndArgs_PartialArgsOverrideJSON(t *testing.T) {
	dir := setupTempDir(t)
	writeConfigJSON(t, dir, Config{
		DatabasusHost: "http://json-host:4005",
		DbID:          "json-db-id",
		Token:         "json-token",
	})

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{
		"--databasus-host", "http://arg-host-only:4005",
	})

	assert.Equal(t, "http://arg-host-only:4005", cfg.DatabasusHost)
	assert.Equal(t, "json-db-id", cfg.DbID)
	assert.Equal(t, "json-token", cfg.Token)
}

func Test_SaveToJSON_ConfigSavedCorrectly(t *testing.T) {
	setupTempDir(t)

	deleteWal := true
	cfg := &Config{
		DatabasusHost:          "http://save-host:4005",
		DbID:                   "save-db-id",
		Token:                  "save-token",
		IsDeleteWalAfterUpload: &deleteWal,
	}

	err := cfg.SaveToJSON()
	require.NoError(t, err)

	saved := readConfigJSON(t)

	assert.Equal(t, "http://save-host:4005", saved.DatabasusHost)
	assert.Equal(t, "save-db-id", saved.DbID)
	assert.Equal(t, "save-token", saved.Token)
}

func Test_SaveToJSON_AfterArgsOverrideJSON_SavedFileContainsMergedValues(t *testing.T) {
	dir := setupTempDir(t)
	writeConfigJSON(t, dir, Config{
		DatabasusHost: "http://json-host:4005",
		DbID:          "json-db-id",
		Token:         "json-token",
	})

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{
		"--databasus-host", "http://override-host:9999",
	})

	err := cfg.SaveToJSON()
	require.NoError(t, err)

	saved := readConfigJSON(t)

	assert.Equal(t, "http://override-host:9999", saved.DatabasusHost)
	assert.Equal(t, "json-db-id", saved.DbID)
	assert.Equal(t, "json-token", saved.Token)
}

func Test_LoadFromJSONAndArgs_PgFieldsLoadedFromJSON(t *testing.T) {
	dir := setupTempDir(t)
	deleteWal := false
	writeConfigJSON(t, dir, Config{
		DatabasusHost:          "http://json-host:4005",
		DbID:                   "json-db-id",
		Token:                  "json-token",
		PgHost:                 "pg-json-host",
		PgPort:                 5433,
		PgUser:                 "pg-json-user",
		PgPassword:             "pg-json-pass",
		PgType:                 "docker",
		PgHostBinDir:           "/usr/bin",
		PgDockerContainerName:  "pg-container",
		PgWalDir:               "/opt/wal",
		IsDeleteWalAfterUpload: &deleteWal,
	})

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{})

	assert.Equal(t, "pg-json-host", cfg.PgHost)
	assert.Equal(t, 5433, cfg.PgPort)
	assert.Equal(t, "pg-json-user", cfg.PgUser)
	assert.Equal(t, "pg-json-pass", cfg.PgPassword)
	assert.Equal(t, "docker", cfg.PgType)
	assert.Equal(t, "/usr/bin", cfg.PgHostBinDir)
	assert.Equal(t, "pg-container", cfg.PgDockerContainerName)
	assert.Equal(t, "/opt/wal", cfg.PgWalDir)
	assert.Equal(t, false, *cfg.IsDeleteWalAfterUpload)
}

func Test_LoadFromJSONAndArgs_PgFieldsLoadedFromArgs(t *testing.T) {
	setupTempDir(t)

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{
		"--pg-host", "arg-pg-host",
		"--pg-port", "5433",
		"--pg-user", "arg-pg-user",
		"--pg-password", "arg-pg-pass",
		"--pg-type", "docker",
		"--pg-host-bin-dir", "/custom/bin",
		"--pg-docker-container-name", "my-pg",
		"--pg-wal-dir", "/var/wal",
	})

	assert.Equal(t, "arg-pg-host", cfg.PgHost)
	assert.Equal(t, 5433, cfg.PgPort)
	assert.Equal(t, "arg-pg-user", cfg.PgUser)
	assert.Equal(t, "arg-pg-pass", cfg.PgPassword)
	assert.Equal(t, "docker", cfg.PgType)
	assert.Equal(t, "/custom/bin", cfg.PgHostBinDir)
	assert.Equal(t, "my-pg", cfg.PgDockerContainerName)
	assert.Equal(t, "/var/wal", cfg.PgWalDir)
}

func Test_LoadFromJSONAndArgs_PgArgsOverrideJSON(t *testing.T) {
	dir := setupTempDir(t)
	writeConfigJSON(t, dir, Config{
		PgHost:   "json-host",
		PgPort:   5432,
		PgUser:   "json-user",
		PgType:   "host",
		PgWalDir: "/json/wal",
	})

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{
		"--pg-host", "arg-host",
		"--pg-port", "5433",
		"--pg-user", "arg-user",
		"--pg-type", "docker",
		"--pg-docker-container-name", "my-container",
		"--pg-wal-dir", "/arg/wal",
	})

	assert.Equal(t, "arg-host", cfg.PgHost)
	assert.Equal(t, 5433, cfg.PgPort)
	assert.Equal(t, "arg-user", cfg.PgUser)
	assert.Equal(t, "docker", cfg.PgType)
	assert.Equal(t, "my-container", cfg.PgDockerContainerName)
	assert.Equal(t, "/arg/wal", cfg.PgWalDir)
}

func Test_LoadFromJSONAndArgs_DefaultsApplied_WhenNoJSONAndNoArgs(t *testing.T) {
	setupTempDir(t)

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{})

	assert.Equal(t, 5432, cfg.PgPort)
	assert.Equal(t, "host", cfg.PgType)
	require.NotNil(t, cfg.IsDeleteWalAfterUpload)
	assert.Equal(t, true, *cfg.IsDeleteWalAfterUpload)
	assert.Equal(t, false, cfg.InsecureSkipVerify)
}

func Test_LoadFromJSONAndArgs_InsecureSkipVerifyLoadedFromJSON(t *testing.T) {
	dir := setupTempDir(t)
	writeConfigJSON(t, dir, Config{
		DatabasusHost:      "https://json-host:4005",
		DbID:               "id",
		Token:              "token",
		InsecureSkipVerify: true,
	})

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{})

	assert.Equal(t, true, cfg.InsecureSkipVerify)
}

func Test_LoadFromJSONAndArgs_InsecureSkipVerifyFromArgsOverridesJSON(t *testing.T) {
	dir := setupTempDir(t)
	writeConfigJSON(t, dir, Config{
		DatabasusHost:      "https://json-host:4005",
		DbID:               "id",
		Token:              "token",
		InsecureSkipVerify: false,
	})

	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.LoadFromJSONAndArgs(fs, []string{"-insecure-skip-verify", "true"})

	assert.Equal(t, true, cfg.InsecureSkipVerify)
}

func Test_SaveToJSON_PgFieldsSavedCorrectly(t *testing.T) {
	setupTempDir(t)

	deleteWal := false
	cfg := &Config{
		DatabasusHost:          "http://host:4005",
		DbID:                   "db-id",
		Token:                  "token",
		PgHost:                 "pg-host",
		PgPort:                 5433,
		PgUser:                 "pg-user",
		PgPassword:             "pg-pass",
		PgType:                 "docker",
		PgHostBinDir:           "/usr/bin",
		PgDockerContainerName:  "pg-container",
		PgWalDir:               "/opt/wal",
		IsDeleteWalAfterUpload: &deleteWal,
	}

	err := cfg.SaveToJSON()
	require.NoError(t, err)

	saved := readConfigJSON(t)

	assert.Equal(t, "pg-host", saved.PgHost)
	assert.Equal(t, 5433, saved.PgPort)
	assert.Equal(t, "pg-user", saved.PgUser)
	assert.Equal(t, "pg-pass", saved.PgPassword)
	assert.Equal(t, "docker", saved.PgType)
	assert.Equal(t, "/usr/bin", saved.PgHostBinDir)
	assert.Equal(t, "pg-container", saved.PgDockerContainerName)
	assert.Equal(t, "/opt/wal", saved.PgWalDir)
	require.NotNil(t, saved.IsDeleteWalAfterUpload)
	assert.Equal(t, false, *saved.IsDeleteWalAfterUpload)
}

func setupTempDir(t *testing.T) string {
	t.Helper()

	origDir, err := os.Getwd()
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	t.Cleanup(func() { os.Chdir(origDir) })

	return dir
}

func writeConfigJSON(t *testing.T, dir string, cfg Config) {
	t.Helper()

	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(dir+"/"+configFileName, data, 0o644))
}

func readConfigJSON(t *testing.T) Config {
	t.Helper()

	data, err := os.ReadFile(configFileName)
	require.NoError(t, err)

	var cfg Config
	require.NoError(t, json.Unmarshal(data, &cfg))

	return cfg
}
