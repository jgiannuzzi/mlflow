package service

import (
	"github.com/mlflow/mlflow/mlflow/go/pkg/contract"
	"github.com/mlflow/mlflow/mlflow/go/pkg/protos"
)

func (m *ModelRegistryService) GetLatestVersions(
	_ *protos.GetLatestVersions,
) (*protos.GetLatestVersions_Response, *contract.Error) {
	return nil, nil
}
