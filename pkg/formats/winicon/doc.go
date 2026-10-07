// Package winicon reads the icon of a Windows executable: the first icon group
// of its resource section and every image that group lists.
//
// An executable keeps an icon as two kinds of resource. An icon group (type 14)
// lists the images of one icon, each entry naming the icon resource (type 3)
// that holds the image. An image is a bitmap, stored as a BITMAPINFOHEADER whose
// height counts the colour bitmap and the 1-bit AND mask together, or a whole
// PNG file.
//
// Transparency follows Windows. A bitmap image of 24 bits or fewer is opaque
// except where its AND mask bit is set. A 32-bit image takes its alpha from the
// fourth byte of each pixel and ignores the mask, unless every one of those
// bytes is zero, in which case the mask decides as it does for the shallower
// depths. A masked pixel decodes to zero in every channel: what a masked pixel's
// colour would do to the desktop cannot be a window icon's pixel. The padding
// bits that end each 4-byte mask row are never read; the shipped icon sets some.
//
// Every size and count read from the executable is bounded before it allocates
// anything, and a damaged executable is an error and never a panic.
//
// The package opens no file: a caller hands over an io.ReaderAt. It is held to
// the standard library, since an icon carries no text.
package winicon
