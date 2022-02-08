package main

import (
	"context"
	"github.com/fnproject/fdk-go"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	rr "github.com/openzipkin/zipkin-go/reporter"
	zipkinHttpReporter "github.com/openzipkin/zipkin-go/reporter/http"
	"strconv"
)

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
