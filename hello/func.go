package main

import (
	"bytes"
	"context"
	"encoding/json"
	fdk "github.com/fnproject/fdk-go"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	rr "github.com/openzipkin/zipkin-go/reporter"
	zipkinHttpReporter "github.com/openzipkin/zipkin-go/reporter/http"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

const endpointUrl = ""

func main() {
	fdk.Handle(fdk.HandlerFunc(myHandler))
}

func myHandler(ctx context.Context, in io.Reader, out io.Writer) {
	reporter, tt, span, err := getSpanWithTracerAndReporter(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer reporter.Close()
	defer span.Finish()

	ctx = zipkin.NewContext(ctx, span)

	event, err := validateEvent(ctx, in, tt)
	if err != nil {
		log.Print("event validation error :", err)
		return
	}

	client, err := bucketClient(ctx, tt)
	if err != nil {
		log.Print("bucket client error :", err)
		return
	}

	object, err := getObject(ctx, client, event, tt)
	if err != nil {
		log.Print("get object error :", err)
		return
	}
	defer object.Content.Close()

	log.Printf("object details are : %+v", object)
	if *object.ContentLength == 0 {
		log.Print("not content")
		return
	}

	var cc []byte
	cBuf := bytes.NewBuffer(cc)
	if _, err = io.Copy(cBuf, object.Content); err != nil {
		log.Print("read file object error", err)
		return
	}

	fileName, err := copyContentAsZip(ctx, cBuf, tt)
	if err != nil {
		log.Print("can not copy as zip", err)
		return
	}
	defer os.RemoveAll(fileName)
	log.Print("content written to zipfile ", fileName)

	dest, str, err := unzipFiles(ctx, fileName, tt)
	if err != nil {
		log.Print("can not write to zip-file ", fileName, err)
		return
	}
	defer os.RemoveAll(dest)
	log.Print("unzipped : \n", strings.Join(str, "\n"))

	msg := Message{Msg: "Hello World"}
	json.NewEncoder(out).Encode(&msg)
}

func validateEvent(ctx context.Context, in io.Reader, tt *zipkin.Tracer) (*BucketEvent, error) {
	span, _ := tt.StartSpanFromContext(ctx, "validateEvent")
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

func getSpanWithTracerAndReporter(ctx context.Context) (rr.Reporter, *zipkin.Tracer, zipkin.Span, error) {
	newCtx := fdk.GetContext(ctx)
	reporter := zipkinHttpReporter.NewReporter(endpointUrl)

	endpoint, err := zipkin.NewEndpoint(newCtx.FnName(), "")
	if err != nil {
		return nil, nil, nil, err
	}
	sampler, err := zipkin.NewCountingSampler(1.0)
	if err != nil {
		return nil, nil, nil, err
	}
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint), zipkin.WithSampler(sampler))
	if err != nil {
		return nil, nil, nil, err
	}

	sopt := zipkin.Parent(setContext(newCtx))
	span, ctx := tracer.StartSpanFromContext(ctx, "hello-fn", sopt)
	return reporter, tracer, span, nil
}

type Message struct {
	Msg string `json:"message"`
}

func setContext(ctx fdk.Context) model.SpanContext {
	traceId, err := model.TraceIDFromHex(ctx.TracingContextData().TraceId())

	if err != nil {
		return model.SpanContext{}
	}

	id, err := strconv.ParseUint(ctx.TracingContextData().SpanId(), 16, 64)
	if err != nil {
		return model.SpanContext{}
	}

	return model.SpanContext{
		TraceID:  traceId,
		ID:       model.ID(id),
		ParentID: nil,
		Sampled:  BoolAddr(ctx.TracingContextData().IsSampled()),
	}
}

func BoolAddr(b bool) *bool {
	boolVar := b
	return &boolVar
}
