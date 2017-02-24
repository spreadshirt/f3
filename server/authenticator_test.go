package server

import (
	"testing"
)

func TestAuthenticatorFromString(t *testing.T) {
	testDataSet := []struct {
		id         string
		raw        string
		parsed     map[string]string
		shouldFail bool
	}{
		{
			"empty",
			"",
			map[string]string{},
			true,
		},
		{
			"valid",
			"foo:bar\nbaz:ban",
			map[string]string{
				"foo": "bar",
				"baz": "ban",
			}, false,
		},
		{
			"partially-valid",
			"Bruce:Willis\nSomethingWhatever\nJefferson:Airplane",
			map[string]string{
				"Bruce":     "Willis",
				"Jefferson": "Airplane",
			}, false,
		},
		{
			"empty-lines",
			"\nAaron:Funk\n\n\n",
			map[string]string{
				"Aaron": "Funk",
			}, false,
		},
	}
	for _, testData := range testDataSet {
		auth, err := AuthenticatorFromString(testData.raw)
		if err == nil && testData.shouldFail {
			t.Errorf("Test %s: should fail but succeeded", testData.id)
			continue
		}
		for user, pass := range testData.parsed {
			valid, _ := auth.CheckPasswd(user, pass)
			if !valid {
				t.Errorf("Test %s: credentials %s:%s could not be validated", testData.id, user, pass)
			}
		}
	}
}
