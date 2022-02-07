package main

import (
	"context"
	"encoding/json"
	fdk "github.com/fnproject/fdk-go"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	rr "github.com/openzipkin/zipkin-go/reporter"
	zipkinHttpReporter "github.com/openzipkin/zipkin-go/reporter/http"
	"github.com/oracle/oci-go-sdk/v56/common/auth"
	"github.com/oracle/oci-go-sdk/v56/objectstorage"
	"io"
	"io/ioutil"
	"log"
	"strconv"
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

	log.Printf("object details are : %+v", object)

	msg := Message{Msg: "Hello World"}
	json.NewEncoder(out).Encode(&msg)
}

func getObject(ctx context.Context,
	client *objectstorage.ObjectStorageClient,
	event *BucketEvent,
	tt *zipkin.Tracer) (*objectstorage.GetObjectResponse, error) {

	span, ctx := tt.StartSpanFromContext(ctx, "GetBucketObject")
	defer span.Finish()

	request := objectstorage.GetObjectRequest{
		NamespaceName: &event.Data.AdditionalDetails.Namespace,
		BucketName:    &event.Data.AdditionalDetails.BucketName,
		ObjectName:    &event.Data.ResourceName,
	}

	object, err := client.GetObject(ctx, request)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}
	return &object, nil
}

func validateEvent(ctx context.Context, in io.Reader, tt *zipkin.Tracer) (*BucketEvent, error) {
	span, _ := tt.StartSpanFromContext(ctx, "validateEvent")
	defer span.Finish()

	bb, err := ioutil.ReadAll(in)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}

	event := BucketEvent{}
	err = json.Unmarshal(bb, &event)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}

	if err = event.validate(); err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}
	return &event, nil
}

func bucketClient(ctx context.Context, tt *zipkin.Tracer) (*objectstorage.ObjectStorageClient, error) {
	span, _ := tt.StartSpanFromContext(ctx, "get bucket client")
	defer span.Finish()

	provider, err := auth.ResourcePrincipalConfigurationProvider()
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}

	client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(provider)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}
	return &client, nil
}

func getSpanWithTracerAndReporter(ctx context.Context) (rr.Reporter, *zipkin.Tracer, zipkin.Span, error) {
	newCtx := fdk.GetContext(ctx)
	reporter := zipkinHttpReporter.NewReporter(endpointUrl)

	endpoint, err := zipkin.NewEndpoint(newCtx.FnName(), "")
	if err != nil {
		return nil, nil, nil, err
	}

	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
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
		log.Println("TRACE ID NOT DEFINED.....")
		return model.SpanContext{}
	}

	id, err := strconv.ParseUint(ctx.TracingContextData().SpanId(), 16, 64)
	if err != nil {
		log.Println("SPAN ID NOT DEFINED.....")
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
