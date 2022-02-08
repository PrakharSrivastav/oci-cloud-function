package main

import (
	"bytes"
	"context"
	"encoding/json"
	fdk "github.com/fnproject/fdk-go"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter"
	"io"
	"log"
	"os"
	"strings"
)

const endpointUrl = ""

func main() {
	fdk.Handle(fdk.HandlerFunc(myHandler))
}

func myHandler(ctx context.Context, in io.Reader, out io.Writer) {
	zipkinReporter, tracer, span, err := getSpanWithTracerAndReporter(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func(reporter reporter.Reporter) { _ = reporter.Close() }(zipkinReporter)
	defer span.Finish()
	ctx = zipkin.NewContext(ctx, span)

	// validate event
	cloudEvent, err := validateCloudEvent(ctx, in, tracer)
	if err != nil {
		log.Print("invalid cloud event error : ", err)
		return
	}

	// get client
	storageClient, err := objectStorageClient(ctx, tracer)
	if err != nil {
		log.Print("bucket storageClient error : ", err)
		return
	}

	// read object from the bucket
	//cloudEvent.Data.ResourceName = "zzz" // introduce an error here
	objectResponse, err := downloadObjectFromBucket(ctx, storageClient, cloudEvent, tracer)
	if err != nil {
		log.Print("get objectResponse error :", err)
		return
	}
	defer func(Content io.ReadCloser) { _ = Content.Close() }(objectResponse.Content)

	log.Printf("objectResponse details are : %+v", objectResponse)
	if *objectResponse.ContentLength == 0 {
		log.Print("empty storage file error")
		return
	}

	var cc []byte
	cBuf := bytes.NewBuffer(cc)
	if _, err = io.Copy(cBuf, objectResponse.Content); err != nil {
		log.Print("read object bytes error", err)
		return
	}

	// save object as local zip file
	fileName, err := saveObjectAsZip(ctx, cBuf, tracer)
	if err != nil {
		log.Print("save object as zip error ", err)
		return
	}
	defer func(path string) { _ = os.RemoveAll(path) }(fileName)

	// unzip local file
	dest, str, err := unzipUploadedFile(ctx, fileName, tracer)
	if err != nil {
		log.Print("zip file extraction error ", fileName, err)
		return
	}
	defer func(path string) { _ = os.RemoveAll(path) }(dest)

	log.Print("unzipped : \n", strings.Join(str, "\n"))

	msg := Message{Msg: "Hello World"}
	if err = json.NewEncoder(out).Encode(&msg); err != nil {
		log.Print("response error ", fileName, err)
		return
	}
}
