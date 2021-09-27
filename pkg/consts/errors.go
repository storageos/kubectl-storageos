package consts

const (
	ErrNotFoundTemplate = `Please ensure you have specified the correct namespace.
	Please check CLI flags of %s command.
	# kubectl storageos %s -h
	`

	ErrUnableToConstructClientConfig         = "unable to connect to Kubernetes with given config"
	ErrUnableToConstructClientConfigTemplate = `Please check your Kubernetes config.
	There must be some misconfiguration in it or the cluster is not responds on network.`

	ErrUnableToContructClientFromConfig         = "unable to contruct client of Kubernetes cluster"
	ErrUnableToContructClientFromConfigTemplate = `Please ensure you have the correct Kubernetes config.`
)
