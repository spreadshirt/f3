package ftplib

import (
	"strconv"
	"testing"
)

func TestParseFeatureSet(t *testing.T) {
	testDataSet := []struct {
		id           string
		featureSet   string
		featureFlags int
		shouldFail   bool
	}{
		{
			"empty",
			"",
			0,
			true,
		},
		{
			"full-set",
			"cd,ls,rmdir,rm,mv,mkdir,get,put",
			F_CD | F_LS | F_RMDIR | F_RM | F_MV | F_MKDIR | F_GET | F_PUT,
			false,
		},
		{
			"invalid-features",
			"cd,invalid,put",
			0,
			true,
		},
		{
			"bad-syntax",
			"cd,get,rmdir,",
			0,
			true,
		},
	}
	for _, testData := range testDataSet {
		flags, err := parseFeatureSet(testData.featureSet)
		if err == nil && testData.shouldFail {
			t.Errorf("Test %s: should fail but succeeded", testData.id)
			continue
		}
		if flags != testData.featureFlags {
			expected := strconv.FormatInt(int64(testData.featureFlags), 2)
			result := strconv.FormatInt(int64(flags), 2)
			t.Errorf("Test %s: Feature set %q parsed incorrectly: expected/result %s != %s", testData.id, testData.featureSet, expected, result)
		}
	}
}
