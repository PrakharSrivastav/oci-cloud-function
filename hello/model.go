package main

import (
	"errors"
	"time"
)

type Message struct {
	Msg string `json:"message"`
}

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

func (b *BucketEvent) validate() error {
	if *b == (BucketEvent{}) {
		return errors.New("empty-body")
	}

	if b.EventType != "com.oraclecloud.objectstorage.createobject" {
		return errors.New("invalid event type")
	}

	if b.Source != "ObjectStorage" {
		return errors.New("invalid source type")
	}

	if b.Data == (Data{}) {
		return errors.New("empty data field")
	}

	if b.Data.ResourceName == "" || b.Data.ResourceID == "" {
		return errors.New("invalid bucket resource")
	}

	if b.Data.AdditionalDetails == (AdditionalDetails{}) {
		return errors.New("empty bucket details")
	}

	if b.Data.AdditionalDetails.BucketName == "" || b.Data.AdditionalDetails.BucketID == "" {
		return errors.New("invalid bucket details")
	}

	if b.Data.AdditionalDetails.Namespace == "" {
		return errors.New("invalid bucket namespace")
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
