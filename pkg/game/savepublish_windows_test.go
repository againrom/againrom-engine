//go:build windows

package game

import (
	"testing"
)

// The two flags have independent safety effects. Write-through requests a
// completed on-disk move before success. Omitting replacement makes final-name
// creation exclusive across processes.
func TestWindowsPublicationIsWriteThroughAndNoReplace(t *testing.T) {
	if windowsPublishFlags&windowsMoveFileWriteThrough == 0 {
		t.Fatal("Windows publication does not request write-through")
	}
	if windowsPublishFlags&windowsMoveFileReplaceExisting != 0 {
		t.Fatal("Windows publication permits replacement")
	}
}
