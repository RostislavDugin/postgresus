package usecases_postgresql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const publicSchemaClashStderr = `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;


pg_restore: warning: errors ignored on restore: 1
`

func Test_IsPreexistingPublicSchemaOnly_WhenOnlyPublicSchemaError_ReturnsTrue(t *testing.T) {
	assert.True(t, IsPreexistingPublicSchemaOnly(publicSchemaClashStderr))
}

func Test_IsPreexistingPublicSchemaOnly_WhenMissingExtensionAlsoIgnored_ReturnsFalse(t *testing.T) {
	mixed := `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;
pg_restore: error: could not execute query: ERROR:  extension "postgis" is not available
Command was: CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;
pg_restore: warning: errors ignored on restore: 2`

	assert.False(t, IsPreexistingPublicSchemaOnly(mixed),
		"a user-facing restore that silently dropped an extension is still a failed restore")
}

func Test_IsPreexistingPublicSchemaOnly_WhenVisibleErrorCountBelowMarker_ReturnsFalse(t *testing.T) {
	truncated := `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;
pg_restore: warning: errors ignored on restore: 4`

	assert.False(t, IsPreexistingPublicSchemaOnly(truncated),
		"stderr that cannot account for all N errors must not be tolerated")
}

func Test_IsPreexistingPublicSchemaOnly_WhenNoIgnoredErrorsMarker_ReturnsFalse(t *testing.T) {
	aborted := `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;
pg_restore: error: aborting because of errors`

	assert.False(t, IsPreexistingPublicSchemaOnly(aborted))
}

func Test_IsPreexistingPublicSchemaOnly_WhenEmpty_ReturnsFalse(t *testing.T) {
	assert.False(t, IsPreexistingPublicSchemaOnly(""))
}

func Test_IsPreexistingPublicSchemaOnly_WhenRelationAlreadyExists_ReturnsFalse(t *testing.T) {
	relationClash := `pg_restore: error: could not execute query: ERROR:  relation "public.t_a" already exists
Command was: CREATE TABLE public.t_a (id integer NOT NULL);
pg_restore: warning: errors ignored on restore: 1`

	assert.False(t, IsPreexistingPublicSchemaOnly(relationClash))
}
