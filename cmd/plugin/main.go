package main

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/storageos/kubectl-storageos/cmd/plugin/cli"
)

func main() {
	cli.InitAndExecute()
}
