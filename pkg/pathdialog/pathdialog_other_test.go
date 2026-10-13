//go:build !windows && !darwin

package pathdialog

import (
	"errors"
	"testing"
)

func TestUnsupportedPlatform(t *testing.T) {
	for _, kind := range []Kind{OpenFile, SaveFile, ChooseFolder} {
		result, err := Show(Request{Kind: kind})
		if result != (Result{}) || !errors.Is(err, ErrUnavailable) {
			t.Fatalf("Show(%d) = %+v, %v", kind, result, err)
		}
	}
}
