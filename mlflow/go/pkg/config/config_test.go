package config_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mlflow/mlflow/mlflow/go/pkg/config"
)

func TestValidDuration(t *testing.T) {
	t.Parallel()

	samples := []string{
		"1000",
		`"1s"`,
		`"2h45m"`,
	}

	for _, sample := range samples {
		currentSample := sample
		t.Run(currentSample, func(t *testing.T) {
			t.Parallel()

			jsonConfig := fmt.Sprintf(`{ "shutdownTimeout": %s }`, currentSample)

			var cfg config.Config

			if err := json.Unmarshal([]byte(jsonConfig), &cfg); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInvalidDuration(t *testing.T) {
	t.Parallel()

	var cfg config.Config

	if err := json.Unmarshal([]byte(`{ "shutdownTimeout": "two seconds" }`), &cfg); err == nil {
		t.Error("expected error")
	}
}
