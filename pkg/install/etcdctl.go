package install

import (
	"fmt"
	"strings"
)

// etcdctlMemberList returns a slice of strings representing the etcdctl command for members list to
// be interpreted by the pod exec:
// {`/bin/bash`, `-c`, `etcdctl --endpoints "http://<endpoints>" member list`}
func etcdctlMemberListCmd(endpoints string) []string {
	return []string{"etcdctl", "--endpoints", endpoints, "member", "list"}
}

// etcdctlPutCmd returns a slice of strings representing the etcdctl command for a simple write to
// be interpreted by the pod exec:
// {`/bin/bash`, `-c`, `etcdctl --endpoints "http://<endpoints>" put foo bar`}
func etcdctlPutCmd(endpoints, key, value string) []string {
	return []string{"etcdctl", "--endpoints", endpoints, "put", key, value}

}

// etcdctlGetCmd returns a slice of strings representing the etcdctl command for a simple read to
// be interpreted by the pod exec:
// {`/bin/bash`, `-c`, `etcdctl --endpoints "http://<endpoints>" get foo`}
func etcdctlGetCmd(endpoints, key string) []string {
	return []string{"etcdctl", "--endpoints", endpoints, "get", key}

}

// etcdctlDelCmd returns a slice of strings representing the etcdctl command for a simple delete to
// be interpreted by the pod exec:
// {`/bin/bash`, `-c`, `etcdctl --endpoints "http://<endpoints>" del foo`}
func etcdctlDelCmd(endpoints, key string) []string {
	return []string{"etcdctl", "--endpoints", endpoints, "del", key}

}

// endpointsSplitter takes endpoints input from user prompt and returns digestable string for etcdctl
// Example:
// input: 1.2.3.4:2379,4.5.6.7:2379
// output: "http://1.2.3.4:2379,http://4.5.6.7:2379"
func endpointsSplitter(endpoints string) string {
	endpointsSlice := strings.Split(endpoints, ",")
	httpEndpointsSlice := make([]string, 0)
	for _, endpoint := range endpointsSlice {
		httpEndpointsSlice = append(httpEndpointsSlice, fmt.Sprintf("%s%s", "http://", endpoint))
	}

	return strings.Join([]string{"\"", strings.Join(httpEndpointsSlice, ","), "\""}, "")
}
