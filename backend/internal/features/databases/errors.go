package databases

import "errors"

var ErrInsufficientPermissionsToTestDatabaseConnection = errors.New(
	"insufficient permissions to test this database connection",
)

var ErrDatabaseConnectionTargetLookup = errors.New("failed to load database connection target")

var ErrDatabaseConnectionAuthorizationLookup = errors.New(
	"failed to verify database connection permissions",
)
