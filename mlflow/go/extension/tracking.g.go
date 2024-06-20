package main

import "C"

import (
	"unsafe"

	"github.com/mlflow/mlflow/mlflow/go/pkg/protos"
)

//export TrackingServiceGetExperiment
func TrackingServiceGetExperiment(
	serviceID int64,
	requestData unsafe.Pointer,
	requestSize C.int,
	responseSize *C.int,
) unsafe.Pointer {
	service, err := trackingServices.Get(serviceID)
	if err != nil {
		return makePointerFromError(err, responseSize)
	}

	return invokeServiceMethod(
		service.GetExperiment,
		new(protos.GetExperiment),
		requestData,
		requestSize,
		responseSize,
	)
}

//export TrackingServiceCreateExperiment
func TrackingServiceCreateExperiment(
	serviceID int64,
	requestData unsafe.Pointer,
	requestSize C.int,
	responseSize *C.int,
) unsafe.Pointer {
	service, err := trackingServices.Get(serviceID)
	if err != nil {
		return makePointerFromError(err, responseSize)
	}

	return invokeServiceMethod(
		service.CreateExperiment,
		new(protos.CreateExperiment),
		requestData,
		requestSize,
		responseSize,
	)
}
