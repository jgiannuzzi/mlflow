package database

import (
	"fmt"
	"log"
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

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
	db, err = gorm.Open(postgres.Open(databaseUrl), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
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
	sqlDb, err := db.DB()
	if err != nil {
		log.Fatalln("Failed to get database connection:", err)
	}
	dbx = sqlx.NewDb(sqlDb, "pgx")
	dbx.Mapper = reflectx.NewMapperFunc("json", nil)
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

				if v.input <= 1000 {
					_, err = tx.NamedExec(query, metrics)
					if err != nil {
						tx.Rollback() // Roll back in case of error
						log.Fatalln("Failed to execute insert:", err)
					}
				} else {
					// Process in batches
					batchSize := 1000

					for i := 0; i < len(metrics); i += batchSize {
						end := i + batchSize
						if end > len(metrics) {
							end = len(metrics)
						}
						batch := metrics[i:end]
						_, err = tx.NamedExec(query, batch)
						if err != nil {
							tx.Rollback() // Roll back in case of error
							log.Fatalln("Failed to execute insert:", err)
						}
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

func BenchmarkSelectMetrics(b *testing.B) {
	for _, v := range inputs {
		b.Run(fmt.Sprintf("GORM rows input_size_%d", v.input), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				metrics := make([]model.Metric, 0, v.input)

				rows, err := db.Limit(v.input).Model(metrics).Rows()
				if err != nil {
					log.Fatalf("Query failed: %v", err)
				}
				defer rows.Close()

				for rows.Next() {
					var metric model.Metric
					rows.Scan(&metric.Key, &metric.Value, &metric.Timestamp, &metric.RunUUID, &metric.Step, &metric.IsNan)
					if err := db.ScanRows(rows, &metric); err != nil {
						log.Fatalf("Failed to scan row: %v", err)
					}
					metrics = append(metrics, metric)
				}

				if len(metrics) != v.input {
					log.Fatalf("Expected %d metrics, got %d", v.input, len(metrics))
				}
			}
		})

		b.Run(fmt.Sprintf("GORM slice input_size_%d", v.input), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				metrics := make([]model.Metric, v.input)

				if err := db.Limit(v.input).Find(&metrics).Error; err != nil {
					log.Fatalf("Query failed: %v", err)
				}

				if len(metrics) != v.input {
					log.Fatalf("Expected %d metrics, got %d", v.input, len(metrics))
				}
			}
		})

		// This does not work with metrics because of the composite primary key
		// b.Run(fmt.Sprintf("GORM batch input_size_%d", v.input), func(b *testing.B) {
		// 	for n := 0; n < b.N; n++ {
		// 		metrics := make([]model.Metric, v.input)
		// 		const batchSize = 1000
		// 		batch := make([]model.Metric, batchSize)
		// 		if err := db.Limit(v.input).FindInBatches(&batch, batchSize, func(tx *gorm.DB, batch int) error {
		// 			return nil
		// 		}).Error; err != nil {
		// 			log.Fatalf("Query failed: %v", err)
		// 		}

		// 		if len(metrics) != v.input {
		// 			log.Fatalf("Expected %d metrics, got %d", v.input, len(metrics))
		// 		}
		// 	}
		// })

		b.Run(fmt.Sprintf("SQLX rows input_size_%d", v.input), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				metrics := make([]model.Metric, 0, v.input)

				rows, err := dbx.Queryx("SELECT * FROM metrics LIMIT $1", v.input)
				if err != nil {
					log.Fatalf("Query failed: %v", err)
				}
				defer rows.Close()

				for rows.Next() {
					var metric model.Metric
					if err := rows.StructScan(&metric); err != nil {
						log.Fatalf("Failed to scan row: %v", err)
					}
					metrics = append(metrics, metric)
				}

				if len(metrics) != v.input {
					log.Fatalf("Expected %d metrics, got %d", v.input, len(metrics))
				}
			}
		})

		b.Run(fmt.Sprintf("SQLX slice input_size_%d", v.input), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				metrics := make([]model.Metric, v.input)

				dbx.Select(&metrics, "SELECT * FROM metrics LIMIT $1", v.input)

				if len(metrics) != v.input {
					log.Fatalf("Expected %d metrics, got %d", v.input, len(metrics))
				}
			}
		})

		b.Run(fmt.Sprintf("GORM rows with SQLX input_size_%d", v.input), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				metrics := make([]model.Metric, 0, v.input)

				stmt := db.Session(&gorm.Session{DryRun: true}).Limit(v.input).Find(&metrics).Statement
				rows, err := dbx.Queryx(stmt.SQL.String(), stmt.Vars...)
				if err != nil {
					log.Fatalf("Query failed: %v", err)
				}
				defer rows.Close()

				for rows.Next() {
					var metric model.Metric
					if err := rows.StructScan(&metric); err != nil {
						log.Fatalf("Failed to scan row: %v", err)
					}
					metrics = append(metrics, metric)
				}

				if len(metrics) != v.input {
					log.Fatalf("Expected %d metrics, got %d", v.input, len(metrics))
				}
			}
		})
	}
}
