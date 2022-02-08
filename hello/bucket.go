package main

import (
	"context"
	"github.com/openzipkin/zipkin-go"
	"github.com/oracle/oci-go-sdk/v56/common/auth"
	"github.com/oracle/oci-go-sdk/v56/objectstorage"
)

func objectStorageClient(ctx context.Context, tt *zipkin.Tracer) (*objectstorage.ObjectStorageClient, error) {
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

func downloadObjectFromBucket(ctx context.Context,
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
