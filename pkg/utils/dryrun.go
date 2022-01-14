package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const stosDryRunDir = "./storageos-dry-run"

var dryRunWriterService = dryRunWriter{
	lock: make(chan bool, 1),
}

type dryRunWriter struct {
	path string
	lock chan bool
}

func (w *dryRunWriter) writeDryRunManifests(filename string, fileData []byte) error {
	w.lock <- true
	defer func() {
		<-w.lock
	}()

	cwd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}

	if w.path == "" {
		w.path = stosDryRunDir
		if _, err = os.Stat(filepath.Join(cwd, w.path)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.WithStack(err)
		} else if err == nil {
			w.path = fmt.Sprintf("%s-%d", stosDryRunDir, time.Now().UnixNano())
		}

		if err = os.Mkdir(filepath.Join(cwd, w.path), 0770); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := os.WriteFile(filepath.Join(cwd, w.path, filename), fileData, 0640); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// WriteDryRunManifests protects existing dry run output by postfixing directory.
// This function is used more than 1 time during the manifests generation,
// So it reuses the same directory for eaach run in a thread safe manner.
func WriteDryRunManifests(filename string, fileData []byte) error {
	return dryRunWriterService.writeDryRunManifests(filename, fileData)
}
