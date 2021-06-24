package install

import (
	"fmt"
	"strings"
)

// etcdctlMemberList returns a slice of strings representing the etcdctl command for members list to
// be interpreted by the pod exec:
// {`/bin/bash`, `-c`, `etcdctl --endpoints "http://<endpoints>" member list`}
func etcdctlMemberListCmd(endpoints string) []string {
	return []string{"/bin/bash", "-c", fmt.Sprintf("%s%s%s", "etcdctl --endpoints ", endpoints, " member list")}
}

// etcdctlSetCmd returns a slice of strings representing the etcdctl command for a simple write to
// be interpreted by the pod exec:
// {`/bin/bash`, `-c`, `etcdctl --endpoints "http://<endpoints>" set foo bar`}
func etcdctlSetCmd(endpoints, key, value string) []string {
	return []string{"/bin/bash", "-c", fmt.Sprintf("%s%s%s%s%s%s", "etcdctl --endpoints ", endpoints, " set ", key, " ", value)}
}

// etcdctlGetCmd returns a slice of strings representing the etcdctl command for a simple read to
// be interpreted by the pod exec:
// {`/bin/bash`, `-c`, `etcdctl --endpoints "http://<endpoints>" get foo bar`}
func etcdctlGetCmd(endpoints, key string) []string {
	return []string{"/bin/bash", "-c", fmt.Sprintf("%s%s%s%s", "etcdctl --endpoints ", endpoints, " get ", key)}
}

// etcdctlRmCmd returns a slice of strings representing the etcdctl command for a simple delete to
// be interpreted by the pod exec:
// {`/bin/bash`, `-c`, `etcdctl --endpoints "http://<endpoints>" rm foo bar`}
func etcdctlRmCmd(endpoints, key string) []string {
	return []string{"/bin/bash", "-c", fmt.Sprintf("%s%s%s%s", "etcdctl --endpoints ", endpoints, " rm ", key)}
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
