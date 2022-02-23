package main

import (
	"context"
	"fmt"
	"github.com/PrakharSrivastav/oci-cloud-function/infrastructure"
	"github.com/PrakharSrivastav/oci-cloud-function/store"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter"
	"github.com/oracle/oci-go-sdk/v56/objectstorage"
	"log"
	"os"
	"time"
)

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
	d.reporter, d.tracer, d.rootSpan, err = infrastructure.GetLocalSpanWithTracerAndReporter(ctx, "mvr-file-transfer:schedular")
	if err != nil {
		return err
	}

	d.ctx = zipkin.NewContext(ctx, d.rootSpan)
	d.storageClient, err = infrastructure.NewLocalStorageClient(d.ctx, d.tracer)
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

func main() {
	ctx := context.Background()
	d := new(dependencies)
	err := d.init(ctx)
	if err != nil {
		log.Fatalf("error initializing deps : %v", err)
	}
	defer d.close()

	if err = d.queueScheduleJob(); err != nil {
		log.Println("error scheduling job")
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		return
	}

	sch, err := d.conn.GetScheduledJobByIDAndName(d.ctx, 1, "Schedule")
	if err != nil {
		log.Println("error getting job", err)
		d.rootSpan.Annotate(time.Now(), "error getting job")
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		return
	}

	d.conn.UpdateScheduledJobStatus(d.ctx, sch.ID, "Started", "Started Job")

	if err = d.uploadFile(); err != nil {
		log.Println("error uploading file")
		d.rootSpan.Tag(string(zipkin.TagError), err.Error())
		d.conn.UpdateScheduledJobStatus(d.ctx, sch.ID, "Failed", err.Error())
		return
	}

	d.conn.UpdateScheduledJobStatus(d.ctx, sch.ID, "Complete", "File Uploaded")

}

func (d *dependencies) uploadFile() error {
	span, _ := d.tracer.StartSpanFromContext(d.ctx, "read and upload fileﬁ")
	defer span.Finish()

	filename := "3.zip"
	f, err := os.Open(fmt.Sprintf("/Users/prakhar/workspace/prakhar/oci-cloud-function/hello/testdata/%s", filename))
	if err != nil {
		log.Println("error opening file", err)
		span.Annotate(time.Now(), "error opening file")
		span.Tag(string(zipkin.TagError), err.Error())
		return err
	}
	defer f.Close()

	ns := "axh1wuvhagpg"
	bktName := "func-app-bucket"
	if _, err = d.storageClient.PutObject(d.ctx, objectstorage.PutObjectRequest{
		NamespaceName: &ns,
		BucketName:    &bktName,
		ObjectName:    &filename,
		PutObjectBody: f,
		OpcMeta:       nil,
	}); err != nil {
		log.Println("error uploading file", err)
		span.Annotate(time.Now(), "error uploading file")
		span.Tag(string(zipkin.TagError), err.Error())
		return err
	}
	return nil
}

func (d *dependencies) queueScheduleJob() error {
	span, ctx := d.tracer.StartSpanFromContext(d.ctx, "schedule job")
	defer span.Finish()

	steps, err := d.conn.GetScheduledSteps(ctx, 1)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), "error reading sch job info")
		return err
	}

	history := make([]store.ScheduledHistory, len(steps))
	for i := range steps {
		history[i] = steps[i].ToHistory("Queued", "queued for processing")
	}
	if err = d.conn.AddScheduledHistory(ctx, history); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		span.Annotate(time.Now(), "error queuing scheduled jobs")
		return err
	}
	return nil
}
