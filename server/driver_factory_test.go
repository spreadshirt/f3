package server

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
			featureChangeDir | featureList | featureRemoveDir | featureRemove | featureMove | featureMakeDir | featureGet | featurePut,
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
		if err != nil && testData.shouldFail {
			continue
		}
		if err != nil && !testData.shouldFail {
			t.Errorf("Test %q failed: %s", testData.id, err)
			continue
		}
		if flags != testData.featureFlags {
			expected := strconv.FormatInt(int64(testData.featureFlags), 2)
			result := strconv.FormatInt(int64(flags), 2)
			t.Errorf("Test %s: Feature set %q parsed incorrectly: expected/result %s != %s", testData.id, testData.featureSet, expected, result)
		}
	}
}

func TestDriverFactory(t *testing.T) {
	testDataSet := []struct {
		config     FactoryConfig
		bucketName string
		id         string
		shouldFail bool
	}{
		{
			FactoryConfig{},
			"",
			"empty-config",
			true,
		},
		{
			FactoryConfig{
				DefaultFeatureSet,
				false,
				"access:secret",
				"https://some-bucket.somewhere.com",
				DefaultRegion,
				true,
				true,
			},
			"some-bucket",
			"valid-minimal-config",
			false,
		},
		{
			FactoryConfig{
				"ls,rm,mkdir,get",
				false,
				"access:secret",
				"https://another-bucket.somewhere.in.some.datacenter.domain.com",
				"us-east-1",
				false,
				true,
			},
			"another-bucket",
			"valid-config",
			false,
		},
	}
	for _, testData := range testDataSet {
		factory, err := NewDriverFactory(&testData.config)
		if err != nil && testData.shouldFail {
			continue
		}
		if err != nil && !testData.shouldFail {
			t.Errorf("Test %q failed: %s", testData.id, err)
			continue
		}
		if factory.bucketName != testData.bucketName {
			t.Errorf("Test %s: bad bucket name %q, expected %q", testData.id, factory.bucketName, testData.bucketName)
		}
		if factory.featureFlags == 0 {
			t.Errorf("Test %s: Empty feature set", testData.id)
		}
	}
}
