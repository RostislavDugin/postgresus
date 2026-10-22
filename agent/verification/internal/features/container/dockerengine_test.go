package container

import (
	"net/netip"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-verification-agent/internal/testutil"
)

func Test_ToHostCapabilities_ReadsCpuLimitSupportFromBothCfsFlags(t *testing.T) {
	testCases := []struct {
		name                      string
		cpuCfsPeriod              bool
		cpuCfsQuota               bool
		isCpuLimitSupportedOnHost bool
	}{
		{name: "both flags set", cpuCfsPeriod: true, cpuCfsQuota: true, isCpuLimitSupportedOnHost: true},
		{name: "period missing", cpuCfsPeriod: false, cpuCfsQuota: true, isCpuLimitSupportedOnHost: false},
		{name: "quota missing", cpuCfsPeriod: true, cpuCfsQuota: false, isCpuLimitSupportedOnHost: false},
		{name: "both flags missing", cpuCfsPeriod: false, cpuCfsQuota: false, isCpuLimitSupportedOnHost: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			capabilities := toHostCapabilities(system.Info{
				CPUCfsPeriod: testCase.cpuCfsPeriod,
				CPUCfsQuota:  testCase.cpuCfsQuota,
			})

			assert.Equal(t, testCase.isCpuLimitSupportedOnHost, capabilities.IsCpuLimitSupported)
		})
	}
}

func Test_ToHostCapabilities_ReadsUserNSRemapFromSecurityOptions(t *testing.T) {
	testCases := []struct {
		name                       string
		securityOptions            []string
		isUserNSRemapEnabledOnHost bool
	}{
		{
			name:                       "userns option present",
			securityOptions:            []string{"name=seccomp,profile=builtin", "name=userns"},
			isUserNSRemapEnabledOnHost: true,
		},
		{
			name:                       "userns option absent",
			securityOptions:            []string{"name=seccomp,profile=builtin"},
			isUserNSRemapEnabledOnHost: false,
		},
		{
			name:                       "no security options",
			securityOptions:            nil,
			isUserNSRemapEnabledOnHost: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			capabilities := toHostCapabilities(system.Info{SecurityOptions: testCase.securityOptions})

			assert.Equal(t, testCase.isUserNSRemapEnabledOnHost, capabilities.IsUserNSRemapEnabled)
		})
	}
}

func Test_ToContainerConfigs_CarriesHardeningAndLimitsOfSpec(t *testing.T) {
	testCases := []struct {
		name                string
		isCpuLimitSupported bool
		expectedNanoCPUs    int64
	}{
		{name: "cpu limit supported", isCpuLimitSupported: true, expectedNanoCPUs: 2_000_000_000},
		{name: "cpu limit unsupported", isCpuLimitSupported: false, expectedNanoCPUs: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			containerManager := NewManager(nil, "agent-1", "databasus/verification-postgres", testutil.DiscardLogger())
			containerManager.isCpuLimitSupported = testCase.isCpuLimitSupported
			spec := containerManager.buildSpec(newTestSpawnPlan())

			config, hostConfig := toContainerConfigs(spec)

			assert.Equal(t, []string{"ALL"}, hostConfig.CapDrop)
			assert.Empty(t, hostConfig.CapAdd)
			assert.Empty(t, hostConfig.SecurityOpt)
			assert.Empty(t, hostConfig.Binds)
			assert.Empty(t, hostConfig.Mounts)
			assert.Equal(t, int64(1024)*1024*1024, hostConfig.Memory)
			require.NotNil(t, hostConfig.PidsLimit)
			assert.EqualValues(t, containerPidsLimit, *hostConfig.PidsLimit)
			assert.Equal(t, testCase.expectedNanoCPUs, hostConfig.NanoCPUs)
			assert.Equal(t, container.NetworkMode("net-id"), hostConfig.NetworkMode)

			assert.Equal(t, network.PortMap{
				pgPublishedPort: []network.PortBinding{
					{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "0"},
				},
			}, hostConfig.PortBindings)
			assert.Equal(t, network.PortSet{pgPublishedPort: struct{}{}}, config.ExposedPorts)
			assert.Equal(t, spec.Image, config.Image)
			assert.Equal(t, spec.Labels, config.Labels)
			assert.Equal(t, "postgres", config.User)
		})
	}
}

func Test_ToExecOptions_CarriesCommandAndNamesNoUser(t *testing.T) {
	testCases := []struct {
		name            string
		isStdinAttached bool
	}{
		{name: "with stdin", isStdinAttached: true},
		{name: "without stdin", isStdinAttached: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cmd := []string{"pg_restore", "--dbname", "restore"}
			env := []string{"PGPASSWORD=pw"}

			options := toExecOptions(cmd, env, testCase.isStdinAttached)

			assert.Equal(t, cmd, options.Cmd)
			assert.Equal(t, env, options.Env)
			assert.Equal(t, testCase.isStdinAttached, options.AttachStdin)
			assert.True(t, options.AttachStdout)
			assert.True(t, options.AttachStderr)
			assert.Empty(t, options.User, "the command runs as the container's unprivileged user")
		})
	}
}
