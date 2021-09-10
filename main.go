package main

import (
	"fmt"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/storageos/kubectl-storageos/cmd/plugin/cli"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			println(fmt.Sprintf("%v", r))
			os.Exit(1)
		}
	}()

	cli.InitAndExecute()
}
