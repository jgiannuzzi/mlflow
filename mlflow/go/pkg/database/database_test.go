package database

import (
	"fmt"
	"testing"

	"github.com/go-faker/faker/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mlflow/mlflow/mlflow/go/pkg/database/model"
)

var db *gorm.DB
var runId string

func init() {
	var err error
	db, err = gorm.Open(postgres.Open("postgresql://postgres:postgres@localhost/postgres"))
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
}

func BenchmarkInsertMetric(b *testing.B) {
	// Run the benchmark
	for n := 0; n < b.N; n++ {
		// Generate a new metric
		metric := generateMetric(b)

		if err := db.Create(metric).Error; err != nil {
			b.Fatalf("Failed to insert experiment: %v", err)
		}
	}
}

func BenchmarkInsertManyMetrics(b *testing.B) {
	for _, v := range []struct {
		input int
	}{
		{input: 10},
		{input: 100},
		{input: 1000},
		{input: 10000},
		{input: 100000},
	} {
		b.Run(fmt.Sprintf("input_size_%d", v.input), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				// Generate new metrics
				metrics := generateMetrics(b, v.input)

				if err := db.CreateInBatches(&metrics, 1000).Error; err != nil {
					b.Fatalf("Failed to insert batch metrics: %v", err)
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
