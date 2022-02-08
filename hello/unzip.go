package main

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/openzipkin/zipkin-go"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func unzipFiles(ctx context.Context, src string, tt *zipkin.Tracer) (string, []string, error) {
	span, _ := tt.StartSpanFromContext(ctx, "unzipFiles files")
	defer span.Finish()

	dest, err := ioutil.TempDir("", "mvr-*")
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		log.Print("cannot create temp dir", err)
		return "", nil, err
	}

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return "", filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return "", filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return "", filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			span.Tag(string(zipkin.TagError), err.Error())
			return "", filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			span.Tag(string(zipkin.TagError), err.Error())
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
