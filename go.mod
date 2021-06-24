module github.com/storageos/kubectl-storageos

go 1.15

require (
	github.com/ahmetalpbalkan/go-cursor v0.0.0-20131010032410-8136607ea412
	github.com/darkowlzz/operator-toolkit v0.0.0-20210603234749-4f4acec01835
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/color v1.7.0
	github.com/frankban/quicktest v1.13.0 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/manifoldco/promptui v0.8.0
	github.com/mattn/go-isatty v0.0.9
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/termui/v3 v3.1.1-0.20200811145416-f40076d26851
	github.com/replicatedhq/troubleshoot v0.10.24
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/tj/go-spin v1.1.0
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200910180754-dd1b699fc489 // indirect
	go.etcd.io/etcd/client/v3 v3.5.0-rc.1 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/cli-runtime v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/kubebuilder-declarative-pattern v0.0.0-20210322221347-4ba4cadcd4ca
	sigs.k8s.io/kustomize/api v0.7.0
	sigs.k8s.io/kustomize/kyaml v0.10.3
)

replace (
	//	github.com/coreos/etcd => go.etcd.io/etcd v3.3.13+incompatible
	//	go.etcd.io/bbolt => go.etcd.io/bbolt v1.3.5
	//	go.etcd.io/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20200910180754-dd1b699fc489 // ae9734ed278b is the SHA for git tag v3.4.13
	google.golang.org/grpc => google.golang.org/grpc v1.27.1
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
)
