package helper

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

//
//import (
//	"bytes"
//	"context"
//	"github.com/openzipkin/zipkin-go"
//	"github.com/openzipkin/zipkin-go/reporter"
//	"github.com/stretchr/testify/require"
//	"io"
//	"io/ioutil"
//	"os"
//	"testing"
//)
//
//func Test_copyContentAsZip(t *testing.T) {
//	var cc []byte
//	cBuf := bytes.NewBuffer(cc)
//	file, err := os.ReadFile("../testdata/2.zip")
//	require.NoError(t, err)
//	require.NotNil(t, file)
//
//	written, err := io.Copy(cBuf, bytes.NewBuffer(file))
//	require.NoError(t, err)
//	require.NotNil(t, written)
//
//	tr, err := zipkin.NewTracer(reporter.NewNoopReporter(), zipkin.WithNoopTracer(true))
//	require.NoError(t, err)
//
//	fileName, err := SaveObjectAsZip(context.Background(), cBuf, tr)
//	require.NoError(t, err)
//	require.NotNil(t, fileName)
//	defer os.RemoveAll(fileName)
//	t.Log(fileName)
//}
//
//func Test_unzip(t *testing.T) {
//	tr, err := zipkin.NewTracer(reporter.NewNoopReporter(), zipkin.WithNoopTracer(true))
//	require.NoError(t, err)
//
//	dest, strings, err := UnzipUploadedFile(context.Background(),
//		"../testdata/2.zip", tr)
//	require.NoError(t, err)
//	require.NotNil(t, strings)
//	require.NotEmpty(t, dest)
//
//	defer os.RemoveAll(dest)
//	t.Log(strings, err)
//}
//
//func TestZipAndUnzip(t *testing.T) {
//
//	readFile, err := ioutil.ReadFile("../testdata/2.zip")
//	require.NoError(t, err)
//	require.NotNil(t, readFile)
//
//	tr, err := zipkin.NewTracer(reporter.NewNoopReporter(), zipkin.WithNoopTracer(true))
//	require.NoError(t, err)
//
//	fileName, err := SaveObjectAsZip(context.Background(), bytes.NewBuffer(readFile), tr)
//	require.NoError(t, err)
//	require.NotNil(t, fileName)
//	defer os.RemoveAll(fileName)
//	t.Log(fileName)
//
//	dst, strings, err := UnzipUploadedFile(context.Background(), fileName, tr)
//	require.NoError(t, err)
//	require.NotNil(t, strings)
//	require.NotEmpty(t, dst)
//	defer os.RemoveAll(dst)
//	t.Log(strings, err)
//}
//
//func Test_parseDataFile(t *testing.T) {
//	//t.SkipNow()
//	//count, err := parseDataFile(context.Background(), "/Users/prakhar/workspace/toyota/docs/MVR_TECH_DATA.txt.20210603")
//	count, err := parseDataFile(context.Background(), "/Users/prakhar/yo1.txt")
//	require.NoError(t, err)
//	t.Log(count)
//	// 10843080
//}

func TestSomething(t *testing.T) {
	fmt.Println([]rune("Ø"))
	_, paths, err := unzipUploadedFile("/Users/prakhar/workspace/prakhar/oci-cloud-function/hello/testdata/3.zip")
	//file, err := os.Open("/Users/prakhar/workspace/prakhar/oci-cloud-function/hello/testdata/MVR_TECH_DATA.txt")
	require.NoError(t, err)
	path := ""
	for i := range paths {
		if strings.Contains(paths[i], "MVR_TECH_DATA") {
			path = paths[i]
			break
		}
	}
	require.NotEmpty(t, path)

	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	count := 0
	buf := bufio.NewReader(file)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			break
		}

		//if !utf8.ValidString(line) {
		//	str := toUtf8([]byte(line))
		//	fmt.Println(str)
		//}
		rr := []rune(line)
		for i := 0; i < len(rr); i++ {
			if utf8.RuneLen(rr[i]) == 3 || utf8.RuneLen(rr[i]) == 2 {
				ss := string(rr[i])
				if ss != "Ø" && ss != "ø" && ss != "Æ" && ss != "æ" && ss != "Å" && ss != "å" && ss != "Ä" && ss != "ä" && ss != "Ö" && ss != "ö" {
					rr[i] = ';'
				}
			}
			//fmt.Printf("Rune %v is '%c'  %v \n", i, rr[i], utf8.RuneLen(rr[i]))
		}

		ll := string(rr)
		ss := strings.Split(ll, ";")
		if len(ss) != 85 {
			count++
			//fmt.Println(len(ss), "skipping")
			fmt.Println(ll)
		}

		//fmt.Println(string(rr))
		//fmt.Println(len("�"), utf8.RuneCountInString(line))
		//fmt.Println(strings.Split(strings.TrimSpace(line), "�"))
	}

	fmt.Println(count)
}

func toUtf8(iso88591Buf []byte) string {
	var buf = bytes.NewBuffer(make([]byte, len(iso88591Buf)*4))
	for _, b := range iso88591Buf {
		r := rune(b)
		buf.WriteRune(r)
	}
	return string(buf.Bytes())
}

func unzipUploadedFile(src string) (string, []string, error) {

	dest, err := ioutil.TempDir("", "mvr-*")
	if err != nil {
		return "", nil, err
	}

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return "", filenames, err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return "", filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return "", filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return "", filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return "", filenames, err
		}
	}
	return dest, filenames, nil
}
