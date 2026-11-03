//go:build !windows

package hostfile

import (
	"errors"
	"os"
)

// publishFile uses an exclusive hard-link creation as the portable
// no-replace publication primitive. The temporary file is a sibling, so both
// names are on one filesystem. Link creation is the commit point. Removing
// the private name is post-commit cleanup: a refusal leaves a hidden alias for
// next-save recovery but cannot turn the already-visible valid save into an
// API failure.
func publishFile(oldPath, newPath string) (PublishResult, error) {
	return PublishLinked(oldPath, newPath, os.Link, os.Remove)
}

// IsCollision reports whether a publication failed because the final name
// already exists.
func IsCollision(err error) bool { return errors.Is(err, os.ErrExist) }
