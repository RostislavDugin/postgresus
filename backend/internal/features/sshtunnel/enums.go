package sshtunnel

type AuthType string

const (
	AuthTypePassword   AuthType = "PASSWORD"
	AuthTypePrivateKey AuthType = "PRIVATE_KEY"
)
