package utils

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/manifoldco/promptui"
	"github.com/storageos/kubectl-storageos/pkg/consts"
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
		if _, ok := r.(stackTracer); ok {
			setError(r)
		} else {
			setError(errors.WithStack(r))
		}
	default:
		setError(errors.WithStack(fmt.Errorf("%v", r)))
	}
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// HandleError tries to convert program error to something useful for user.
func HandleError(command string, err error, printStackTrace bool) error {
	if stacked, ok := err.(stackTracer); ok && printStackTrace {
		println("Stack trace:")
		for _, f := range stacked.StackTrace() {
			println(fmt.Sprintf("%+s:%d", f, f))
		}
	}

	errToTest := err
	for {
		if errToTest == nil {
			return err
		}
		switch {
		// Some resource has not found.
		case kerrors.IsNotFound(errToTest):
			return errors.Wrap(err, fmt.Sprintf(consts.ErrNotFoundTemplate, command, command))
		// Something is wrong with Kube config.
		case errToTest.Error() == consts.ErrUnableToConstructClientConfig:
			return errors.Wrap(err, consts.ErrUnableToConstructClientConfigTemplate)
		// Clientset construction has failed.
		case errToTest.Error() == consts.ErrUnableToContructClientFromConfig:
			return errors.Wrap(err, consts.ErrUnableToContructClientFromConfigTemplate)
		default:
			errToTest = errors.Unwrap(errToTest)
		}
	}
}

// FetchHttpContent downloads something from given URL
func FetchHttpContent(url string, headers map[string]string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusMultipleChoices {
		return nil, errors.WithStack(fmt.Errorf("error fetching content of %s, status code: %d", url, resp.StatusCode))
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
