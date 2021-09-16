package utils

import (
	"fmt"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/storageos/kubectl-storageos/pkg/logger"
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
			logger.Printf("Prompt failed %v\n", err)
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
