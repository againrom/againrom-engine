package hostfile

import (
	"errors"
	"fmt"
	"os"
)

// PublishLinked is the non-Windows publication protocol with its two
// operating-system operations injected so Windows CI can exercise every
// result. The exclusive hard link is the commit point. Once it succeeds the
// final path is valid and must never be rolled back merely because removal of
// the private source name was refused: doing so creates an outcome where both
// rollback and cleanup can fail while the caller is falsely told that no save
// exists.
func PublishLinked(oldPath, newPath string,
	link func(string, string) error, remove func(string) error) (PublishResult, error) {

	if err := link(oldPath, newPath); err != nil {
		return PublishResult{}, err
	}
	result := PublishResult{Committed: true}
	if err := remove(oldPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		result.CleanupErr = fmt.Errorf("remove private name after committed save: %w", err)
	}
	return result, nil
}
