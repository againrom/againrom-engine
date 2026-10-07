//go:build !windows

package game

import (
	"errors"
	"os"
)

// publishSaveFile uses an exclusive hard-link creation as the portable
// no-replace publication primitive. The temporary file is a sibling, so both
// names are on one filesystem. Link creation is the commit point. Removing
// the private name is post-commit cleanup: a refusal leaves a hidden alias for
// next-save recovery but cannot turn the already-visible valid save into an
// API failure.
func publishSaveFile(oldPath, newPath string) (savePublishResult, error) {
	return publishLinkedSave(oldPath, newPath, os.Link, os.Remove)
}

func isPublishCollision(err error) bool { return errors.Is(err, os.ErrExist) }
