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
		envKey,
		envValue,
		defaultValue,
		expected string
	}{
		{
			"env-set",
			pseudoRandomString(),
			"some-env-value",
			"some-default",
			"some-env-value",
		},
		{
			"env-set-but-no-default",
			pseudoRandomString(),
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
			if tCase.envKey != "" {
				os.Setenv(tCase.envKey, tCase.expected)
			}
			actual := getEnvOrDefault(tCase.envKey, tCase.defaultValue)
			if actual != tCase.expected {
				t.Fatalf("Expected %q but was %q", tCase.expected, actual)
			}
		})
	}
}

func pseudoRandomString() string {
	return strconv.FormatInt(time.Now().UnixNano(), 16)
}
