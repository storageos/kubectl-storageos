package installer

import (
	"reflect"
	"testing"
)

func TestEndpointsSplitter(t *testing.T) {
	tcases := []struct {
		name         string
		endpoints    string
		expEndpoints string
	}{
		{
			name:         "multiple IPs",
			endpoints:    "1.2.3.4:2379,5.6.7.8:2379",
			expEndpoints: "\"http://1.2.3.4:2379,http://5.6.7.8:2379\"",
		},
		{
			name:         "single IP",
			endpoints:    "1.2.3.4:2379",
			expEndpoints: "\"http://1.2.3.4:2379\"",
		},
		{
			name:         "domain",
			endpoints:    "storageos.default:2379",
			expEndpoints: "\"http://storageos.default:2379\"",
		},
		{
			name:         "multiple domains",
			endpoints:    "storageos.default:2379,storageos.system:2379",
			expEndpoints: "\"http://storageos.default:2379,http://storageos.system:2379\"",
		},
	}
	for _, tc := range tcases {
		endpoints := endpointsSplitter(tc.endpoints)
		if endpoints != tc.expEndpoints {
			t.Errorf("expected %s, got %s", tc.expEndpoints, endpoints)
		}
	}
}

func TestEtcdctlMemberList(t *testing.T) {
	tcases := []struct {
		name      string
		endpoints string
		cmd       []string
	}{
		{
			name:      "test 1",
			endpoints: "\"http://1.2.3.4:2379,http://5.6.7.8:2379\"",
			cmd: []string{
				"/bin/bash",
				"-c",
				"etcdctl --endpoints \"http://1.2.3.4:2379,http://5.6.7.8:2379\" member list",
			},
		},
		{
			name:      "test 2",
			endpoints: "\"http://1.2.3.4:2379\"",
			cmd: []string{
				"/bin/bash",
				"-c",
				"etcdctl --endpoints \"http://1.2.3.4:2379\" member list",
			},
		},
	}
	for _, tc := range tcases {
		cmd := etcdctlMemberListCmd(tc.endpoints)
		if !reflect.DeepEqual(cmd, tc.cmd) {
			t.Errorf("expected %v, got %v", tc.cmd, cmd)
		}
	}
}

func TestEtcdctlPutCmd(t *testing.T) {
	tcases := []struct {
		name      string
		endpoints string
		key       string
		value     string
		cmd       []string
	}{
		{
			name:      "test 1",
			endpoints: "\"http://1.2.3.4:2379,http://5.6.7.8:2379\"",
			key:       "foo",
			value:     "bar",
			cmd: []string{
				"/bin/bash",
				"-c",
				"etcdctl --endpoints \"http://1.2.3.4:2379,http://5.6.7.8:2379\" put foo bar",
			},
		},
		{
			name:      "test 2",
			endpoints: "\"http://1.2.3.4:2379\"",
			key:       "test-key",
			value:     "test-val",
			cmd: []string{
				"/bin/bash",
				"-c",
				"etcdctl --endpoints \"http://1.2.3.4:2379\" put test-key test-val",
			},
		},
	}
	for _, tc := range tcases {
		cmd := etcdctlPutCmd(tc.endpoints, tc.key, tc.value)
		if !reflect.DeepEqual(cmd, tc.cmd) {
			t.Errorf("expected %v, got %v", tc.cmd, cmd)
		}
	}
}

func TestEtcdctlGetCmd(t *testing.T) {
	tcases := []struct {
		name      string
		endpoints string
		key       string
		cmd       []string
	}{
		{
			name:      "test 1",
			endpoints: "\"http://1.2.3.4:2379,http://5.6.7.8:2379\"",
			key:       "foo",
			cmd: []string{
				"/bin/bash",
				"-c",
				"etcdctl --endpoints \"http://1.2.3.4:2379,http://5.6.7.8:2379\" get foo",
			},
		},
		{
			name:      "test 2",
			endpoints: "\"http://1.2.3.4:2379\"",
			key:       "test-key",
			cmd: []string{
				"/bin/bash",
				"-c",
				"etcdctl --endpoints \"http://1.2.3.4:2379\" get test-key",
			},
		},
	}
	for _, tc := range tcases {
		cmd := etcdctlGetCmd(tc.endpoints, tc.key)
		if !reflect.DeepEqual(cmd, tc.cmd) {
			t.Errorf("expected %v, got %v", tc.cmd, cmd)
		}
	}
}

func TestEtcdctlDelCmd(t *testing.T) {
	tcases := []struct {
		name      string
		endpoints string
		key       string
		cmd       []string
	}{
		{
			name:      "test 1",
			endpoints: "\"http://1.2.3.4:2379,http://5.6.7.8:2379\"",
			key:       "foo",
			cmd: []string{
				"/bin/bash",
				"-c",
				"etcdctl --endpoints \"http://1.2.3.4:2379,http://5.6.7.8:2379\" del foo",
			},
		},
		{
			name:      "test 2",
			endpoints: "\"http://1.2.3.4:2379\"",
			key:       "test-key",
			cmd: []string{
				"/bin/bash",
				"-c",
				"etcdctl --endpoints \"http://1.2.3.4:2379\" del test-key",
			},
		},
	}
	for _, tc := range tcases {
		cmd := etcdctlDelCmd(tc.endpoints, tc.key)
		if !reflect.DeepEqual(cmd, tc.cmd) {
			t.Errorf("expected %v, got %v", tc.cmd, cmd)
		}
	}
}
