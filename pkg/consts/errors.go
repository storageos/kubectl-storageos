package consts

const (
	ErrNotFoundTemplate = `Please ensure you have specified the correct namespace.
	Please check CLI flags of %s command.
	# kubectl storageos %s -h
	`

	ErrUnableToConstructClientConfig            = "unable to connect to Kubernetes with given config"
	ErrUnableToConstructClientConfigTemplate    = `Please ensure your Kubernetes client config has been configured correctly`
	ErrUnableToContructClientFromConfig         = "unable to construct client for Kubernetes cluster"
	ErrUnableToContructClientFromConfigTemplate = `Please ensure you have the correct Kubernetes config.`
)
