package utils

import (
	"reflect"
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
			targetKind: "StorageOSCluster",
			targetName: "storageoscluster-sample",
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
			targetKind: "StorageOSCluster",
			targetName: "storageoscluster-sample",
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

func TestGetAllManifestsOfKindFromMultiDoc(t *testing.T) {
	tcases := []struct {
		name          string
		multiManifest string
		kind          string
		expManifests  []string
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
			expManifests: []string{`apiVersion: apps/v1
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
`},
			expError: false,
		},
		{
			name: "find 2 deployments",
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
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-test-2
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
			expManifests: []string{`apiVersion: apps/v1
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
				`apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-test-2
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

`},
			expError: false,
		},
	}
	for _, tc := range tcases {
		mans, err := GetAllManifestsOfKindFromMultiDoc(tc.multiManifest, tc.kind)
		if err != nil {
			if !tc.expError {
				t.Errorf("unexpected error %v", err)
			}
		}
		if len(tc.expManifests) != len(mans) {
			//		if kyaml.MustParse(man).MustString() != kyaml.MustParse(tc.expManifest).MustString() {
			t.Errorf("expected %v manifests, got %v", len(tc.expManifests), len(mans))
		}

	}
}

func TestGenericPatchesFromSupportBundle(t *testing.T) {
	tcases := []struct {
		name         string
		spec         string
		instruction  string
		value        string
		fields       []string
		lookUpValue  string
		skipByFields [][]string
		expPatches   []KustomizePatch
		expError     bool
	}{
		{
			name: "skip all runs",
			spec: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: StorageOS
spec:
  collectors:
    - clusterResources: {}
    - logs:
        name: storageos-operator-logs
        selector:
          - name=storageos-controller-manager
        namespace: storageos
        limits:
          maxLines: 10000
    - logs:
        name: storageos-logs
        selector:
          - app: storageos
        namespace: storageos
        limits:
          maxLines: 1000000
    - run:
        name: "backend-disks"
        collectorName: "lsblk"
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["lsblk"]
        timeout: 90s
    - run:
        name: "free-disk-space"
        collectorName: "df"
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["df -h"]
        timeout: 90s
    - exec:
        name: storageos-cli-info
        collectorName: storageos-cli
        selector:
          - run=cli
        namespace: storageos
        timeout: 90s
        command: ["/bin/sh"]
        args:
        - -c
        - "
          export STORAGEOS_ENDPOINTS='http://storageos.storageos.svc:5705';
          echo STORAGEOS CLUSTER;
          storageos get cluster -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  LICENCE;
          storageos get licence -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  NAMESPACE;
          storageos get namespace -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS VOLUMES;
          storageos get volumes --all-namespaces -ojson --timeout 30s;
          echo '-----------------------------------------';
"`,
			instruction:  "collectors",
			value:        "test",
			fields:       []string{"name"},
			lookUpValue:  "",
			skipByFields: [][]string{{"run"}},
			expPatches: []KustomizePatch{
				{
					Op:    "replace",
					Value: "test",
					Path:  "/spec/collectors/1/logs/name",
				},
				{
					Op:    "replace",
					Value: "test",
					Path:  "/spec/collectors/2/logs/name",
				},
				{
					Op:    "replace",
					Value: "test",
					Path:  "/spec/collectors/5/exec/name",
				},
			},
			expError: false,
		},
		{
			name: "skip operator logs",
			spec: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: StorageOS
spec:
  collectors:
    - clusterResources: {}
    - logs:
        name: storageos-operator-logs
        selector:
          - name=storageos-controller-manager
        namespace: storageos
        limits:
          maxLines: 10000
    - logs:
        selector:
          - app: storageos
        namespace: storageos
        limits:
          maxLines: 1000000
    - run:
        name: "backend-disks"
        collectorName: "lsblk"
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["lsblk"]
        timeout: 90s
    - run:
        name: "free-disk-space"
        collectorName: "df"
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["df -h"]
        timeout: 90s
    - exec:
        name: storageos-cli-info
        collectorName: storageos-cli
        selector:
          - run=cli
        namespace: storageos
        timeout: 90s
        command: ["/bin/sh"]
        args:
        - -c
        - "
          export STORAGEOS_ENDPOINTS='http://storageos.storageos.svc:5705';
          echo STORAGEOS CLUSTER;
          storageos get cluster -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  LICENCE;
          storageos get licence -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  NAMESPACE;
          storageos get namespace -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS VOLUMES;
          storageos get volumes --all-namespaces -ojson --timeout 30s;
          echo '-----------------------------------------';
"`,
			instruction:  "collectors",
			value:        "test-ns",
			fields:       []string{"namespace"},
			lookUpValue:  "storageos-operator-logs",
			skipByFields: [][]string{{"logs", "name"}},
			expPatches: []KustomizePatch{
				{
					Op:    "replace",
					Value: "test-ns",
					Path:  "/spec/collectors/2/logs/namespace",
				},
				{
					Op:    "replace",
					Value: "test-ns",
					Path:  "/spec/collectors/3/run/namespace",
				},
				{
					Op:    "replace",
					Value: "test-ns",
					Path:  "/spec/collectors/4/run/namespace",
				},
				{
					Op:    "replace",
					Value: "test-ns",
					Path:  "/spec/collectors/5/exec/namespace",
				},
			},
			expError: false,
		},
		{
			name: "skip operator logs",
			spec: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: StorageOS
spec:
  collectors:
    - clusterResources: {}
    - logs:
        name: storageos-operator-logs    
        selector:
          - name=storageos-operator-operator-logs
        namespace: storageos
        limits:
          maxLines: 10000
    - logs:
        selector:
          - app=storageos
        namespace: storageos
        limits:
          maxLines: 1000000
    - run:
        name: network-checks
        collectorName: netcat
        image: arau/tools:0.9
        namespace: storageos
        hostNetwork: true
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command:
        - "/bin/sh"
        - "-c"
        - "
          #!/bin/bash
          #
          # IOPort = 5703 # DataPlane
          # SupervisorPort = 5704 # For sync
          # ExternalAPIPort = 5705 # REST API
          # InternalAPIPort = 5710 # Grpc API
          # GossipPort = 5711 # Gossip+Healthcheck

          echo \"Source node for the test:\";
          hostname -f -I; echo;

          parallel -j2 nc -vnz ::: $(echo $NODES_PRIVATE_IPS| sed \"s/,/ /g\" ) \
                              ::: 5703 5704 5705 5710 5711
          "
        timeout: 90s
    - run:
        name: backend-disks
        collectorName: lsblk
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["lsblk"]
        timeout: 90s
    - run:
        name: free-disk-space
        collectorName: df
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["df -h"]
        timeout: 90s
    - run:
        name: ps-all-nodes
        collectorName: processlist"
        image: arau/tools:0.9
        namespace: kube-system
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["ps"]
        args: ["auxwwwf"]
        timeout: 90s
    - exec:
        name: storageos-cli-info
        collectorName: storageos-cli
        selector:
          - run=cli
        namespace: kube-system
        timeout: 90s
        command: ["/bin/sh"]
        args:
        - -c
        - "
          export STORAGEOS_ENDPOINTS='http://storageos.kube-system.svc:5705';
          echo STORAGEOS CLUSTER;
          storageos get cluster -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  LICENCE;
          storageos get licence -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  NAMESPACE;
          storageos get namespace -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS NODES;
          storageos get nodes -o json;
          echo '-----------------------------------------';
          echo STORAGEOS VOLUMES;
          storageos get volumes --all-namespaces -ojson;
          echo '-----------------------------------------';
          "
  analyzers:
    - clusterVersion:
        outcomes:
          - fail:
              when: "< 1.9.0"
              message: StorageOS requires at least Kubernetes 1.9.0 with CSI enabled or later.
              uri: https://kubernetes.io
          - warn:
              when: "< 1.15.0"
              message: Your cluster meets the minimum version of Kubernetes, but we recommend you update to 1.15.0 or later.
              uri: https://kubernetes.io
          - pass:
          - pass:
              message: Your cluster meets the recommended and required versions of Kubernetes.
    - customResourceDefinition:
        customResourceDefinitionName: storageosclusters.storageos.com
        outcomes:
          - fail:
              message: The StorageOSCluster CRD was not found in the cluster.
          - pass:
              message: StorageOS CRD is installed and available.
    - nodeResources:
        checkName: Must have at least 3 nodes in the cluster
        outcomes:
          - warn:
              when: "count() < 3"
              message: This application recommends at last 3 nodes.
          - pass:
              message: This cluster has enough nodes.
    - deploymentStatus:
        name: storageos-operator
        namespace: storageos
        outcomes:
          - fail:
              when: "< 1"
              message: The API Manager deployment does not have any ready replicas.
          - warn:
              when: "= 1"
              message: The API Manager deployment has only a single ready replica.
          - pass:
              message: There are multiple replicas of the API Manager deployment ready.
    - deploymentStatus:
        name: storageos-api-manager
        namespace: storageos
        outcomes:
          - fail:
              when: "< 1"
              message: The API Manager deployment does not have any ready replicas.
          - warn:
              when: "= 1"
              message: The API Manager deployment has only a single ready replica.
          - pass:
              message: There are multiple replicas of the API Manager deployment ready.
    - deploymentStatus:
        name: storageos-api-manager
        namespace: storageos
        outcomes:
          - fail:
              when: "< 1"
              message: The API Manager deployment does not have any ready replicas.
          - warn:
              when: "= 1"
              message: The API Manager deployment has only a single ready replica.
          - pass:
              message: There are multiple replicas of the API Manager deployment ready.
    - deploymentStatus:
        name: storageos-csi-helper
        namespace: storageos
        outcomes:
          - fail:
              when: "< 1"
              message: The CSI helper deployment does not have any ready replicas.
          - pass:
              message: The CSI helper deployment is ready.
    - deploymentStatus:
        name: storageos-scheduler
        namespace: storageos
        outcomes:
          - fail:
              when: "< 1"
              message: The scheduler deployment does not have any ready replicas.
          - pass:		
`,
			instruction:  "analyzers",
			value:        "test-ns",
			fields:       []string{"namespace"},
			lookUpValue:  "storageos-operator",
			skipByFields: [][]string{{"deploymentStatus", "name"}},
			expPatches: []KustomizePatch{
				{
					Op:    "replace",
					Value: "test-ns",
					Path:  "/spec/analyzers/4/deploymentStatus/namespace",
				},
				{
					Op:    "replace",
					Value: "test-ns",
					Path:  "/spec/analyzers/5/deploymentStatus/namespace",
				},
				{
					Op:    "replace",
					Value: "test-ns",
					Path:  "/spec/analyzers/6/deploymentStatus/namespace",
				},
				{
					Op:    "replace",
					Value: "test-ns",
					Path:  "/spec/analyzers/7/deploymentStatus/namespace",
				},
			},
			expError: false,
		},
		{
			name: "skip operator logs",
			spec: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: StorageOS
spec:
  collectors:
    - clusterResources: {}
    - logs:
        name: storageos-operator-logs    
        selector:
          - name=storageos-operator-operator-logs
        namespace: storageos
        limits:
          maxLines: 10000
    - logs:
        selector:
          - app=storageos
        namespace: storageos
        limits:
          maxLines: 1000000
    - run:
        name: network-checks
        collectorName: netcat
        image: arau/tools:0.9
        namespace: storageos
        hostNetwork: true
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command:
        - "/bin/sh"
        - "-c"
        - "
          #!/bin/bash
          #
          # IOPort = 5703 # DataPlane
          # SupervisorPort = 5704 # For sync
          # ExternalAPIPort = 5705 # REST API
          # InternalAPIPort = 5710 # Grpc API
          # GossipPort = 5711 # Gossip+Healthcheck

          echo \"Source node for the test:\";
          hostname -f -I; echo;

          parallel -j2 nc -vnz ::: $(echo $NODES_PRIVATE_IPS| sed \"s/,/ /g\" ) \
                              ::: 5703 5704 5705 5710 5711
          "
        timeout: 90s
    - run:
        name: backend-disks
        collectorName: lsblk
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["lsblk"]
        timeout: 90s
    - run:
        name: free-disk-space
        collectorName: df
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["df -h"]
        timeout: 90s
    - run:
        name: ps-all-nodes
        collectorName: processlist"
        image: arau/tools:0.9
        namespace: kube-system
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["ps"]
        args: ["auxwwwf"]
        timeout: 90s
    - exec:
        name: storageos-cli-info
        collectorName: storageos-cli
        selector:
          - run=cli
        namespace: kube-system
        timeout: 90s
        command: ["/bin/sh"]
        args:
        - -c
        - "
          export STORAGEOS_ENDPOINTS='http://storageos.kube-system.svc:5705';
          echo STORAGEOS CLUSTER;
          storageos get cluster -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  LICENCE;
          storageos get licence -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  NAMESPACE;
          storageos get namespace -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS NODES;
          storageos get nodes -o json;
          echo '-----------------------------------------';
          echo STORAGEOS VOLUMES;
          storageos get volumes --all-namespaces -ojson;
          echo '-----------------------------------------';
          "
  analyzers:
    - clusterVersion:
        outcomes:
          - fail:
              when: "< 1.9.0"
              message: StorageOS requires at least Kubernetes 1.9.0 with CSI enabled or later.
              uri: https://kubernetes.io
          - warn:
              when: "< 1.15.0"
              message: Your cluster meets the minimum version of Kubernetes, but we recommend you update to 1.15.0 or later.
              uri: https://kubernetes.io
          - pass:
          - pass:
              message: Your cluster meets the recommended and required versions of Kubernetes.
    - customResourceDefinition:
        customResourceDefinitionName: storageosclusters.storageos.com
        outcomes:
          - fail:
              message: The StorageOSCluster CRD was not found in the cluster.
          - pass:
              message: StorageOS CRD is installed and available.
    - nodeResources:
        checkName: Must have at least 3 nodes in the cluster
        outcomes:
          - warn:
              when: "count() < 3"
              message: This application recommends at last 3 nodes.
          - pass:
              message: This cluster has enough nodes.
    - deploymentStatus:
        name: storageos-operator
        namespace: storageos
        outcomes:
          - fail:
              when: "< 1"
              message: The API Manager deployment does not have any ready replicas.
          - warn:
              when: "= 1"
              message: The API Manager deployment has only a single ready replica.
          - pass:
              message: There are multiple replicas of the API Manager deployment ready.
    - deploymentStatus:
        name: storageos-api-manager
        namespace: storageos
        outcomes:
          - fail:
              when: "< 1"
              message: The API Manager deployment does not have any ready replicas.
          - warn:
              when: "= 1"
              message: The API Manager deployment has only a single ready replica.
          - pass:
              message: There are multiple replicas of the API Manager deployment ready.
    - deploymentStatus:
        name: storageos-api-manager
        namespace: storageos
        outcomes:
          - fail:
              when: "< 1"
              message: The API Manager deployment does not have any ready replicas.
          - warn:
              when: "= 1"
              message: The API Manager deployment has only a single ready replica.
          - pass:
              message: There are multiple replicas of the API Manager deployment ready.
    - deploymentStatus:
        name: storageos-csi-helper
        namespace: storageos
        outcomes:
          - fail:
              when: "< 1"
              message: The CSI helper deployment does not have any ready replicas.
          - pass:
              message: The CSI helper deployment is ready.
    - deploymentStatus:
        name: storageos-scheduler
        namespace: storageos
        outcomes:
          - fail:
              when: "< 1"
              message: The scheduler deployment does not have any ready replicas.
          - pass:		
`,
			instruction:  "analyzers",
			value:        "test",
			fields:       []string{"checkName"},
			lookUpValue:  "",
			skipByFields: [][]string{{"deploymentStatus"}, {"clusterVersion"}, {"customResourceDefinition"}},

			expPatches: []KustomizePatch{
				{
					Op:    "replace",
					Value: "test",
					Path:  "/spec/analyzers/2/nodeResources/checkName",
				},
			},
			expError: false,
		},
	}
	for _, tc := range tcases {
		patches, err := GenericPatchesForSupportBundle(tc.spec, tc.instruction, tc.value, tc.fields, tc.lookUpValue, tc.skipByFields)
		if err != nil {
			if !tc.expError {
				t.Errorf("unexpected error %v", err)
			}
		}
		if !reflect.DeepEqual(tc.expPatches, patches) {
			t.Errorf("expected %v, got %v", tc.expPatches, patches)
		}

	}
}

func TestSpecificPatchForSupportBundle(t *testing.T) {
	tcases := []struct {
		name         string
		spec         string
		instruction  string
		value        string
		fields       []string
		lookUpValue  string
		findByFields []string
		expPatch     KustomizePatch
		expError     bool
	}{
		{
			name: "find deployment",
			spec: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: StorageOS
spec:
  collectors:
    - clusterResources: {}
    - logs:
        name: storageos-operator-logs
        selector:
          - name=storageos-controller-manager
        namespace: storageos
        limits:
          maxLines: 10000
    - logs:
        selector:
          - app: storageos
        namespace: storageos
        limits:
          maxLines: 1000000
    - run:
        name: "backend-disks"
        collectorName: "lsblk"
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["lsblk"]
        timeout: 90s
    - run:
        name: "free-disk-space"
        collectorName: "df"
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["df -h"]
        timeout: 90s
    - exec:
        name: storageos-cli-info
        collectorName: storageos-cli
        selector:
          - run=cli
        namespace: storageos
        timeout: 90s
        command: ["/bin/sh"]
        args:
        - -c
        - "
          export STORAGEOS_ENDPOINTS='http://storageos.storageos.svc:5705';
          echo STORAGEOS CLUSTER;
          storageos get cluster -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  LICENCE;
          storageos get licence -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  NAMESPACE;
          storageos get namespace -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS VOLUMES;
          storageos get volumes --all-namespaces -ojson --timeout 30s;
          echo '-----------------------------------------';
"`,
			instruction:  "collectors",
			value:        "test-ns",
			fields:       []string{"logs", "namespace"},
			lookUpValue:  "storageos-operator-logs",
			findByFields: []string{"logs", "name"},
			expPatch: KustomizePatch{
				Op:    "replace",
				Value: "test-ns",
				Path:  "/spec/collectors/1/logs/namespace",
			},
			expError: false,
		},
		{
			name: "find deployment",
			spec: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: StorageOS
spec:
  collectors:
    - clusterResources: {}
    - logs:
        name: storageos-operator-logs
        selector:
          - name=storageos-controller-manager
        namespace: storageos
        limits:
          maxLines: 10000
    - logs:
        selector:
          - app: storageos
        namespace: storageos
        limits:
          maxLines: 1000000
    - run:
        name: "backend-disks"
        collectorName: "lsblk"
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["lsblk"]
        timeout: 90s
    - run:
        name: "free-disk-space"
        collectorName: "df"
        image: arau/tools:0.9
        namespace: storageos
        hostPID: true
        nodeSelector:
          node-role.kubernetes.io/worker: "true"
        command: ["df -h"]
        timeout: 90s
    - exec:
        name: storageos-cli-info
        collectorName: storageos-cli
        selector:
          - run=cli
        namespace: storageos
        timeout: 90s
        command: ["/bin/sh"]
        args:
        - -c
        - "
          export STORAGEOS_ENDPOINTS='http://storageos.storageos.svc:5705';
          echo STORAGEOS CLUSTER;
          storageos get cluster -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  LICENCE;
          storageos get licence -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS  NAMESPACE;
          storageos get namespace -ojson;
          echo '-----------------------------------------';
          echo STORAGEOS VOLUMES;
          storageos get volumes --all-namespaces -ojson --timeout 30s;
          echo '-----------------------------------------';
"`,
			instruction:  "collectors",
			value:        "test-ns",
			fields:       []string{"exec", "namespace"},
			lookUpValue:  "storageos-cli-info",
			findByFields: []string{"exec", "name"},
			expPatch: KustomizePatch{
				Op:    "replace",
				Value: "test-ns",
				Path:  "/spec/collectors/5/exec/namespace",
			},
			expError: false,
		},
	}
	for _, tc := range tcases {
		patch, err := SpecificPatchForSupportBundle(tc.spec, tc.instruction, tc.value, tc.fields, tc.lookUpValue, tc.findByFields)
		if err != nil {
			if !tc.expError {
				t.Errorf("unexpected error %v", err)
			}
		}
		if !reflect.DeepEqual(tc.expPatch, patch) {
			t.Errorf("expected %v, got %v", tc.expPatch, patch)
		}

	}
}

func TestAllInstructionTypesExcept(t *testing.T) {
	tcases := []struct {
		name        string
		instruction string
		exceptions  []string
		expList     [][]string
		expError    bool
	}{
		{
			name:        "collectors",
			instruction: "collectors",
			exceptions:  []string{"clusterInfo", "run", "logs", "copy"},
			expList: [][]string{
				{"clusterResources"},
				{"data"},
				{"secret"},
				{"http"},
				{"exec"},
				{"postgresql"},
				{"mysql"},
				{"redis"},
				{"ceph"},
				{"longhorn"},
				{"registryImages"},
			},
			expError: false,
		},
	}
	for _, tc := range tcases {
		list, err := AllInstructionTypesExcept(tc.instruction, tc.exceptions...)
		if err != nil {
			if !tc.expError {
				t.Errorf("unexpected error %v", err)
			}
		}
		if !reflect.DeepEqual(list, tc.expList) {
			t.Errorf("expected %v, got %v", tc.expList, list)
		}
	}
}
