package config

import (
	"bufio"
	"flag"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadConfig(t *testing.T, args ...string) *Config {
	t.Helper()

	cfg := &Config{}
	cfg.LoadFromJSONAndArgs(flag.NewFlagSet("test", flag.ContinueOnError), args)

	return cfg
}

var allFlagNames = []string{
	"databasus-host", "agent-id", "token", "max-cpu", "max-ram-mb",
	"max-disk-gb", "max-concurrent-jobs", "allow-insecure-http", "verification-pg-image-repo",
}

func emptyReader() *bufio.Reader {
	return bufio.NewReader(strings.NewReader(""))
}

func Test_Validate_WhenConfigComplete_ReturnsCapacity(t *testing.T) {
	cfg := &Config{
		DatabasusHost:     "https://primary.example:4005",
		AgentID:           "agent-1",
		Token:             "secret",
		MaxCPU:            8,
		MaxRAMMb:          4096,
		MaxDiskGb:         100,
		MaxConcurrentJobs: 4,
	}

	capacity, err := cfg.Validate()

	require.NoError(t, err)
	assert.Equal(t, 2, capacity.CPUPerJob)
	assert.Equal(t, 1024, capacity.RAMMbPerJob)
}

func Test_Validate_WhenRequiredStringMissing_ReturnsError(t *testing.T) {
	for name, cfg := range map[string]*Config{
		"no databasus-host": {AgentID: "a", Token: "t", MaxCPU: 4, MaxRAMMb: 2048, MaxDiskGb: 10, MaxConcurrentJobs: 1},
		"no agent-id":       {DatabasusHost: "https://x:4005", Token: "t", MaxCPU: 4, MaxRAMMb: 2048, MaxDiskGb: 10, MaxConcurrentJobs: 1},
		"no token":          {DatabasusHost: "https://x:4005", AgentID: "a", MaxCPU: 4, MaxRAMMb: 2048, MaxDiskGb: 10, MaxConcurrentJobs: 1},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := cfg.Validate()
			require.Error(t, err)
		})
	}
}

func Test_Validate_WhenStringsSetButCapacityInvalid_ReturnsCapacityError(t *testing.T) {
	cfg := &Config{
		DatabasusHost:     "https://primary.example:4005",
		AgentID:           "agent-1",
		Token:             "secret",
		MaxCPU:            2,
		MaxRAMMb:          4096,
		MaxDiskGb:         100,
		MaxConcurrentJobs: 4,
	}

	_, err := cfg.Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CPU per job")
}

func Test_DeriveCapacity_WhenConfigValid_DerivesPerJobSplit(t *testing.T) {
	cfg := &Config{MaxCPU: 8, MaxRAMMb: 4096, MaxDiskGb: 100, MaxConcurrentJobs: 4}

	capacity, err := cfg.DeriveCapacity()

	require.NoError(t, err)
	assert.Equal(t, 2, capacity.CPUPerJob)
	assert.Equal(t, 1024, capacity.RAMMbPerJob)
	assert.Equal(t, 8, capacity.MaxCPU)
	assert.Equal(t, 4096, capacity.MaxRAMMb)
	assert.Equal(t, 100, capacity.MaxDiskGb)
	assert.Equal(t, 4, capacity.MaxConcurrentJobs)
}

func Test_DeriveCapacity_WhenConcurrentJobsExceedCPU_ReturnsError(t *testing.T) {
	cfg := &Config{MaxCPU: 2, MaxRAMMb: 4096, MaxDiskGb: 100, MaxConcurrentJobs: 4}

	_, err := cfg.DeriveCapacity()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CPU per job")
}

func Test_DeriveCapacity_WhenRAMBelowFloor_ReturnsError(t *testing.T) {
	cfg := &Config{MaxCPU: 4, MaxRAMMb: 256, MaxDiskGb: 100, MaxConcurrentJobs: 1}

	_, err := cfg.DeriveCapacity()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max-ram-mb")
}

