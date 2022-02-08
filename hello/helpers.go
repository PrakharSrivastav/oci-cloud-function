package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/openzipkin/zipkin-go"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func unzipUploadedFile(ctx context.Context, src string, tt *zipkin.Tracer) (string, []string, error) {
	span, _ := tt.StartSpanFromContext(ctx, "unzipUploadedFile")
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

func saveObjectAsZip(ctx context.Context, cBuf *bytes.Buffer, tracer *zipkin.Tracer) (string, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "saveObjectAsZip")
	defer span.Finish()

	zipFile, err := ioutil.TempFile("", "mvr-*.zip")
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		log.Print("can not create temp dir", err)
		return "", err
	}
	defer zipFile.Close()

	_, err = io.Copy(zipFile, cBuf)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return "", err
	}

	return zipFile.Name(), nil
}

func validateCloudEvent(ctx context.Context, in io.Reader, tt *zipkin.Tracer) (*BucketEvent, error) {
	span, _ := tt.StartSpanFromContext(ctx, "validateCloudEvent")
	defer span.Finish()

	var bb []byte
	bbuf := bytes.NewBuffer(bb)

	if _, err := io.Copy(bbuf, in); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}

	event := BucketEvent{}
	if err := json.Unmarshal(bbuf.Bytes(), &event); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}

	if err := event.validate(); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}
	return &event, nil
}
