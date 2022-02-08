package main

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func Test_unzip(t *testing.T) {
	dest, strings, err := unzipFiles(context.Background(), "/Users/prakhar/workspace/prakhar/oci-cloud-function/hello/TEKNISKE_DATA_07.02.2021.zip", nil)
	defer os.RemoveAll(dest)
	t.Log(strings, err)
}

func TestZipAndUnzip(t *testing.T) {

	readFile, err := ioutil.ReadFile("1.zip")
	require.NoError(t, err)
	require.NotNil(t, readFile)

	fileName, err := copyContentAsZip(bytes.NewBuffer(readFile))
	require.NoError(t, err)
	require.NotNil(t, fileName)
	defer os.RemoveAll(fileName)
	t.Log(fileName)

	dst, strings, err := unzipFiles(context.Background(), fileName, nil)
	defer os.RemoveAll(dst)
	t.Log(strings, err)
}
