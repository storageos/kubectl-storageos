module github.com/storageos/kubectl-storageos

go 1.16

require (
	github.com/ahmetalpbalkan/go-cursor v0.0.0-20131010032410-8136607ea412
	github.com/darkowlzz/operator-toolkit v0.0.0-20210603234749-4f4acec01835
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/color v1.12.0
	github.com/frankban/quicktest v1.13.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/golang/snappy v0.0.3 // indirect
	github.com/hashicorp/go-version v1.1.0
	github.com/improbable-eng/etcd-cluster-operator v0.2.0
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/manifoldco/promptui v0.8.0
	github.com/mattn/go-isatty v0.0.12
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/termui/v3 v3.1.1-0.20200811145416-f40076d26851
	github.com/replicatedhq/troubleshoot v0.13.5
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/storageos/cluster-operator v2.1.0+incompatible
	github.com/tj/go-spin v1.1.0
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/cli-runtime v0.21.1
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.9.0
	sigs.k8s.io/kubebuilder-declarative-pattern v0.0.0-20210322221347-4ba4cadcd4ca
	sigs.k8s.io/kustomize/api v0.8.8
	sigs.k8s.io/kustomize/kyaml v0.10.17
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.8
	github.com/hashicorp/go-version => github.com/hashicorp/go-version v1.3.0
	github.com/improbable-eng/etcd-cluster-operator => github.com/storageos/etcd-cluster-operator v0.3.0
	github.com/longhorn/longhorn-manager => github.com/replicatedhq/longhorn-manager v1.1.2-0.20210622201804-05b01947b99d
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc95
	github.com/replicatedhq/troubleshoot => github.com/storageos/troubleshoot v0.9.48-0.20210927120021-b81ef00486cf
	k8s.io/api => k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.1
	k8s.io/apiserver => k8s.io/apiserver v0.21.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.1
	k8s.io/client-go => k8s.io/client-go v0.21.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.1
	k8s.io/code-generator => k8s.io/code-generator v0.21.1
	k8s.io/component-base => k8s.io/component-base v0.21.1
	k8s.io/cri-api => k8s.io/cri-api v0.21.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.1
	k8s.io/kubectl => k8s.io/kubectl v0.21.1
	k8s.io/kubelet => k8s.io/kubelet v0.21.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.1
	k8s.io/metrics => k8s.io/metrics v0.21.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.1
	sigs.k8s.io/controller-runtime => github.com/kubernetes-sigs/controller-runtime v0.9.0
)
