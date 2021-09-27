package utils

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/manifoldco/promptui"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

const promptTimeout = time.Minute

// AskUser creates an interactive prompt and waits for user input with timeout
func AskUser(prompt promptui.Prompt) (string, error) {
	ticker := time.NewTicker(promptTimeout)
	defer ticker.Stop()

	resultChan := make(chan string)
	errorChan := make(chan error)

	go func() {
		result, err := prompt.Run()
		if err != nil {
			logger.Printf("Prompt failed %s\n", err.Error())
			errorChan <- err
		}

		resultChan <- result
	}()

	select {
	case <-ticker.C:
		return "", fmt.Errorf("timeout exceded, missing config flag: %s", prompt.Label)
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return "", err
	}
}

// ConvertPanicToError tries to catch panic and convert it to normal error.
func ConvertPanicToError(setError func(err error)) {
	r := recover()
	if r == nil {
		return
	}

	switch r := r.(type) {
	case string:
		setError(errors.New(r))
	case error:
		setError(r)
	default:
		setError(fmt.Errorf("%v", r))
	}
}

// HandleError tries to convert program error to something useful for user.
func HandleError(command string, err error) error {
	errToTest := err

	for {
		if errToTest == nil {
			return err
		}

		if kerrors.IsNotFound(errToTest) {
			return errors.Wrap(err, fmt.Sprintf(`Maybe you have specified a wrong namespace.
	Please check CLI flags of %s command.
	# kubectl storageos %s -h
	`, command, command))
		}

		errToTest = errors.Unwrap(errToTest)
	}
}
