package main

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func Test_copyContentAsZip(t *testing.T) {
	var cc []byte
	cBuf := bytes.NewBuffer(cc)
	file, err := os.ReadFile("1.zip")
	require.NoError(t, err)
	require.NotNil(t, file)

	written, err := io.Copy(cBuf, bytes.NewBuffer(file))
	require.NoError(t, err)
	require.NotNil(t, written)

	fileName, err := copyContentAsZip(cBuf)
	require.NoError(t, err)
	require.NotNil(t, fileName)
	defer os.RemoveAll(fileName)
	t.Log(fileName)
}
