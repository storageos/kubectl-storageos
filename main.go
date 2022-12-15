package main

import (
	"errors"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/storageos/kubectl-storageos/cmd"
)

func main() {
	IReturnAnError()

	cmd.InitAndExecute()
}

func IReturnAnError() error {
	return errors.New("fail")
}
