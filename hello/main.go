package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PrakharSrivastav/oci-cloud-function/infrastructure"
	"github.com/PrakharSrivastav/oci-cloud-function/model"
	"github.com/PrakharSrivastav/oci-cloud-function/store"
	fdk "github.com/fnproject/fdk-go"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter"
	"github.com/oracle/oci-go-sdk/v56/objectstorage"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	fdk.Handle(fdk.HandlerFunc(myHandler))
}

type Message struct {
	Msg string `json:"message"`
}

type dependencies struct {
	reporter      reporter.Reporter
	tracer        *zipkin.Tracer
	rootSpan      zipkin.Span
	serviceName   string
	storageClient *objectstorage.ObjectStorageClient
	conn          *store.Client
	ctx           context.Context
}

func (d *dependencies) init(ctx context.Context) error {
	var err error
	d.reporter, d.tracer, d.rootSpan, err = infrastructure.GetSpanWithTracerAndReporter(ctx, "mvr-file-transfer:processes")
	if err != nil {
		return err
	}

	d.ctx = zipkin.NewContext(ctx, d.rootSpan)
	d.storageClient, err = infrastructure.NewStorageClient(d.ctx, d.tracer)
	if err != nil {
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		d.rootSpan.Finish()
		d.reporter.Close()
		return err
	}

	d.conn, err = store.GetConnection()
	if err != nil {
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		d.rootSpan.Finish()
		d.reporter.Close()
	}
	return nil
}

func (d *dependencies) close() {
	d.rootSpan.Finish()
	d.conn.Close()
	d.reporter.Close()
}

func myHandler(ctx context.Context, in io.Reader, out io.Writer) {
	d := new(dependencies)
	err := d.init(ctx)
	if err != nil {
		log.Printf("deps.init.error : %v", err)
		return
	}
	defer d.close()

	event, err := d.validateCloudEvent(in) // validate event
	if err != nil {
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		d.rootSpan.Annotate(time.Now(), fmt.Sprintf("event.validation.error : %v", err))
		log.Printf("event.validation.error : %v", err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, d.errorMessage(err))
		return
	}

	sch, _ := d.conn.GetScheduledJobByIDAndName(d.ctx, 1, "DownloadFromBucket")
	objectResponse, err := d.downloadObjFromBkt(event) // download from bucket
	if err != nil {
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		d.rootSpan.Annotate(time.Now(), fmt.Sprintf("download.object.error : %v", err))
		log.Print("download.object.error :", err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, d.errorMessage(err))
		return
	}
	defer func(Content io.ReadCloser) { _ = Content.Close() }(objectResponse.Content)

	if *objectResponse.ContentLength == 0 { // empty file content
		log.Print("empty.file.error")
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		d.rootSpan.Annotate(time.Now(), "empty.file.error")
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, `{"error":"empty.file.error"}`)
		return
	}

	fileName, err := d.saveObjectAsZip(objectResponse.Content) // save zip-file
	if err != nil {
		d.conn.UpdateScheduledJobStatus(d.ctx, sch.ID, "Error", err.Error())
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		d.rootSpan.Annotate(time.Now(), fmt.Sprintf("save.zip.file.error : %v", err))
		log.Print("save.zip.file.error :", err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, d.errorMessage(err))
		return
	}
	defer func(path string) { _ = os.RemoveAll(path) }(fileName)
	d.conn.UpdateScheduledJobStatus(d.ctx, sch.ID, "Complete", "OK")

	sch, _ = d.conn.GetScheduledJobByIDAndName(d.ctx, 1, "UnzipData")
	dest, str, err := d.unzipUploadedFile(fileName) // unzip local file
	if err != nil {
		d.conn.UpdateScheduledJobStatus(d.ctx, sch.ID, "Error", err.Error())
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		d.rootSpan.Annotate(time.Now(), fmt.Sprintf("save.zip.file.error : %v", err))
		log.Print("zip.file.extraction.error ", fileName, err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, d.errorMessage(err))
		return
	}
	defer func(path string) { _ = os.RemoveAll(path) }(dest)
	d.conn.UpdateScheduledJobStatus(d.ctx, sch.ID, "Complete", "OK")

	sch, _ = d.conn.GetScheduledJobByIDAndName(d.ctx, 1, "WriteToDatabase")
	_, err = d.parseDataFile(str)
	if err != nil {
		d.conn.UpdateScheduledJobStatus(d.ctx, sch.ID, "Error", err.Error())
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		d.rootSpan.Annotate(time.Now(), fmt.Sprintf("parseDataFile : %v", err))
		log.Print("parseDataFile ", err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, d.errorMessage(err))
		return
	}
	d.conn.UpdateScheduledJobStatus(d.ctx, sch.ID, "Complete", "OK")

	msg := Message{Msg: "Hello World"}
	if err = json.NewEncoder(out).Encode(&msg); err != nil {
		log.Print("response error ", fileName, err)
		fdk.WriteStatus(out, http.StatusBadRequest)
		fdk.SetHeader(out, "Content-Type", "application/json")
		_, _ = io.WriteString(out, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
		return
	}
	sch, _ = d.conn.GetScheduledJobByIDAndName(d.ctx, 1, "Overall")
	d.conn.UpdateScheduledJobStatus(d.ctx, sch.ID, "Complete", "OK")

}

