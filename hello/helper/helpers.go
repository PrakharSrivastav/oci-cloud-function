package helper

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/PrakharSrivastav/oci-cloud-function/model"
	"github.com/openzipkin/zipkin-go"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func UnzipUploadedFile(ctx context.Context, src string, tt *zipkin.Tracer) (string, []string, error) {
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

func SaveObjectAsZip(ctx context.Context, cBuf *bytes.Buffer, tracer *zipkin.Tracer) (string, error) {
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

func ValidateCloudEvent(ctx context.Context, in io.Reader, tracer *zipkin.Tracer) (*model.BucketEvent, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "validateCloudEvent")
	defer span.Finish()

	var bb []byte
	bbuf := bytes.NewBuffer(bb)

	if _, err := io.Copy(bbuf, in); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}

	event := model.BucketEvent{}
	if err := json.Unmarshal(bbuf.Bytes(), &event); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}

	if err := event.Validate(); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), "validation error 1")
		span.Annotate(time.Now(), "validation error 2")
		span.Annotate(time.Now(), "validation error 3")
		span.Annotate(time.Now(), "validation error 4")

		return nil, err
	}
	return &event, nil
}

func parseDataFile(ctx context.Context, path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	wg := new(sync.WaitGroup)

	fileBuffer := bufio.NewReader(file)
	count := 0
	batch := make([]string, 0, 100000)
	line := ""

	totalcount := 0

	for {
		line, err = fileBuffer.ReadString('\n')
		if err != nil {
			break
		}
		batch = append(batch, line)
		if len(batch) != 100000 {
			continue
		}
		count++
		wg.Add(1)
		go handle(batch, count, wg)
		totalcount = totalcount + len(batch)
		//handle2(batch, count)
		batch = make([]string, 0, 100000)
	}
	if len(batch) > 0 {
		wg.Add(1)
		go handle(batch, count, wg)
		totalcount = totalcount + len(batch)
		//handle2(batch, count)
	}

	wg.Wait()

	if err != nil {
		if err == io.EOF {
			log.Printf("totalcount : %d", totalcount)
			log.Printf("globalcount : %d", globalcount)
			return totalcount, nil
		}
		return 0, err
	}
	return totalcount, nil
}

func handle(list []string, num int, wg *sync.WaitGroup) {
	log.Print("batch number ", num)
	globalcount = globalcount + len(list)
	time.Sleep(10 * time.Second)
	wg.Done()
}

var globalcount = 0

func handle2(list []string, num int) {
	log.Print("batch2 number ", num)
	globalcount = globalcount + len(list)
	time.Sleep(5 * time.Second)
}
