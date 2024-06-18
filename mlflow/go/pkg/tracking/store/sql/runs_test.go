package sql_test

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mlflow/mlflow/mlflow/go/pkg/tracking/store/sql"
	"github.com/mlflow/mlflow/mlflow/go/pkg/tracking/store/sql/models"
)

type testData struct {
	name         string
	query        string
	expectedSQL  string
	expectedVars []any
}

var whitespaceRegex = regexp.MustCompile(`\s+`)

func removeWhitespace(s string) string {
	return whitespaceRegex.ReplaceAllString(s, "")
}

var tests = []testData{
	{
		name:  "simple metric query",
		query: "metrics.accuracy > 0.72",
		expectedSQL: `
SELECT "run_uuid" FROM "runs"
JOIN (SELECT "run_uuid","value" FROM "latest_metrics" WHERE key = $1 AND value > $2)
AS filter_0
ON runs.run_uuid = filter_0.run_uuid`,
		expectedVars: []any{"accuracy", 0.72},
	},
	{
		name:  "simple metric and param query",
		query: "metrics.accuracy > 0.72 AND params.batch_size = '2'",
		expectedSQL: `
SELECT "run_uuid" FROM "runs"
JOIN (SELECT "run_uuid","value" FROM "latest_metrics" WHERE key = $1 AND value > $2)
AS filter_0 ON runs.run_uuid = filter_0.run_uuid
JOIN (SELECT "run_uuid","value" FROM "params" WHERE key = $3 AND value = $4)
AS filter_1 ON runs.run_uuid = filter_1.run_uuid
`,
		expectedVars: []any{"accuracy", 0.72, "batch_size", "2"},
	},
	{
		name:  "tag query",
		query: "tags.environment = 'notebook' AND tags.task ILIKE 'classif%'",
		expectedSQL: `
SELECT "run_uuid" FROM "runs"
JOIN (SELECT "run_uuid","value" FROM "tags" WHERE key = $1 AND value = $2)
AS filter_0 ON runs.run_uuid = filter_0.run_uuid
JOIN (SELECT "run_uuid","value" FROM "tags" WHERE key = $3 AND value ILIKE $4)
AS filter_1 ON runs.run_uuid = filter_1.run_uuid`,
		expectedVars: []any{"environment", "notebook", "task", "classif%"},
	},
	{
		name:  "datasests IN query",
		query: "datasets.digest IN ('s8ds293b', 'jks834s2')",
		expectedSQL: `
SELECT "run_uuid" FROM "runs"
JOIN (SELECT "experiment_id","digest" FROM "datasets" WHERE digest IN ($1,$2))
AS filter_0 ON runs.experiment_id = filter_0.experiment_id
`,
		expectedVars: []any{"s8ds293b", "jks834s2"},
	},
	{
		name:  "attributes query",
		query: "attributes.run_id = 'a1b2c3d4'",
		expectedSQL: `
SELECT "run_uuid" FROM "runs"
WHERE runs.run_uuid = $1
`,
		expectedVars: []any{"a1b2c3d4"},
	},
	{
		name:  "run_name query",
		query: "attributes.run_name = 'my-run'",
		expectedSQL: `
SELECT "run_uuid" FROM "runs"
JOIN (SELECT "run_uuid","value" FROM "tags" WHERE key = $1 AND value = $2)
AS filter_0 ON runs.run_uuid = filter_0.run_uuid`,
		expectedVars: []any{"mlflow.runName", "my-run"},
	},
	{
		name:  "datasets.context query",
		query: "datasets.context = 'train'",
		expectedSQL: `
SELECT "run_uuid" FROM "runs"
JOIN (
	SELECT inputs.destination_id AS run_uuid
	FROM "inputs"
	JOIN input_tags
	ON inputs.input_uuid = input_tags.input_uuid
	AND input_tags.name = 'mlflow.data.context'
	AND input_tags.value = $1
	WHERE inputs.destination_type = 'RUN'
) AS filter_0 ON runs.run_uuid = filter_0.run_uuid`,
		expectedVars: []any{"train"},
	},
}

func TestSearchRuns(t *testing.T) {
	t.Parallel()

	mockedDB, _, err := sqlmock.New()
	require.NoError(t, err)

	database, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       mockedDB,
		DriverName: "postgres",
	}), &gorm.Config{DryRun: true})

	require.NoError(t, err)

	for _, testData := range tests {
		currentTestData := testData

		t.Run(currentTestData.name, func(t *testing.T) {
			t.Parallel()

			transaction := database.Model(&models.Run{})

			contractErr := sql.ApplyFilter(database, transaction, currentTestData.query)
			if contractErr != nil {
				t.Fatal("contractErr: ", contractErr)
			}

			actualSQL := transaction.Select("ID").Find(&models.Run{}).Statement.SQL.String()
			assert.Equal(t, removeWhitespace(testData.expectedSQL), removeWhitespace(actualSQL))
			assert.Equal(t, testData.expectedVars, transaction.Statement.Vars)
		})
	}
}
