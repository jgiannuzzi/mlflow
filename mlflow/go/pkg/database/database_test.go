package database

import (
	"fmt"
	"log"
	"testing"

	"github.com/go-faker/faker/v4"
	// Fight me
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	"github.com/jmoiron/sqlx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mlflow/mlflow/mlflow/go/pkg/database/model"
)

var (
	db    *gorm.DB
	dbx   *sqlx.DB
	runId string
)

func init() {
	var err error
	databaseUrl := "postgresql://postgres:postgres@localhost/postgres"
	db, err = gorm.Open(postgres.Open(databaseUrl))
	if err != nil {
		panic(err)
	}

	// Create experiment
	var experiment model.Experiment
	if err := faker.FakeData(&experiment); err != nil {
		panic(err)
	}
	experiment.LifecycleStage = "active"
	experiment.ExperimentID = 0

	if err := db.Create(&experiment).Error; err != nil {
		panic(fmt.Errorf("Failed to insert experiment: %v", err))
	}

	// Create run
	var run model.Run
	if err := faker.FakeData(&run); err != nil {
		panic(err)
	}
	run.SourceType = "LOCAL"
	run.LifecycleStage = "active"
	run.Status = "RUNNING"

	// linked to the experiment
	run.ExperimentID = experiment.ExperimentID

	if err := db.Create(&run).Error; err != nil {
		panic(fmt.Errorf("Failed to insert run: %v", err))
	}

	// Metrics need a link to a run
	runId = run.RunUUID

	// SQLX
	dbx, err = sqlx.Connect("pgx", databaseUrl)
	if err != nil {
		log.Fatalln(err)
	}
}

type RunInput struct {
	input int
}

var inputs = []RunInput{
	{input: 1},
	{input: 10},
	{input: 100},
	{input: 1000},
	{input: 10000},
	{input: 100000},
}

func BenchmarkInsertMetrics(b *testing.B) {
	for _, v := range inputs {
		b.Run(fmt.Sprintf("GORM input_size_%d", v.input), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				// Generate new metrics
				metrics := generateMetrics(b, v.input)

				if err := db.CreateInBatches(&metrics, 1000).Error; err != nil {
					b.Fatalf("Failed to insert batch metrics: %v", err)
				}

			}
		})

		b.Run(fmt.Sprintf("SQLX input_size_%d", v.input), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				// Start transaction
				tx, err := dbx.Beginx()
				if err != nil {
					log.Fatalln("Failed to start transaction:", err)
				}

				// Generate new metrics
				metrics := generateMetrics(b, v.input)
				query := "INSERT INTO metrics (key, value, timestamp, run_uuid, step, is_nan) VALUES (:key, :value, :timestamp, :run_uuid, :step, :is_nan)"

				// Insert each metric individually
				for _, metric := range metrics {
					paramMap := map[string]interface{}{
						"key":       metric.Key,
						"value":     metric.Value,
						"timestamp": metric.Timestamp,
						"run_uuid":  metric.RunUUID,
						"step":      metric.Step,
						"is_nan":    metric.IsNan,
					}
					_, err := tx.NamedExec(query, paramMap)
					if err != nil {
						tx.Rollback() // Roll back in case of error
						log.Fatalln("Failed to execute insert:", err)
					}
				}

				// Commit transaction
				if err := tx.Commit(); err != nil {
					log.Fatalln("Failed to commit transaction:", err)
				}

			}
		})
	}
}

func generateMetric(b *testing.B) *model.Metric {
	b.Helper()

	var metric model.Metric
	if err := faker.FakeData(&metric); err != nil {
		panic(err)
	}
	metric.RunUUID = runId

	return &metric
}

func generateMetrics(b *testing.B, n int) []*model.Metric {
	b.Helper()

	metrics := make([]*model.Metric, n)
	for i := 0; i < n; i++ {
		metric := generateMetric(b)
		metrics[i] = metric
	}

	return metrics
}

func BenchmarkSelectMetrics_Gorm(b *testing.B) {
	for _, v := range inputs {
		b.Run(fmt.Sprintf("GORM input_size_%d", v.input), func(b *testing.B) {
			n := v.input
			var metrics []*model.Metric
			result := db.Limit(n).Find(&metrics)
			if result.Error != nil {
				log.Fatalf("Query failed: %v", result.Error)
			}
		})

		b.Run(fmt.Sprintf("SQLX input_size_%d", v.input), func(b *testing.B) {
			n := v.input
			metrics := make([]*model.Metric, n)

			rows, err := dbx.Queryx(fmt.Sprintf("select key, value, timestamp, run_uuid, step, is_nan from metrics LIMIT %d", n))
			if err != nil {
				log.Fatalf("Query failed: %v", err)
			}
			defer rows.Close()

			idx := 0

			for rows.Next() {
				var metric model.Metric
				err = rows.StructScan(&metric)
				metrics[idx] = &metric
				idx += 1
			}
		})
	}
}
