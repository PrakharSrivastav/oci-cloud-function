package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/PrakharSrivastav/oci-cloud-function/hello/helper"
	"github.com/PrakharSrivastav/oci-cloud-function/infrastructure"
	fdk "github.com/fnproject/fdk-go"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	fdk.Handle(fdk.HandlerFunc(myHandler))
}

type Message struct {
	Msg string `json:"message"`
}

func myHandler(ctx context.Context, in io.Reader, out io.Writer) {
	zipkinReporter, tracer, span, err := infrastructure.GetSpanWithTracerAndReporter(ctx, "hello-fn")
	if err != nil {
		log.Fatal(err)
	}
	defer func(reporter reporter.Reporter) { _ = reporter.Close() }(zipkinReporter)
	defer span.Finish()
	ctx = zipkin.NewContext(ctx, span)

	// validate event
	cloudEvent, err := helper.ValidateCloudEvent(ctx, in, tracer)
	if err != nil {
		log.Print("invalid cloud event error : ", err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
		span.Tag(string(zipkin.TagError), err.Error())
		return
	}

	// get client
	storageClient, err := infrastructure.NewStorageClient(ctx, tracer)
	if err != nil {
		log.Print("bucket storageClient error : ", err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
		return
	}

	// read object from the bucket
	//cloudEvent.Data.ResourceName = "zzz" // introduce an error here
	objectResponse, err := helper.DownloadObjectFromBucket(ctx, storageClient, cloudEvent, tracer)
	if err != nil {
		log.Print("get objectResponse error :", err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
		return
	}
	defer func(Content io.ReadCloser) { _ = Content.Close() }(objectResponse.Content)

	log.Printf("objectResponse details are : %+v", objectResponse)
	if *objectResponse.ContentLength == 0 {
		log.Print("empty storage file error")
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, `{"error":"empty storage file error"}`)
		return
	}

	var cc []byte
	cBuf := bytes.NewBuffer(cc)
	func() {
		span, _ := tracer.StartSpanFromContext(ctx, "copyObjToBuffer")
		defer span.Finish()
		if _, err = io.Copy(cBuf, objectResponse.Content); err != nil {
			log.Print("read object bytes error", err)
			fdk.WriteStatus(out, http.StatusBadRequest)
			fdk.SetHeader(out, "Content-Type", "application/json")
			_, _ = io.WriteString(out, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
			return
		}
	}()

	// save object as local zip file
	fileName, err := helper.SaveObjectAsZip(ctx, cBuf, tracer)
	if err != nil {
		log.Print("save object as zip error ", err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
		return
	}
	defer func(path string) { _ = os.RemoveAll(path) }(fileName)

	// unzip local file
	dest, str, err := helper.UnzipUploadedFile(ctx, fileName, tracer)
	if err != nil {
		log.Print("zip file extraction error ", fileName, err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
		return
	}
	defer func(path string) { _ = os.RemoveAll(path) }(dest)

	log.Print("unzipped : \n", strings.Join(str, "\n"))

	msg := Message{Msg: "Hello World"}
	if err = json.NewEncoder(out).Encode(&msg); err != nil {
		log.Print("response error ", fileName, err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
		return
	}
}
