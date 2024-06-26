package utils

import (
	"fmt"
	"os"
	"strings"
)

const MLFlowTruncateLongValues = "MLFLOW_TRUNCATE_LONG_VALUES"

type EnvironmentError struct {
	message string
}

func newEnvironmentError(format string, a ...any) *EnvironmentError {
	return &EnvironmentError{message: fmt.Sprintf(format, a...)}
}

func (e *EnvironmentError) Error() string {
	return e.message
}

func ReadTruncateLongValuesEnvironmentVariable() (bool, error) {
	truncateLongValues := strings.ToLower(os.Getenv(MLFlowTruncateLongValues))

	switch truncateLongValues {
	// value is not set, or true.
	case "", "true", "1":
		return true, nil
	// explicit false
	case "false", "0":
		return false, nil
	default:
		return false, newEnvironmentError(
			MLFlowTruncateLongValues +
				" value must be one of ['true', 'false', '1', '0'] (case-insensitive), but got " +
				truncateLongValues,
		)
	}
}
