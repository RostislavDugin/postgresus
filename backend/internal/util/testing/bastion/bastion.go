// Package bastion builds the SSH tunnel configuration that reaches a bastioned test topology.
//
// It sits apart from containers because containers must not import features: sshtunnel's own tests
// live in package sshtunnel and import containers, so the reverse edge would be an import cycle.
// Putting it in sshtunnel instead would pull testcontainers into the production build graph.
package bastion

import (
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/testing/containers"
)

func GetTunnelConfig(bastionedDatabase containers.BastionedDatabase) sshtunnel.Config {
	return sshtunnel.Config{
		IsEnabled: true,
		Host:      bastionedDatabase.Bastion.Host,
		Port:      bastionedDatabase.Bastion.Port,
		Username:  containers.SshBastionUsername,
		AuthType:  sshtunnel.AuthTypePassword,
		Password:  containers.SshBastionPassword,
	}
}