func Test_DeriveCapacity_WhenRAMPerJobBelowFloor_ReturnsError(t *testing.T) {
	cfg := &Config{MaxCPU: 8, MaxRAMMb: 1024, MaxDiskGb: 100, MaxConcurrentJobs: 8}

	_, err := cfg.DeriveCapacity()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "per job")
}

func Test_DeriveCapacity_WhenZeroFields_ReturnsError(t *testing.T) {
	for name, cfg := range map[string]*Config{
		"no concurrent jobs": {MaxCPU: 4, MaxRAMMb: 2048, MaxDiskGb: 10, MaxConcurrentJobs: 0},
		"no cpu":             {MaxCPU: 0, MaxRAMMb: 2048, MaxDiskGb: 10, MaxConcurrentJobs: 1},
		"no disk":            {MaxCPU: 4, MaxRAMMb: 2048, MaxDiskGb: 0, MaxConcurrentJobs: 1},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := cfg.DeriveCapacity()
			require.Error(t, err)
		})
	}
}

func Test_ValidateTransport_WhenHTTPS_PassesWithoutPrompt(t *testing.T) {
	cfg := &Config{DatabasusHost: "https://primary.example:4005"}

	err := cfg.ValidateTransport(false, emptyReader())

	require.NoError(t, err)
}

func Test_ValidateTransport_WhenHTTPNonTTYWithoutFlag_FailsNamingBothFixes(t *testing.T) {
	cfg := &Config{DatabasusHost: "http://primary.example:4005"}

	err := cfg.ValidateTransport(false, emptyReader())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "https://")
	assert.Contains(t, err.Error(), "--allow-insecure-http")
}

func Test_ValidateTransport_WhenHTTPWithAllowFlag_PassesWithWarn(t *testing.T) {
	cfg := &Config{DatabasusHost: "http://primary.example:4005", AllowInsecureHTTP: true}

	err := cfg.ValidateTransport(false, emptyReader())

	require.NoError(t, err)
}

func Test_ValidateTransport_WhenHTTPTTYAndOperatorConsents_Passes(t *testing.T) {
	cfg := &Config{DatabasusHost: "http://primary.example:4005"}

	err := cfg.ValidateTransport(true, bufio.NewReader(strings.NewReader("y\n")))

	require.NoError(t, err)
}

func Test_ValidateTransport_WhenHTTPTTYAndOperatorDeclines_ReturnsError(t *testing.T) {
	cfg := &Config{DatabasusHost: "http://primary.example:4005"}

	err := cfg.ValidateTransport(true, bufio.NewReader(strings.NewReader("n\n")))

	require.Error(t, err)
}

func Test_ValidateTransport_WhenSchemeUnsupported_ReturnsError(t *testing.T) {
	cfg := &Config{DatabasusHost: "ftp://primary.example:4005"}

	err := cfg.ValidateTransport(true, emptyReader())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "https://")
}

func Test_MaskSensitive_WhenEmpty_ReturnsNotSet(t *testing.T) {
	assert.Equal(t, "(not set)", maskSensitive(""))
}

func Test_MaskSensitive_WhenValue_RevealsQuarterThenMasks(t *testing.T) {
	assert.Equal(t, "ab***", maskSensitive("abcdefgh"))
}

func Test_GetVerificationPgImageRepo_WhenUnset_ReturnsBundledDefault(t *testing.T) {
	cfg := &Config{}

	assert.Equal(t, DefaultVerificationPgImageRepo, cfg.GetVerificationPgImageRepo())
}

func Test_GetVerificationPgImageRepo_WhenSet_ReturnsConfiguredValue(t *testing.T) {
	cfg := &Config{VerificationPgImageRepo: "registry.internal/pg-verify"}

	assert.Equal(t, "registry.internal/pg-verify", cfg.GetVerificationPgImageRepo())
}

