package sql

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mlflow/mlflow/mlflow/go/pkg/contract"
	"github.com/mlflow/mlflow/mlflow/go/pkg/protos"
	"github.com/mlflow/mlflow/mlflow/go/pkg/tracking/service/query"
	"github.com/mlflow/mlflow/mlflow/go/pkg/tracking/service/query/parser"
	"github.com/mlflow/mlflow/mlflow/go/pkg/tracking/store"
	"github.com/mlflow/mlflow/mlflow/go/pkg/tracking/store/sql/models"
	"github.com/mlflow/mlflow/mlflow/go/pkg/utils"
)

var runOrder = regexp.MustCompile(
	`^(attribute|metric|param|tag)s?\.("[^"]+"|` + "`[^`]+`" + `|[\w\.]+)(?i:\s+(ASC|DESC))?$`,
)

type PageToken struct {
	Offset int32 `json:"offset"`
}

func getLifecyleStages(runViewType protos.ViewType) []models.LifecycleStage {
	switch runViewType {
	case protos.ViewType_ACTIVE_ONLY:
		return []models.LifecycleStage{
			models.LifecycleStageActive,
		}
	case protos.ViewType_DELETED_ONLY:
		return []models.LifecycleStage{
			models.LifecycleStageDeleted,
		}
	case protos.ViewType_ALL:
		return []models.LifecycleStage{
			models.LifecycleStageActive,
			models.LifecycleStageDeleted,
		}
	}

	return []models.LifecycleStage{
		models.LifecycleStageActive,
		models.LifecycleStageDeleted,
	}
}

func getOffset(pageToken string) (int, *contract.Error) {
	if pageToken != "" {
		var token PageToken
		if err := json.NewDecoder(
			base64.NewDecoder(
				base64.StdEncoding,
				strings.NewReader(pageToken),
			),
		).Decode(&token); err != nil {
			return 0, contract.NewErrorWith(
				protos.ErrorCode_INVALID_PARAMETER_VALUE,
				fmt.Sprintf("invalid page_token: %q", pageToken),
				err,
			)
		}

		return int(token.Offset), nil
	}

	return 0, nil
}

//nolint:funlen,cyclop,gocognit
func applyFilters(store *TrackingSQLStore, transaction *gorm.DB, filter string) *contract.Error {
	filterConditions, err := query.ParseFilter(filter)
	if err != nil {
		return contract.NewErrorWith(
			protos.ErrorCode_INVALID_PARAMETER_VALUE,
			"error parsing search filter",
			err,
		)
	}

	log.Debugf("Filter conditions: %#v", filterConditions)

	for index, clause := range filterConditions {
		var kind any

		key := clause.Key
		comparison := strings.ToUpper(clause.Operator.String())
		value := clause.Value

		switch clause.Identifier {
		case parser.Metric:
			kind = &models.LatestMetric{}
		case parser.Parameter:
			kind = &models.Param{}
		case parser.Tag:
			kind = &models.Tag{}
		case parser.Dataset:
			kind = &models.Dataset{}
		case parser.Attribute:
			kind = nil
		}

		// Treat "attributes.run_name == <value>" as "tags.`mlflow.runName` == <value>".
		// The name column in the runs table is empty for runs logged in MLflow <= 1.29.0.
		if key == "run_name" {
			kind = &models.Tag{}
			key = utils.TagRunName
		}

		isSqliteAndILike := store.db.Dialector.Name() == "sqlite" && comparison == "ILIKE"
		table := fmt.Sprintf("filter_%d", index)

		switch {
		case kind == nil:
			if isSqliteAndILike {
				key = fmt.Sprintf("LOWER(runs.%s)", key)
				comparison = "LIKE"

				if str, ok := value.(string); ok {
					value = strings.ToLower(str)
				}

				transaction.Where(fmt.Sprintf("%s %s ?", key, comparison), value)
			} else {
				transaction.Where(fmt.Sprintf("runs.%s %s ?", key, comparison), value)
			}
		case clause.Identifier == parser.Dataset && key == "context":
			// SELECT *
			// FROM runs
			// JOIN (
			//   SELECT inputs.destination_id AS run_uuid
			//   FROM inputs
			//   JOIN input_tags
			//   ON inputs.input_uuid = input_tags.input_uuid
			//   AND input_tags.name = 'mlflow.data.context'
			//   AND input_tags.value %s ?
			//   WHERE inputs.destination_type = 'RUN'
			// ) AS filter_0
			// ON runs.run_uuid = filter_0.run_uuid
			valueColumn := "input_tags.value "
			if isSqliteAndILike {
				valueColumn = "LOWER(input_tags.value) "

				if str, ok := value.(string); ok {
					value = strings.ToLower(str)
				}
			}

			transaction.Joins(
				fmt.Sprintf("JOIN (?) AS %s ON runs.run_uuid = %s.run_uuid", table, table),
				store.db.Select("inputs.destination_id AS run_uuid").
					Joins(
						"JOIN input_tags ON inputs.input_uuid = input_tags.input_uuid"+
							" AND input_tags.name = 'mlflow.data.context'"+
							" AND "+valueColumn+comparison+" ?",
						value,
					).
					Where("inputs.destination_type = 'RUN'").
					Model(&models.Input{}),
			)
		case clause.Identifier == parser.Dataset:
			// add join with datasets
			// JOIN (
			// 		SELECT "experiment_id", key
			//		FROM datasests
			//		WHERE key comparison value
			// ) AS filter_0 ON runs.experiment_id = dataset.experiment_id
			//
			// columns: name, digest, context
			where := key + " " + comparison + " ?"
			if isSqliteAndILike {
				where = "LOWER(" + key + ") LIKE ?"

				if str, ok := value.(string); ok {
					value = strings.ToLower(str)
				}
			}

			transaction.Joins(
				fmt.Sprintf("JOIN (?) AS %s ON runs.experiment_id = %s.experiment_id", table, table),
				store.db.Select("experiment_id", key).Where(where, value).Model(kind),
			)
		default:
			where := fmt.Sprintf("value %s ?", comparison)
			if isSqliteAndILike {
				where = "LOWER(value) LIKE ?"

				if str, ok := value.(string); ok {
					value = strings.ToLower(str)
				}
			}

			transaction.Joins(
				fmt.Sprintf("JOIN (?) AS %s ON runs.run_uuid = %s.run_uuid", table, table),
				store.db.Select("run_uuid", "value").Where("key = ?", key).Where(where, value).Model(kind),
			)
		}
	}

	return nil
}

