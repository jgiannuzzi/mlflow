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

func init() {
	var err error
	db, err = gorm.Open(postgres.Open("postgresql://postgres:postgres@localhost/postgres"))
	if err != nil {
		panic(err)
	}
}

func BenchmarkInsertExperiment(b *testing.B) {
	// Run the benchmark
	for n := 0; n < b.N; n++ {
		// Generate a new experiment
		exp := generateExperiment(b)

		if err := db.Create(exp).Error; err != nil {
			b.Fatalf("Failed to insert experiment: %v", err)
		}
	}
}

func BenchmarkInsertManyExperiments(b *testing.B) {
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
				// Generate new experiments
				exp := generateExperiments(b, v.input)

				if err := db.CreateInBatches(&exp, 1000).Error; err != nil {
					b.Fatalf("Failed to insert experiment: %v", err)
				}
			}
		})
	}
}

func generateExperiment(b *testing.B) *model.Experiment {
	b.Helper()

	var exp model.Experiment
	if err := faker.FakeData(&exp); err != nil {
		panic(err)
	}
	exp.LifecycleStage = "active"
	exp.ExperimentID = 0

	return &exp
}

func generateExperiments(b *testing.B, n int) []*model.Experiment {
	b.Helper()

	exps := make([]*model.Experiment, n)
	for i := 0; i < n; i++ {
		exp := generateExperiment(b)
		exps[i] = exp
	}

	return exps
}
