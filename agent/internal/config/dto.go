package config

type parsedFlags struct {
	databasusHost         *string
	dbID                  *string
	token                 *string
	pgHost                *string
	pgPort                *int
	pgUser                *string
	pgPassword            *string
	pgType                *string
	pgHostBinDir          *string
	pgDockerContainerName *string
	pgWalDir              *string
	insecureSkipVerify    *bool

	sources map[string]string
}
