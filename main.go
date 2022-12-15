package main

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/storageos/kubectl-storageos/cmd"
)

func main() {
	cmd.InitAndExecute()
}
