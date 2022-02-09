package model

import (
	"errors"
	"strings"
	"time"
)

type BucketEvent struct {
	EventType          string    `json:"eventType"`
	CloudEventsVersion string    `json:"cloudEventsVersion"`
	EventTypeVersion   string    `json:"eventTypeVersion"`
	Source             string    `json:"source"`
	EventTime          time.Time `json:"eventTime"`
	ContentType        string    `json:"contentType"`
	Data               Data      `json:"data"`
	EventID            string    `json:"eventID"`
	Extensions         struct {
		CompartmentID string `json:"compartmentId"`
	} `json:"extensions"`
}

func (b *BucketEvent) Validate() error {
	if *b == (BucketEvent{}) {
		return errors.New("empty-body")
	}

	if b.EventType != "com.oraclecloud.objectstorage.createobject" {
		return ErrInvalidCloudEventType
	}

	if b.Source != "ObjectStorage" {
		return ErrInvalidCloudEventSource
	}

	if b.Data == (Data{}) {
		return ErrEmptyCloudEventData
	}

	if b.Data.ResourceName == "" || b.Data.ResourceID == "" {
		return ErrInvalidBucketObject
	}

	split := strings.Split(b.Data.ResourceName, ".")
	if len(split) == 0 || split[len(split)-1] != "zip" {
		return ErrNotAZipFile
	}

	if b.Data.AdditionalDetails == (AdditionalDetails{}) {
		return ErrEmptyBucketInformation
	}

	if b.Data.AdditionalDetails.BucketName == "" || b.Data.AdditionalDetails.BucketID == "" {
		return ErrInvalidBucketInformation
	}

	if b.Data.AdditionalDetails.Namespace == "" {
		return ErrInvalidNamespace
	}
	return nil
}

type Data struct {
	CompartmentID      string            `json:"compartmentId"`
	CompartmentName    string            `json:"compartmentName"`
	ResourceName       string            `json:"resourceName"`
	ResourceID         string            `json:"resourceId"`
	AvailabilityDomain string            `json:"availabilityDomain"`
	AdditionalDetails  AdditionalDetails `json:"additionalDetails"`
}

type AdditionalDetails struct {
	BucketName    string `json:"bucketName"`
	VersionID     string `json:"versionId"`
	ArchivalState string `json:"archivalState"`
	Namespace     string `json:"namespace"`
	BucketID      string `json:"bucketId"`
	ETag          string `json:"eTag"`
}
