package helper

import (
	"bytes"
	"context"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func Test_copyContentAsZip(t *testing.T) {
	var cc []byte
	cBuf := bytes.NewBuffer(cc)
	file, err := os.ReadFile("../testdata/1.zip")
	require.NoError(t, err)
	require.NotNil(t, file)

	written, err := io.Copy(cBuf, bytes.NewBuffer(file))
	require.NoError(t, err)
	require.NotNil(t, written)

	tr, err := zipkin.NewTracer(reporter.NewNoopReporter(), zipkin.WithNoopTracer(true))
	require.NoError(t, err)

	fileName, err := SaveObjectAsZip(context.Background(), cBuf, tr)
	require.NoError(t, err)
	require.NotNil(t, fileName)
	defer os.RemoveAll(fileName)
	t.Log(fileName)
}

func Test_unzip(t *testing.T) {
	tr, err := zipkin.NewTracer(reporter.NewNoopReporter(), zipkin.WithNoopTracer(true))
	require.NoError(t, err)

	dest, strings, err := UnzipUploadedFile(context.Background(),
		"../testdata/1.zip", tr)
	require.NoError(t, err)
	require.NotNil(t, strings)
	require.NotEmpty(t, dest)

	defer os.RemoveAll(dest)
	t.Log(strings, err)
}

func TestZipAndUnzip(t *testing.T) {

	readFile, err := ioutil.ReadFile("../testdata/1.zip")
	require.NoError(t, err)
	require.NotNil(t, readFile)

	tr, err := zipkin.NewTracer(reporter.NewNoopReporter(), zipkin.WithNoopTracer(true))
	require.NoError(t, err)

	fileName, err := SaveObjectAsZip(context.Background(), bytes.NewBuffer(readFile), tr)
	require.NoError(t, err)
	require.NotNil(t, fileName)
	defer os.RemoveAll(fileName)
	t.Log(fileName)

	dst, strings, err := UnzipUploadedFile(context.Background(), fileName, tr)
	require.NoError(t, err)
	require.NotNil(t, strings)
	require.NotEmpty(t, dst)
	defer os.RemoveAll(dst)
	t.Log(strings, err)
}

func Test_parseDataFile(t *testing.T) {
	t.SkipNow()
	count, err := parseDataFile(context.Background(), "/Users/prakhar/workspace/toyota/docs/MVR_TECH_DATA.txt.20210603")
	require.NoError(t, err)
	t.Log(count)
	// 10843080
}
