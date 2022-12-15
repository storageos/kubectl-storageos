package cmd

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tcases := []struct {
		name    string
		args    []string
		expArgs []string
	}{
		{
			name:    "no equal signs in args",
			args:    []string{"--namespace", "test-ns", "-o", "yaml", "--password", "my-pass", "--username", "my-user"},
			expArgs: []string{"--namespace", "test-ns", "-o", "yaml", "--password", "my-pass", "--username", "my-user"},
		},
		{
			name:    "all equal signs in args",
			args:    []string{"--namespace=test-ns", "-o=yaml", "--password=my-pass", "--username=my-user"},
			expArgs: []string{"--namespace", "test-ns", "-o", "yaml", "--password", "my-pass", "--username", "my-user"},
		},
		{
			name:    "mixed format in args",
			args:    []string{"--namespace=test-ns", "-oyaml", "--password", "my-pass", "--username", "my-user"},
			expArgs: []string{"--namespace", "test-ns", "-oyaml", "--password", "my-pass", "--username", "my-user"},
		},
	}
	for _, tc := range tcases {
		expArgs := parseArgs(tc.args)
		if !reflect.DeepEqual(expArgs, tc.expArgs) {
			t.Errorf("case: %s - expected %v, got %v", tc.name, tc.expArgs, expArgs)
		}
	}
}
