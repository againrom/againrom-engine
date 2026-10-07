// Package bmp decodes the one Windows BMP shape this game ships.
//
// It is the seventh format leaf and the only one that is not this engine's own
// invention: `graphics\infowindow` — where a non-composed actor's picture lives
// — is not `.256` at all, and neither is the spell book's icon atlas under
// `graphics\interface`. Both are plain 24-bit uncompressed Windows bitmaps
// (`SPR256-PICT-043`, `MAGIC-ICON-024`).
//
// THE SHAPE IS NARROW AND THIS PACKAGE REFUSES EVERYTHING ELSE. Every shipped
// node measured is `planes=1`, `bpp=24`, `compression=0`, `dataOff=54`,
// `nClrs=0` — 86 of 86 on the EN root and 81 of 81 on the RU, walked rather
// than sampled. A general BMP reader would carry palettes, run-length
// compression, bit fields and five header versions this corpus does not
// contain, and every one of those is a branch nothing here could exercise.
// Refusing them is what keeps the decoder's own contract testable.
//
// IT DOES NOT TRUST THE HEADER'S OWN LENGTH. `imgSize` is **0** on 85 of the 86
// shipped portrait nodes and correct on exactly one, so the pixel run's length
// is computed from the width, the height and the row stride and never read out
// of the file. For the same reason a file LONGER than the computed run is
// accepted and its tail ignored: 85 of 86 carry two bytes past the end of their
// own pixels, and what those two bytes are is not established.
package bmp
