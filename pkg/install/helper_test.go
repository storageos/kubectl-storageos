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
    nodeContainer: storageos/node:latest
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
    nodeContainer: storageos/node:latest
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
			t.Errorf("expected %v, got %v", kyaml.MustParse(tc.expManifest).MustString(), kyaml.MustParse(manifest).MustString())
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

func TestAddPatchesToKustomize(t *testing.T) {
	tcases := []struct {
		name       string
		kust       string
		targetKind string
		targetName string
		patches    []KustomizePatch
		expKust    string
		expErr     bool
	}{
		{
			name: "add single patch for kvbackend",
			kust: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- storageos-cluster.yaml`,
			targetKind: stosClusterKind,
			targetName: defaultStosClusterName,
			patches: []KustomizePatch{
				{
					Op:    "replace",
					Path:  "/spec/kvBackend/address",
					Value: "storageos.storageos-etcd:2379",
				},
			},
			expErr: false,
			expKust: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- storageos-cluster.yaml

patches:
- target:
    kind: StorageOSCluster
    name: storageoscluster-sample
  patch: |
    - op: replace
      path: /spec/kvBackend/address
      value: storageos.storageos-etcd:2379`,
		},
		{
			name: "add multiple patches",
			kust: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- storageos-cluster.yaml`,
			targetKind: stosClusterKind,
			targetName: defaultStosClusterName,
			patches: []KustomizePatch{
				{
					Op:    "replace",
					Path:  "/spec/kvBackend/address",
					Value: "storageos.storageos-etcd:2379",
				},
				{
					Op:    "add",
					Path:  "/spec/abc/def",
					Value: "newvalue",
				},
				{
					Op:    "add",
					Path:  "/spec/ghi/jkl",
					Value: "newvalue",
				},
			},
			expErr: false,
			expKust: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- storageos-cluster.yaml

patches:
- target:
    kind: StorageOSCluster
    name: storageoscluster-sample
  patch: |
    - op: replace
      path: /spec/kvBackend/address
      value: storageos.storageos-etcd:2379
    - op: add
      path: /spec/abc/def
      value: newvalue
    - op: add
      path: /spec/ghi/jkl
      value: newvalue`,
		},
	}
	for _, tc := range tcases {
		kust, err := AddPatchesToKustomize(tc.kust, tc.targetKind, tc.targetName, tc.patches)
		if err != nil {
			if !tc.expErr {
				t.Errorf("got err %v unexpectedly", err)
			}
		}
		if kyaml.MustParse(kust).MustString() != kyaml.MustParse(tc.expKust).MustString() {
			t.Errorf("expected %v, got %v", kyaml.MustParse(tc.expKust).MustString(), kyaml.MustParse(kust).MustString())
		}

	}
}

func TestGetManifestFromMultiDoc(t *testing.T) {
	tcases := []struct {
		name          string
		multiManifest string
		kind          string
		expManifest   string
		expError      bool
	}{
		{
			name: "find deployment",
			multiManifest: `
apiVersion: v1
kind: Service
metadata:
  name: my-test-svc
  labels:
    app: test
spec:
  type: LoadBalancer
  ports:
  - port: 80
  selector:
    app: test
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-test
  labels:
    app: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: test
        image: test
`,
			kind: "Deployment",
			expManifest: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: my-test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - image: test
        name: test
`,
			expError: false,
		},
	}
	for _, tc := range tcases {
		man, err := GetManifestFromMultiDoc(tc.multiManifest, tc.kind)
		if err != nil {
			if !tc.expError {
				t.Errorf("unexpected error %v", err)
			}
		}
		if kyaml.MustParse(man).MustString() != kyaml.MustParse(tc.expManifest).MustString() {
			t.Errorf("expected %v, got %v", kyaml.MustParse(tc.expManifest).MustString(), kyaml.MustParse(man).MustString())
		}

	}
}
