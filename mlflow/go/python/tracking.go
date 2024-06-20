package main

import "C"

import (
	"encoding/json"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/mlflow/mlflow/mlflow/go/pkg/config"
	"github.com/mlflow/mlflow/mlflow/go/pkg/contract"
	"github.com/mlflow/mlflow/mlflow/go/pkg/protos"
	"github.com/mlflow/mlflow/mlflow/go/pkg/tracking/service"
)

var (
	trackingServices        = make(map[int64]contract.TrackingService)
	trackingServicesMutex   sync.Mutex
	trackingServicesCounter int64
)

//export CreateTrackingService
func CreateTrackingService(configJSON *C.char) int64 {
	var config *config.Config
	if err := json.Unmarshal([]byte(C.GoString(configJSON)), &config); err != nil {
		logrus.Error(err)

		return -1
	}

	logger := logrus.New()

	logLevel, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logrus.Error(err)

		return -1
	}
	logger.SetLevel(logLevel)

	logger.Warn("The experimental Go server is not yet fully supported and may not work as expected")
	logger.Debugf("Loaded config: %#v", config)

	trackingService, err := service.NewTrackingService(logger, config)
	if err != nil {
		logger.Error(err)

		return -1
	}

	trackingServicesMutex.Lock()
	defer trackingServicesMutex.Unlock()
	trackingServicesCounter++
	trackingServices[trackingServicesCounter] = trackingService

	return trackingServicesCounter
}

//export DestroyTrackingService
func DestroyTrackingService(id int64) {
	trackingServicesMutex.Lock()
	defer trackingServicesMutex.Unlock()
	delete(trackingServices, id)
}

//nolint:ireturn
func getTrackingService(id int64) (contract.TrackingService, *contract.Error) {
	trackingServicesMutex.Lock()
	trackingService, ok := trackingServices[id]
	trackingServicesMutex.Unlock()

	if !ok {
		return nil, contract.NewError(
			protos.ErrorCode_RESOURCE_DOES_NOT_EXIST,
			"Service not found",
		)
	}

	return trackingService, nil
}
