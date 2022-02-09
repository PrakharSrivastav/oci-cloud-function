package helper

import (
	"context"
	"github.com/PrakharSrivastav/oci-cloud-function/model"
	"github.com/openzipkin/zipkin-go"
	"github.com/oracle/oci-go-sdk/v56/objectstorage"
)

func DownloadObjectFromBucket(ctx context.Context,
	client *objectstorage.ObjectStorageClient,
	event *model.BucketEvent,
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
