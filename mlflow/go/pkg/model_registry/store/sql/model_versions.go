package sql

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/mlflow/mlflow/mlflow/go/pkg/contract"
	"github.com/mlflow/mlflow/mlflow/go/pkg/model_registry/store/sql/models"
	"github.com/mlflow/mlflow/mlflow/go/pkg/protos"
)

// Validate whether there is a single registered model with the given name.
func assertModelExists(db *gorm.DB, name string) *contract.Error {
	var rows []int

	err := db.Model(&models.RegisteredModel{}).Where("name = ?", name).Select("1").Find(&rows).Error
	if err != nil {
		return contract.NewErrorWith(
			protos.ErrorCode_INTERNAL_ERROR,
			fmt.Sprintf("failed to get registered models for %q", name),
			err,
		)
	}

	if len(rows) == 0 {
		return contract.NewError(
			protos.ErrorCode_RESOURCE_DOES_NOT_EXIST,
			fmt.Sprintf("registered model with name=%q not found", name),
		)
	}

	if len(rows) > 1 {
		return contract.NewError(
			protos.ErrorCode_INVALID_STATE,
			fmt.Sprintf(
				"expected only 1 registered model with name=%q. Found %d.",
				name,
				len(rows),
			),
		)
	}

	return nil
}

func (m *ModelRegistrySQLStore) GetLatestVersions(
	name string, stages []string,
) ([]*protos.ModelVersion, *contract.Error) {
	existsErr := assertModelExists(m.db, name)
	if existsErr != nil {
		return nil, existsErr
	}

	var modelVersions []*models.ModelVersion

	subQuery := m.db.
		Model(&models.ModelVersion{}).
		Select("name, MAX(version) AS max_version").
		Where("name = ?", name).
		Where("current_stage <> ?", models.StageDeletedInternal).
		Group("name, current_stage")

	if len(stages) > 0 {
		subQuery = subQuery.Where("current_stage IN (?)", stages)
	}

	err := m.db.
		Model(&models.ModelVersion{}).
		Joins("JOIN (?) AS sub ON model_versions.name = sub.name AND model_versions.version = sub.max_version", subQuery).
		Find(&modelVersions).Error
	if err != nil {
		return nil, contract.NewErrorWith(
			protos.ErrorCode_INTERNAL_ERROR,
			fmt.Sprintf("could not query latest model version for %q", name),
			err,
		)
	}

	results := make([]*protos.ModelVersion, 0, len(modelVersions))
	for _, modelVersion := range modelVersions {
		results = append(results, modelVersion.ToProto())
	}

	return results, nil
}
