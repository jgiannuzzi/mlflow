package main

import "C"

import (
	"unsafe"

	"github.com/mlflow/mlflow/mlflow/go/pkg/tracking/service"
)

var trackingServices = newServiceMap[service.TrackingService]()

//export CreateTrackingService
func CreateTrackingService(configData unsafe.Pointer, configSize C.int) int64 {
	return trackingServices.Create(service.NewTrackingService, configData, configSize)
}

//export DestroyTrackingService
func DestroyTrackingService(id int64) {
	trackingServices.Destroy(id)
}
