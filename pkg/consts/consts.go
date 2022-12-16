package consts

const (
	OperatorOldestSupportedVersion = "v2.2.0"
	ClusterOperatorLastVersion     = "v2.4.4"

	PortalManagerFirstSupportedVersion   = "v2.6.0"
	MetricsExporterFirstSupportedVersion = "v2.8.0"

	OldOperatorName = "storageos-cluster-operator"
	NewOperatorName = "storageos-operator"

	NewOperatorNamespace = "storageos"
	OldOperatorNamespace = "storageos-operator"
	OldClusterNamespace  = "kube-system"

	EtcdOperatorName      = "storageos-etcd-controller-manager"
	EtcdOperatorNamespace = "storageos-etcd"

	EtcdSecretName = "storageos-etcd-secret"
)
