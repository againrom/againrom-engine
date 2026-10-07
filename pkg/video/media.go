package video

import "errors"

// MaxMediaBytes bounds one compressed movie before it is handed to either
// decode backend: the optional installed-decoder subprocess this project no
// longer starts at runtime (see smackerdecode.go), and this project's own
// pkg/video/smacker port, which reads the same bound (InspectNativeInput's
// own comment on this constant's prior home, process.go, before the swap).
// This is an authored safety limit, not a property of any Smacker file
// format.
const MaxMediaBytes = 128 << 20

// ErrAbsent is a normal numbered-scan miss, not a decoder failure.
var ErrAbsent = errors.New("video: media absent")
