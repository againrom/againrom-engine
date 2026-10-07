package game

import (
	"errors"
	"fmt"
	"os"
)

// publishLinkedSave is the non-Windows publication protocol with its two
// operating-system operations injected so Windows CI can exercise every
// result. The exclusive hard link is the commit point. Once it succeeds the
// final path is valid and must never be rolled back merely because removal of
// the private source name was refused: doing so creates an outcome where both
// rollback and cleanup can fail while the caller is falsely told that no save
// exists.
func publishLinkedSave(oldPath, newPath string,
	link func(string, string) error, remove func(string) error) (savePublishResult, error) {

	if err := link(oldPath, newPath); err != nil {
		return savePublishResult{}, err
	}
	result := savePublishResult{committed: true}
	if err := remove(oldPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		result.cleanupErr = fmt.Errorf("remove private name after committed save: %w", err)
	}
	return result, nil
}
