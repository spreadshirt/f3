package main

import (
	"os"
	"strconv"
	"testing"
	"time"
)

func TestGetEnvOrDefault(t *testing.T) {
	tCases := []struct {
		name,
		key,
		envValue,
		defaultValue,
		expected string
	}{
		{
			"env-set",
			strconv.FormatInt(time.Now().UnixNano(), 16),
			"some-env-value",
			"some-default",
			"some-env-value",
		},
		{
			"env-set-but-no-default",
			strconv.FormatInt(time.Now().UnixNano(), 16),
			"some-env-value",
			"",
			"some-env-value",
		},
		{
			"env-not-set-but-default",
			"",
			"",
			"some-default",
			"some-default",
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			if tCase.key != "" {
				os.Setenv(tCase.key, tCase.expected)
			}
			actual := getEnvOrDefault(tCase.key, tCase.defaultValue)
			if actual != tCase.expected {
				t.Fatalf("Expected %q but was %q", tCase.expected, actual)
			}
		})
	}
}