func (d *dependencies) errorMessage(err error) string {
	return fmt.Sprintf(`{"error": "%s"}`, err.Error())
}

func (d *dependencies) validateCloudEvent(in io.Reader) (*model.BucketEvent, error) {
	span, _ := d.tracer.StartSpanFromContext(d.ctx, "cloud.event.validation")
	defer span.Finish()

	bb := new(bytes.Buffer)
	if _, err := io.Copy(bb, in); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("read.request.error : %v", err))
		return nil, err
	}

	event := model.BucketEvent{}
	if err := json.Unmarshal(bb.Bytes(), &event); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("unmarshal.request.error : %v", err))
		return nil, err
	}

	if err := event.Validate(); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("event.validation.error : %v", err))
		return nil, err
	}

	return &event, nil
}

func (d *dependencies) downloadObjFromBkt(event *model.BucketEvent) (*objectstorage.GetObjectResponse, error) {
	span, ctx := d.tracer.StartSpanFromContext(d.ctx, "download.bucket.object")
	defer span.Finish()

	request := objectstorage.GetObjectRequest{
		NamespaceName: &event.Data.AdditionalDetails.Namespace,
		BucketName:    &event.Data.AdditionalDetails.BucketName,
		ObjectName:    &event.Data.ResourceName,
	}

	object, err := d.storageClient.GetObject(ctx, request)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("download.bucket.object.error : %v", err))
		return nil, err
	}

	return &object, nil
}

func (d *dependencies) saveObjectAsZip(content io.ReadCloser) (string, error) {
	span, _ := d.tracer.StartSpanFromContext(d.ctx, "save.downloaded.object")
	defer span.Finish()

	cBuf := new(bytes.Buffer)
	if _, err := io.Copy(cBuf, content); err != nil {
		log.Println("save.downloaded.object.error", err)
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("save.downloaded.object.error : %v", err))
		return "", err
	}

	zipFile, err := ioutil.TempFile("", "mvr-*.zip")
	if err != nil {
		log.Println("temp.dir.creation.error", err)
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("temp.dir.creation.error : %v", err))
		return "", err
	}
	defer zipFile.Close()
	if _, err = io.Copy(zipFile, cBuf); err != nil {
		log.Println("save.zip.file.error", err)
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("save.zip.file.error : %v", err))
		return "", err
	}

	return zipFile.Name(), nil
}

func (d *dependencies) unzipUploadedFile(src string) (string, []string, error) {
	span, _ := d.tracer.StartSpanFromContext(d.ctx, "unzip.downloaded.file")
	defer span.Finish()

	dest, err := ioutil.TempDir("", "mvr-*")
	if err != nil {
		log.Print("temp.dir.create.error", err)
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("temp.dir.create.error : %v", err))
		return "", nil, err
	}

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		log.Print("open.zip.file.error : ", err)
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("open.zip.file.error : %v", err))
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

func (d *dependencies) parseDataFile(paths []string) (int, error) {
	span, _ := d.tracer.StartSpanFromContext(d.ctx, "parse.and.save.to.db")
	defer span.Finish()
	path := ""
	for i := range paths {
		if strings.Contains(paths[i], "MVR_TECH_DATA") {
			path = paths[i]
			break
		}
	}

	if path == "" {
		err := errors.New("no files to parse")
		log.Print("no files to parse", err)
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("no files to parse : %v", err))
		return 0, err
	}
	batchSize := 400

	file, err := os.Open(path)
	if err != nil {
		log.Print("could not open source file", err)
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("could not open source file : %v", err))
		return 0, err
	}
	defer file.Close()

	fileBuffer := bufio.NewReader(file)
	batch := make([]string, 0, batchSize)
	line := ""
	concurrentGoroutines := make(chan struct{}, 10)
	count := 0
	wg := new(sync.WaitGroup)
	for {
		line, err = fileBuffer.ReadString('\n')
		if err != nil {
			break
		}
		batch = append(batch, line)
		if len(batch) != batchSize {
			continue
		}
		count++
		wg.Add(1)
		go d.conn.SaveBatch(batch, wg, concurrentGoroutines)
		batch = make([]string, 0, batchSize)
	}

	if len(batch) > 0 {
		wg.Add(1)
		go d.conn.SaveBatch(batch, wg, concurrentGoroutines)
	}

	wg.Wait()
	if err != nil {
		if err == io.EOF {
			return 1, nil
		}
		log.Print("error.parsing.file", err)
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), fmt.Sprintf("error.parsing.file : %v", err))
		return 0, err
	}
	return 1, nil
}
