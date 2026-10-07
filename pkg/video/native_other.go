//go:build !windows || !386

package video

import (
	"fmt"
	"io"
)

func decodeNative(_, _ string, _ io.Writer, _, _ bool) error {
	return fmt.Errorf("video: decoder helper must be built for windows/386")
}
