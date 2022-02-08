package main

import (
	"bytes"
	"context"
	"github.com/openzipkin/zipkin-go"
	"io"
	"io/ioutil"
	"log"
)

func copyContentAsZip(ctx context.Context, cBuf *bytes.Buffer, tracer *zipkin.Tracer) (string, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "save-bucket-obj-as-file")
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