func Test_LoadFromJSONAndArgs_WhenFlagsProvided_OverrideEveryValue(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := loadConfig(t,
		"--databasus-host", "https://flag:4005",
		"--agent-id", "agent-x",
		"--token", "tok",
		"--max-cpu", "8",
		"--max-ram-mb", "4096",
		"--max-disk-gb", "100",
		"--max-concurrent-jobs", "4",
		"--allow-insecure-http",
		"--verification-pg-image-repo", "registry/pg",
	)

	assert.Equal(t, "https://flag:4005", cfg.DatabasusHost)
	assert.Equal(t, "agent-x", cfg.AgentID)
	assert.Equal(t, "tok", cfg.Token)
	assert.Equal(t, 8, cfg.MaxCPU)
	assert.Equal(t, 4096, cfg.MaxRAMMb)
	assert.Equal(t, 100, cfg.MaxDiskGb)
	assert.Equal(t, 4, cfg.MaxConcurrentJobs)
	assert.True(t, cfg.AllowInsecureHTTP)
	assert.Equal(t, "registry/pg", cfg.VerificationPgImageRepo)

	for _, name := range allFlagNames {
		assert.Equal(t, "command line args", cfg.flagSources[name], name)
	}
}

func Test_LoadFromJSONAndArgs_WhenOnlyFilePresent_LoadsFileAndMarksSource(t *testing.T) {
	t.Chdir(t.TempDir())
	source := &Config{DatabasusHost: "https://file:4005", AgentID: "agent-file", MaxCPU: 6}
	require.NoError(t, source.SaveToJSON())

	cfg := loadConfig(t)

	assert.Equal(t, "https://file:4005", cfg.DatabasusHost)
	assert.Equal(t, "agent-file", cfg.AgentID)
	assert.Equal(t, 6, cfg.MaxCPU)
	assert.Equal(t, configFileName, cfg.flagSources["databasus-host"])
}

func Test_LoadFromJSONAndArgs_WhenFlagAndFile_FlagOverridesFile(t *testing.T) {
	t.Chdir(t.TempDir())
	source := &Config{DatabasusHost: "https://file:4005", AgentID: "agent-file"}
	require.NoError(t, source.SaveToJSON())

	cfg := loadConfig(t, "--databasus-host", "https://flag:4005")

	assert.Equal(t, "https://flag:4005", cfg.DatabasusHost, "flag overrides file")
	assert.Equal(t, "agent-file", cfg.AgentID, "untouched field keeps the file value")
	assert.Equal(t, "command line args", cfg.flagSources["databasus-host"])
	assert.Equal(t, configFileName, cfg.flagSources["agent-id"])
}

func Test_LoadFromJSONAndArgs_WhenZeroValuedFlags_DoNotOverrideFile(t *testing.T) {
	t.Chdir(t.TempDir())
	source := &Config{MaxCPU: 8, VerificationPgImageRepo: "registry/pg"}
	require.NoError(t, source.SaveToJSON())

	cfg := loadConfig(t, "--max-cpu", "0", "--verification-pg-image-repo", "")

	assert.Equal(t, 8, cfg.MaxCPU, "an explicit zero flag cannot unset a file value")
	assert.Equal(t, "registry/pg", cfg.VerificationPgImageRepo)
}

func Test_LoadFromJSONAndArgs_WhenAllowInsecureHTTPInFile_StaysTrueWithoutFlag(t *testing.T) {
	t.Chdir(t.TempDir())
	source := &Config{AllowInsecureHTTP: true}
	require.NoError(t, source.SaveToJSON())

	cfg := loadConfig(t)

	assert.True(t, cfg.AllowInsecureHTTP)
	assert.Equal(t, configFileName, cfg.flagSources["allow-insecure-http"])
}

func Test_LoadFromJSONAndArgs_WhenNothingProvided_AllSourcesNotConfigured(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := loadConfig(t)

	for _, name := range allFlagNames {
		assert.Equal(t, "not configured", cfg.flagSources[name], name)
	}
}
