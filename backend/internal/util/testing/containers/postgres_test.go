package containers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParsePostgresMajorVersion_WhenTagIncludesPrereleaseSuffix_ReturnsMajor(t *testing.T) {
	assert.Equal(t, 19, ParsePostgresMajorVersion("postgres:19beta2"))
}
