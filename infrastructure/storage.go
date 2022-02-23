package infrastructure

import (
	"context"
	"github.com/openzipkin/zipkin-go"
	"github.com/oracle/oci-go-sdk/v56/common"
	"github.com/oracle/oci-go-sdk/v56/common/auth"
	"github.com/oracle/oci-go-sdk/v56/objectstorage"
)

func NewStorageClient(ctx context.Context, tracer *zipkin.Tracer) (*objectstorage.ObjectStorageClient, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "get bucket client")
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

func NewLocalStorageClient(ctx context.Context, tracer *zipkin.Tracer) (*objectstorage.ObjectStorageClient, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "get local bucket client")
	defer span.Finish()

	provider := common.DefaultConfigProvider()
	client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(provider)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		return nil, err
	}

	return &client, nil

}