//nolint:funlen, cyclop
func applyOrderBy(store *TrackingSQLStore, transaction *gorm.DB, orderBy []string) *contract.Error {
	startTimeOrder := false

	for index, orderByClause := range orderBy {
		components := runOrder.FindStringSubmatch(orderByClause)
		log.Debugf("Components: %#v", components)
		//nolint:mnd
		if len(components) < 3 {
			return contract.NewError(
				protos.ErrorCode_INVALID_PARAMETER_VALUE,
				"invalid order by clause: "+orderByClause,
			)
		}

		column := strings.Trim(components[2], "`\"")

		var kind any

		switch components[1] {
		case "attribute":
			if column == "start_time" {
				startTimeOrder = true
			}
		case "metric":
			kind = &models.LatestMetric{}
		case "param":
			kind = &models.Param{}
		case "tag":
			kind = &models.Tag{}
		default:
			return contract.NewError(
				protos.ErrorCode_INVALID_PARAMETER_VALUE,
				fmt.Sprintf(
					"invalid entity type '%s'. Valid values are ['metric', 'parameter', 'tag', 'attribute']",
					components[1],
				),
			)
		}

		if kind != nil {
			table := fmt.Sprintf("order_%d", index)
			transaction.Joins(
				fmt.Sprintf("LEFT OUTER JOIN (?) AS %s ON runs.run_uuid = %s.run_uuid", table, table),
				store.db.Select("run_uuid", "value").Where("key = ?", column).Model(kind),
			)

			column = table + ".value"
		}

		transaction.Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: column,
			},
			Desc: len(components) == 4 && strings.ToUpper(components[3]) == "DESC",
		})
	}

	if !startTimeOrder {
		transaction.Order("runs.start_time DESC")
	}

	transaction.Order("runs.run_uuid")

	return nil
}

func mkNextPageToken(runLength, maxResults, offset int) (*string, *contract.Error) {
	var nextPageToken *string

	if runLength == maxResults {
		var token strings.Builder
		if err := json.NewEncoder(
			base64.NewEncoder(base64.StdEncoding, &token),
		).Encode(PageToken{
			Offset: int32(offset + maxResults),
		}); err != nil {
			return nil, contract.NewErrorWith(
				protos.ErrorCode_INTERNAL_ERROR,
				"error encoding 'nextPageToken' value",
				err,
			)
		}

		nextPageToken = utils.PtrTo(token.String())
	}

	return nextPageToken, nil
}

