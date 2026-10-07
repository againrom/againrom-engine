package vfs

import "againrom/pkg/formats/res"

// DecodeText is an install byte string turned into a UTF-8 Go string, for a
// consumer that has to REPORT the install's words rather than draw them.
//
// The install's byte strings — entry names, registry values, the text files
// under `main.res` — are one single-byte code page, and it is the archive tier's
// own. This tier is where the tiers above it reach that code page, because
// pkg/game reads every container entry through an address here and holds no
// archive reader of its own (0027 SC-8): a caller that imported the reader for
// this one function would be re-opening exactly the door that migration shut.
//
// THE DRAW PATH MUST NOT CALL THIS. A shipped byte string reaches the renderer
// unconverted, and the font's own byte-to-glyph selector is where the code page
// is applied there. Converting first would corrupt every non-ASCII byte on the
// way. The two consumers want different things from the same bytes and neither
// is converted for the other.
func DecodeText(b []byte) string { return res.DecodeCP866(b) }

func WindowsCyrillicToDOS(b []byte) []byte { return res.WindowsCyrillicToDOS(b) }
