package main

import "C"

import (
	"unsafe"

	"github.com/mlflow/mlflow/mlflow/go/pkg/artifacts/service"
)

var artifactsServices = newServiceMap[service.ArtifactsService]()

//export CreateArtifactsService
func CreateArtifactsService(configData unsafe.Pointer, configSize C.int) int64 {
	return artifactsServices.Create(service.NewArtifactsService, configData, configSize)
}

//export DestroyArtifactsService
func DestroyArtifactsService(id int64) {
	artifactsServices.Destroy(id)
}
