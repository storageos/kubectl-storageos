package installer

import (
	"reflect"
	"testing"
)

func TestEndpointsSplitter(t *testing.T) {
	tcases := []struct {
		name         string
		endpoints    string
		tls          bool
		expEndpoints []string
	}{
		{
			name:         "multiple IPs",
			endpoints:    "1.2.3.4:2379,5.6.7.8:2379",
			tls:          false,
			expEndpoints: []string{"http://1.2.3.4:2379", "http://5.6.7.8:2379"},
		},
		{
			name:         "single IP",
			endpoints:    "1.2.3.4:2379",
			tls:          false,
			expEndpoints: []string{"http://1.2.3.4:2379"},
		},
		{
			name:         "domain",
			endpoints:    "storageos.default:2379",
			tls:          false,
			expEndpoints: []string{"http://storageos.default:2379"},
		},
		{
			name:         "multiple domains",
			endpoints:    "storageos.default:2379,storageos.system:2379",
			tls:          false,
			expEndpoints: []string{"http://storageos.default:2379", "http://storageos.system:2379"},
		},
		{
			name:         "multiple domains with prefix",
			endpoints:    "http://storageos.default:2379,http://storageos.system:2379",
			tls:          false,
			expEndpoints: []string{"http://storageos.default:2379", "http://storageos.system:2379"},
		},
		{
			name:         "multiple IPs TLS",
			endpoints:    "1.2.3.4:2379,5.6.7.8:2379",
			tls:          true,
			expEndpoints: []string{"https://1.2.3.4:2379", "https://5.6.7.8:2379"},
		},
		{
			name:         "single IP TLS",
			endpoints:    "1.2.3.4:2379",
			tls:          true,
			expEndpoints: []string{"https://1.2.3.4:2379"},
		},
		{
			name:         "domain TLS",
			endpoints:    "storageos.default:2379",
			tls:          true,
			expEndpoints: []string{"https://storageos.default:2379"},
		},
		{
			name:         "multiple domains TLS",
			endpoints:    "storageos.default:2379,storageos.system:2379",
			tls:          true,
			expEndpoints: []string{"https://storageos.default:2379", "https://storageos.system:2379"},
		},
		{
			name:         "multiple domains with prefix TLS",
			endpoints:    "https://storageos.default:2379,https://storageos.system:2379",
			tls:          true,
			expEndpoints: []string{"https://storageos.default:2379", "https://storageos.system:2379"},
		},
	}
	for _, tc := range tcases {
		endpoints := endpointsSplitter(tc.endpoints, tc.tls)
		if !reflect.DeepEqual(endpoints, tc.expEndpoints) {
			t.Errorf("expected %s, got %s", tc.expEndpoints, endpoints)
		}
	}
}

func TestEtcdctlMemberList(t *testing.T) {
	tcases := []struct {
		name      string
		endpoints string
		tls       bool
		cmd       []string
	}{
		{
			name:      "member list test 1",
			endpoints: "http://1.2.3.4:2379",
			tls:       false,
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"http://1.2.3.4:2379",
				"member",
				"list",
			},
		},
		{
			name:      "member list test 2",
			endpoints: "http://1.2.3.4:2379",
			tls:       false,
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"http://1.2.3.4:2379",
				"member",
				"list",
			},
		},
		{
			name:      "member list test 3 tls",
			endpoints: "https://1.2.3.4:2379",
			tls:       true,
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"https://1.2.3.4:2379",
				"--key",
				keyPath,
				"--cert",
				certPath,
				"--cacert",
				caCertPath,
				"member",
				"list",
			},
		},
		{
			name:      "member list test 4 tls",
			endpoints: "https://1.2.3.4:2379",
			tls:       true,
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"https://1.2.3.4:2379",
				"--key",
				keyPath,
				"--cert",
				certPath,
				"--cacert",
				caCertPath,
				"member",
				"list",
			},
		},
	}
	for _, tc := range tcases {
		cmd := etcdctlMemberListCmd(tc.endpoints, tc.tls)
		if !reflect.DeepEqual(cmd, tc.cmd) {
			t.Errorf("expected %v, got %v", tc.cmd, cmd)
		}
	}
}

