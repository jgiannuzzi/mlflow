package service

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mlflow/mlflow/mlflow/go/pkg/contract"
	"github.com/mlflow/mlflow/mlflow/go/pkg/protos"
)

// CreateExperiment implements MlflowService.
func (m TrackingService) CreateExperiment(input *protos.CreateExperiment) (
	*protos.CreateExperiment_Response, *contract.Error,
) {
	if input.GetArtifactLocation() != "" {
		artifactLocation := strings.TrimRight(input.GetArtifactLocation(), "/")

		// We don't check the validation here as this was already covered in the validator.
		url, _ := url.Parse(artifactLocation)
		switch url.Scheme {
		case "file", "":
			path, err := filepath.Abs(url.Path)
			if err != nil {
				return nil, contract.NewError(
					protos.ErrorCode_INVALID_PARAMETER_VALUE,
					fmt.Sprintf("error getting absolute path: %v", err),
				)
			}

			if runtime.GOOS == "windows" {
				url.Scheme = "file"
				path = "/" + strings.ReplaceAll(path, "\\", "/")
			}

			url.Path = path
			artifactLocation = url.String()
		}

		input.ArtifactLocation = &artifactLocation
	}

	experimentID, err := m.Store.CreateExperiment(input)
	if err != nil {
		return nil, err
	}

	response := protos.CreateExperiment_Response{
		ExperimentId: &experimentID,
	}

	return &response, nil
}

// GetExperiment implements MlflowService.
func (m TrackingService) GetExperiment(input *protos.GetExperiment) (*protos.GetExperiment_Response, *contract.Error) {
	experiment, cErr := m.Store.GetExperiment(input.GetExperimentId())
	if cErr != nil {
		return nil, cErr
	}

	response := protos.GetExperiment_Response{
		Experiment: experiment,
	}

	return &response, nil
}

func (m TrackingService) DeleteExperiment(
	input *protos.DeleteExperiment,
) (*protos.DeleteExperiment_Response, *contract.Error) {
	err := m.Store.DeleteExperiment(input.GetExperimentId())
	if err != nil {
		return nil, err
	}

	return &protos.DeleteExperiment_Response{}, nil
}
