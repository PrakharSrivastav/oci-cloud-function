package infrastructure

import (
	"context"
	"github.com/fnproject/fdk-go"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	rr "github.com/openzipkin/zipkin-go/reporter"
	zipkinHttpReporter "github.com/openzipkin/zipkin-go/reporter/http"
	"strconv"
)

const endpointUrl = "https://aaaac7wupury6aaaaaaaaaavku.apm-agt.eu-amsterdam-1.oci.oraclecloud.com/20200101/observations/public-span/?dataFormat=zipkin&dataFormatVersion=2&dataKey=QAG4HXI6MVIW45ABJUUDTRJ4PYPRGHYU"

func GetSpanWithTracerAndReporter(ctx context.Context, functionName string) (rr.Reporter, *zipkin.Tracer, zipkin.Span, error) {
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
	span, ctx := tracer.StartSpanFromContext(ctx, functionName, sopt)
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
		Sampled:  boolAddr(ctx.TracingContextData().IsSampled()),
	}
}

func boolAddr(b bool) *bool {
	boolVar := b
	return &boolVar
}

func GetLocalSpanWithTracerAndReporter(ctx context.Context, funcationName string) (rr.Reporter, *zipkin.Tracer, zipkin.Span, error) {
	reporter := zipkinHttpReporter.NewReporter(endpointUrl)

	endpoint, err := zipkin.NewEndpoint(funcationName, "")
	if err != nil {
		return nil, nil, nil, err
	}

	sampler, err := zipkin.NewCountingSampler(1.0)
	if err != nil {
		return nil, nil, nil, err
	}

	// initialize our tracer
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint), zipkin.WithSampler(sampler))
	if err != nil {
		return nil, nil, nil, err
	}
	span, _ := tracer.StartSpanFromContext(ctx, funcationName, zipkin.Kind(model.Client))
	return reporter, tracer, span, nil

}
