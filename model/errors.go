package model

import "errors"

var (
	ErrInvalidCloudEventType    = errors.New("invalid.cloud.event.type")
	ErrInvalidCloudEventSource  = errors.New("invalid.cloud.event.source")
	ErrEmptyCloudEventData      = errors.New("empty.cloud.event.data")
	ErrInvalidBucketObject      = errors.New("invalid.bucket.object")
	ErrNotAZipFile              = errors.New("not.a.zip.file")
	ErrEmptyBucketInformation   = errors.New("empty.bucket.information")
	ErrInvalidBucketInformation = errors.New("invalid.bucket.information")
	ErrInvalidNamespace         = errors.New("invalid.namespace")
)
