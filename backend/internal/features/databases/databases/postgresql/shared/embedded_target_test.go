package postgresql_shared

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ValidateNotEmbeddedTarget_RejectsEmbeddedLogicalDatabase(t *testing.T) {
	testCases := []struct {
		name string
		host string
	}{
		{name: "temporary socket", host: "/tmp"},
		{name: "private socket", host: "/databasus-data/pgsocket"},
		{name: "localhost", host: "localhost"},
		{name: "loopback range", host: "127.42.0.9"},
		{name: "IPv6 loopback", host: "::1"},
		{name: "IPv4 unspecified", host: "0.0.0.0"},
		{name: "IPv6 unspecified", host: "::"},
		{name: "Docker host alias", host: "host.docker.internal"},
		{name: "Docker bridge", host: "172.17.0.1"},
		{name: "libpq host list with socket", host: "remote.invalid,/tmp"},
		{name: "libpq host list with loopback", host: "remote.invalid,127.0.0.1"},
		{name: "libpq host list with default socket", host: "remote.invalid,"},
		{name: "abbreviated loopback", host: "127.1"},
		{name: "three-part loopback", host: "127.0.1"},
		{name: "decimal loopback", host: "2130706433"},
		{name: "octal loopback", host: "0177.0.0.1"},
		{name: "hexadecimal loopback", host: "0x7f000001"},
		{name: "single-part unspecified", host: "0"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateNotEmbeddedTarget(EmbeddedTargetSpec{
				Host:         testCase.host,
				Port:         5437,
				DatabaseName: "DATABASUS",
			})

			require.ErrorContains(t, err, "backing up Databasus internal database is not allowed")
		})
	}
}

func Test_ValidateNotEmbeddedTarget_AllowsNonEmbeddedLogicalDatabase(t *testing.T) {
	testCases := []EmbeddedTargetSpec{
		{Host: "localhost", Port: 5437, DatabaseName: "application"},
		{Host: "database.example.com", Port: 5437, DatabaseName: "databasus"},
		{
			Host:               "localhost",
			Port:               5437,
			DatabaseName:       "databasus",
			IsSSHTunnelEnabled: true,
			SSHBastionHost:     "bastion.example.com",
		},
	}

	for _, target := range testCases {
		require.NoError(t, ValidateNotEmbeddedTarget(target))
	}
}

func Test_ValidateNotEmbeddedTarget_RejectsLocalBastionAndPhysicalBackup(t *testing.T) {
	require.Error(t, ValidateNotEmbeddedTarget(EmbeddedTargetSpec{
		Host:               "localhost",
		Port:               5437,
		DatabaseName:       "databasus",
		IsSSHTunnelEnabled: true,
		SSHBastionHost:     "127.0.0.1",
	}))

	require.Error(t, ValidateNotEmbeddedTarget(EmbeddedTargetSpec{
		Host:       "/tmp",
		Port:       5437,
		IsPhysical: true,
	}))
}
