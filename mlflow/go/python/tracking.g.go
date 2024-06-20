package main

import "C"

import (
	"encoding/json"
	"unsafe"

	"google.golang.org/protobuf/proto"

	"github.com/mlflow/mlflow/mlflow/go/pkg/contract"
	"github.com/mlflow/mlflow/mlflow/go/pkg/protos"
	"github.com/mlflow/mlflow/mlflow/go/pkg/validation"
)

//export TrackingServiceGetExperiment
func TrackingServiceGetExperiment(
	serviceID int64,
	requestData unsafe.Pointer,
	requestSize C.int,
	responseSize *C.int,
) unsafe.Pointer {
	responseData, err := func() (unsafe.Pointer, *contract.Error) {
		service, cErr := getTrackingService(serviceID)
		if cErr != nil {
			return nil, cErr
		}

		request := new(protos.GetExperiment)

		if err := proto.Unmarshal(C.GoBytes(requestData, requestSize), request); err != nil {
			return nil, contract.NewError(
				protos.ErrorCode_BAD_REQUEST,
				err.Error(),
			)
		}

		validate, cErr := getValidator()
		if cErr != nil {
			return nil, cErr
		}

		if err := validate.Struct(request); err != nil {
			return nil, validation.NewErrorFromValidationError(err)
		}

		response, cErr := service.GetExperiment(request)
		if cErr != nil {
			return nil, cErr
		}

		res, err := proto.Marshal(response)
		if err != nil {
			return nil, contract.NewError(
				protos.ErrorCode_INTERNAL_ERROR,
				err.Error(),
			)
		}

		*responseSize = C.int(len(res))

		return C.CBytes(res), nil
	}()
	if err != nil {
		data, _ := json.Marshal(err)
		*responseSize = C.int(len(data))

		return C.CBytes(data)
	}

	return responseData
}

//export TrackingServiceCreateExperiment
func TrackingServiceCreateExperiment(
	serviceID int64,
	requestData unsafe.Pointer,
	requestSize C.int,
	responseSize *C.int,
) unsafe.Pointer {
	responseData, err := func() (unsafe.Pointer, *contract.Error) {
		service, cErr := getTrackingService(serviceID)
		if cErr != nil {
			return nil, cErr
		}

		request := new(protos.CreateExperiment)

		if err := proto.Unmarshal(C.GoBytes(requestData, requestSize), request); err != nil {
			return nil, contract.NewError(
				protos.ErrorCode_BAD_REQUEST,
				err.Error(),
			)
		}

		validate, cErr := getValidator()
		if cErr != nil {
			return nil, cErr
		}

		if err := validate.Struct(request); err != nil {
			return nil, validation.NewErrorFromValidationError(err)
		}

		response, cErr := service.CreateExperiment(request)
		if cErr != nil {
			return nil, cErr
		}

		res, err := proto.Marshal(response)
		if err != nil {
			return nil, contract.NewError(
				protos.ErrorCode_INTERNAL_ERROR,
				err.Error(),
			)
		}

		*responseSize = C.int(len(res))
		return C.CBytes(res), nil
	}()
	if err != nil {
		data, _ := json.Marshal(err)
		*responseSize = C.int(len(data))

		return C.CBytes(data)
	}

	return responseData
}