func (s TrackingSQLStore) SearchRuns(
	experimentIDs []string, filter string,
	runViewType protos.ViewType, maxResults int, orderBy []string, pageToken string,
) (*store.PagedList[*protos.Run], *contract.Error) {
	// ViewType
	lifecyleStages := getLifecyleStages(runViewType)
	transaction := s.db.Where("runs.experiment_id IN ?", experimentIDs).Where("runs.lifecycle_stage IN ?", lifecyleStages)

	// MaxResults
	transaction.Limit(maxResults)

	// PageToken
	offset, contractError := getOffset(pageToken)
	if contractError != nil {
		return nil, contractError
	}

	transaction.Offset(offset)

	// Filter
	contractError = applyFilters(&s, transaction, filter)
	if contractError != nil {
		return nil, contractError
	}

	// OrderBy
	contractError = applyOrderBy(&s, transaction, orderBy)
	if contractError != nil {
		return nil, contractError
	}

	// Actual query
	var runs []models.Run

	transaction.Preload("LatestMetrics").Preload("Params").Preload("Tags").
		Preload("Inputs", "inputs.destination_type = 'RUN'").
		Preload("Inputs.Dataset").Preload("Inputs.Tags").Find(&runs)

	if transaction.Error != nil {
		return nil, contract.NewErrorWith(
			protos.ErrorCode_INTERNAL_ERROR,
			"Failed to query search runs",
			transaction.Error,
		)
	}

	contractRuns := make([]*protos.Run, 0, len(runs))
	for _, run := range runs {
		contractRuns = append(contractRuns, run.ToProto())
	}

	nextPageToken, contractError := mkNextPageToken(len(runs), maxResults, offset)
	if contractError != nil {
		return nil, contractError
	}

	return &store.PagedList[*protos.Run]{
		Items:         contractRuns,
		NextPageToken: nextPageToken,
	}, nil
}

const RunIDMaxLength = 32

const (
	ArtifactFolderName  = "artifacts"
	RunNameIntegerScale = 3
	RunNameMaxLength    = 20
)

func ensureRunName(runModel *models.Run) *contract.Error {
	var runNameFromTags string

	for _, tag := range runModel.Tags {
		if *tag.Key == utils.TagRunName {
			runNameFromTags = *tag.Value

			break
		}
	}

	if utils.IsNotNilOrEmptyString(runModel.Name) && runNameFromTags != "" && *runModel.Name != runNameFromTags {
		return contract.NewError(
			protos.ErrorCode_INVALID_PARAMETER_VALUE,
			fmt.Sprintf(
				"Both 'run_name' argument and 'mlflow.runName' tag are specified, but with \n"+
					"different values (run_name=%q, mlflow.runName=%q).",
				*runModel.Name,
				runNameFromTags,
			),
		)
	}

	if utils.IsNilOrEmptyString(runModel.Name) {
		randomName, err := utils.GenerateRandomName()
		if err != nil {
			return contract.NewErrorWith(
				protos.ErrorCode_INTERNAL_ERROR,
				"failed to generate random run name",
				err,
			)
		}

		runModel.Name = utils.PtrTo(randomName)
	}

	if runNameFromTags == "" {
		runModel.Tags = append(runModel.Tags, models.Tag{
			Key:   utils.PtrTo(utils.TagRunName),
			Value: runModel.Name,
		})
	}

	return nil
}

func (s TrackingSQLStore) CreateRun(input *protos.CreateRun) (*protos.Run, *contract.Error) {
	experiment, err := s.GetExperiment(input.GetExperimentId())
	if err != nil {
		return nil, err
	}

	if models.LifecycleStage(experiment.GetLifecycleStage()) != models.LifecycleStageActive {
		return nil, contract.NewError(
			protos.ErrorCode_INVALID_PARAMETER_VALUE,
			fmt.Sprintf(
				"The experiment %q must be in the 'active' state.\n"+
					"Current state is %q.",
				input.GetExperimentId(),
				experiment.GetLifecycleStage(),
			),
		)
	}

	runModel := models.NewRunFromCreateRunProto(input)

	artifactLocation, appendErr := url.JoinPath(
		experiment.GetArtifactLocation(),
		*runModel.ID,
		ArtifactFolderName,
	)
	if appendErr != nil {
		return nil, contract.NewError(
			protos.ErrorCode_INTERNAL_ERROR,
			"failed to append run ID to experiment artifact location",
		)
	}

	runModel.ArtifactURI = &artifactLocation

	errRunName := ensureRunName(runModel)
	if errRunName != nil {
		return nil, errRunName
	}

	if err := s.db.Create(&runModel).Error; err != nil {
		return nil, contract.NewErrorWith(
			protos.ErrorCode_INTERNAL_ERROR,
			fmt.Sprintf(
				"failed to create run for experiment_id %q",
				input.GetExperimentId(),
			),
			err,
		)
	}

	return runModel.ToProto(), nil
}
