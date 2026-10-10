package container

import (
	"testing"

	"github.com/moby/moby/api/types/system"
	"github.com/stretchr/testify/assert"
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