func TestEtcdctlPutCmd(t *testing.T) {
	tcases := []struct {
		name      string
		endpoints string
		tls       bool
		key       string
		value     string
		cmd       []string
	}{
		{
			name:      "put test 1",
			endpoints: "http://1.2.3.4:2379",
			tls:       false,
			key:       "foo",
			value:     "bar",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"http://1.2.3.4:2379",
				"put",
				"foo",
				"bar",
			},
		},
		{
			name:      "put test 2",
			endpoints: "http://1.2.3.4:2379",
			tls:       false,
			key:       "test-key",
			value:     "test-val",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"http://1.2.3.4:2379",
				"put",
				"test-key",
				"test-val",
			},
		},
		{
			name:      "put test 3 tls",
			endpoints: "https://1.2.3.4:2379",
			tls:       true,
			key:       "foo",
			value:     "bar",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"https://1.2.3.4:2379",
				"--key",
				keyPath,
				"--cert",
				certPath,
				"--cacert",
				caCertPath,
				"put",
				"foo",
				"bar",
			},
		},
		{
			name:      "put test 4 tls",
			endpoints: "https://1.2.3.4:2379",
			tls:       true,
			key:       "test-key",
			value:     "test-val",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"https://1.2.3.4:2379",
				"--key",
				keyPath,
				"--cert",
				certPath,
				"--cacert",
				caCertPath,
				"put",
				"test-key",
				"test-val",
			},
		},
	}
	for _, tc := range tcases {
		cmd := etcdctlPutCmd(tc.endpoints, tc.key, tc.value, tc.tls)
		if !reflect.DeepEqual(cmd, tc.cmd) {
			t.Errorf("expected %v, got %v", tc.cmd, cmd)
		}
	}
}

func TestEtcdctlGetCmd(t *testing.T) {
	tcases := []struct {
		name      string
		endpoints string
		tls       bool
		key       string
		cmd       []string
	}{
		{
			name:      "test 1",
			endpoints: "http://1.2.3.4:2379",
			tls:       false,
			key:       "foo",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"http://1.2.3.4:2379",
				"get",
				"foo",
			},
		},
		{
			name:      "test 2",
			endpoints: "http://1.2.3.4:2379",
			tls:       false,
			key:       "test-key",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"http://1.2.3.4:2379",
				"get",
				"test-key",
			},
		},
		{
			name:      "get test 3 tls",
			endpoints: "https://1.2.3.4:2379",
			tls:       true,
			key:       "foo",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"https://1.2.3.4:2379",
				"--key",
				keyPath,
				"--cert",
				certPath,
				"--cacert",
				caCertPath,
				"get",
				"foo",
			},
		},
		{
			name:      "get test 4 tls",
			endpoints: "https://1.2.3.4:2379",
			tls:       true,
			key:       "test-key",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"https://1.2.3.4:2379",
				"--key",
				keyPath,
				"--cert",
				certPath,
				"--cacert",
				caCertPath,
				"get",
				"test-key",
			},
		},
	}
	for _, tc := range tcases {
		cmd := etcdctlGetCmd(tc.endpoints, tc.key, tc.tls)
		if !reflect.DeepEqual(cmd, tc.cmd) {
			t.Errorf("expected %v, got %v", tc.cmd, cmd)
		}
	}
}

func TestEtcdctlDelCmd(t *testing.T) {
	tcases := []struct {
		name      string
		endpoints string
		tls       bool
		key       string
		cmd       []string
	}{
		{
			name:      "del test 1",
			endpoints: "http://1.2.3.4:2379",
			tls:       false,
			key:       "foo",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"http://1.2.3.4:2379",
				"del",
				"foo",
			},
		},
		{
			name:      "del test 2",
			endpoints: "http://1.2.3.4:2379",
			tls:       false,
			key:       "test-key",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"http://1.2.3.4:2379",
				"del",
				"test-key",
			},
		},
		{
			name:      "del test 3 tls",
			endpoints: "https://1.2.3.4:2379",
			tls:       true,
			key:       "foo",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"https://1.2.3.4:2379",
				"--key",
				keyPath,
				"--cert",
				certPath,
				"--cacert",
				caCertPath,
				"del",
				"foo",
			},
		},
		{
			name:      "del test 4 tls",
			endpoints: "https://1.2.3.4:2379",
			tls:       true,
			key:       "test-key",
			cmd: []string{
				"etcdctl",
				"--endpoints",
				"https://1.2.3.4:2379",
				"--key",
				keyPath,
				"--cert",
				certPath,
				"--cacert",
				caCertPath,
				"del",
				"test-key",
			},
		},
	}
	for _, tc := range tcases {
		cmd := etcdctlDelCmd(tc.endpoints, tc.key, tc.tls)
		if !reflect.DeepEqual(cmd, tc.cmd) {
			t.Errorf("expected %v, got %v", tc.cmd, cmd)
		}
	}
}
