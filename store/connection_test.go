package store

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetConnection(t *testing.T) {
	connection, err := GetConnection()
	require.NoError(t, err)
	require.NotNil(t, connection)
}
