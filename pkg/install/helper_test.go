package install

import (
	"testing"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetFieldInManifest(t *testing.T) {
	tcases := []struct {
		name        string
		manifest    string
		value       string
		valueName   string
		fields      []string
		expManifest string
		expError    bool
	}{
		{
			name: "set pod name",
			manifest: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-pod
  namespace: default
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
spec:
  templates:
    spec:
      containers:
      - name: nginx
        image: nginx:latest
`,
			value:     "my-pod",
			valueName: "name",
			fields:    []string{"metadata"},
			expManifest: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-pod
  namespace: default
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
spec:
  templates:
    spec:
      containers:
      - name: nginx
        image: nginx:latest	
`,
		},
		{
			name: "set stos cluster kvbackend",
			manifest: `apiVersion: storageos.com/v1
kind: StorageOSCluster
metadata:
  name: example-storageoscluster
  namespace: "default"
spec:
  secretRefName: "storageos-api"
  secretRefNamespace: "default"
  namespace: "kube-system"
  k8sDistro: "kubecover"
  # storageClassName: fast
  # tlsEtcdSecretRefName:
  # tlsEtcdSecretRefNamespace:
  disableTelemetry: true
  images:
    nodeContainer: soegarots/node:latest
    apiManagerContainer: storageos/api-manager:develop
  #   initContainer:
  #   csiNodeDriverRegistrarContainer:
  #   csiClusterDriverRegistrarContainer:
  #   csiExternalProvisionerContainer:
  #   csiExternalAttacherContainer:
  #   csiExternalResizerContainer:
  #   csiLivenessProbeContainer:
  #   kubeSchedulerContainer:
  kvBackend:
    address: "1.2.3.4:2379"
  debug: true,
`,
			value:     "storageos-etcd.eco-system:2379",
			valueName: "address",
			fields:    []string{"spec", "kvBackend"},
			expManifest: `apiVersion: storageos.com/v1
kind: StorageOSCluster
metadata:
  name: example-storageoscluster
  namespace: "default"
spec:
  secretRefName: "storageos-api"
  secretRefNamespace: "default"
  namespace: "kube-system"
  k8sDistro: "kubecover"
  # storageClassName: fast
  # tlsEtcdSecretRefName:
  # tlsEtcdSecretRefNamespace:
  disableTelemetry: true
  images:
    nodeContainer: soegarots/node:latest
    apiManagerContainer: storageos/api-manager:develop
  #   initContainer:
  #   csiNodeDriverRegistrarContainer:
  #   csiClusterDriverRegistrarContainer:
  #   csiExternalProvisionerContainer:
  #   csiExternalAttacherContainer:
  #   csiExternalResizerContainer:
  #   csiLivenessProbeContainer:
  #   kubeSchedulerContainer:
  kvBackend:
    address: "storageos-etcd.eco-system:2379"
  debug: true,
`,
		},
		{
			name:      "set resource for kustomize",
			manifest:  `resources:`,
			value:     "[etcd-cluster.yaml]",
			valueName: "resources",
			fields:    nil,
			expManifest: `resources:
- etcd-cluster.yaml
`,
			expError: false,
		},
	}
	for _, tc := range tcases {
		manifest, err := SetFieldInManifest(tc.manifest, tc.value, tc.valueName, tc.fields...)
		if err != nil {
			if !tc.expError {
				t.Errorf("unexpected error %v", err)
			}
		}

		if kyaml.MustParse(manifest).MustString() != kyaml.MustParse(tc.expManifest).MustString() {
			t.Errorf("expected %v, got %v", kyaml.MustParse(manifest).MustString(), kyaml.MustParse(tc.expManifest).MustString())
		}
	}
}

func TestGetFieldInManifest(t *testing.T) {
	tcases := []struct {
		name      string
		manifest  string
		fields    []string
		expString string
		expError  bool
	}{
		{
			name: "find pod name",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-pod
  namespace: default
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
spec:
  templates:
    spec:
      containers:
      - name: nginx
        image: nginx:latest
`,
			fields:    []string{"metadata", "name"},
			expString: "test-pod",
		},
		{
			name: "find pod namespace",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-pod
  namespace: default
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
spec:
  templates:
    spec:
      containers:
      - name: nginx
        image: nginx:latest
`,
			fields:    []string{"metadata", "namespace"},
			expString: "default",
		},
	}
	for _, tc := range tcases {
		field, err := GetFieldInManifest(tc.manifest, tc.fields...)
		if err != nil {
			if !tc.expError {
				t.Errorf("unexpected error %v", err)
			}
		}
		if field != tc.expString {
			t.Errorf("expected %v, got %v", tc.expString, field)
		}
	}
}
