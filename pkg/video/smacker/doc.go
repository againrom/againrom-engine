// Package smacker is a portable, pure-Go decoder for the Smacker (.smk) video
// container used by ROM1's cutscenes. It is a source-derived port of
// libsmacker 1.2.0 (http://libsmacker.sourceforge.net), Copyright (C)
// 2012-2021 Greg Kennedy, licensed under the GNU Lesser General Public
// License version 2.1 or later (LGPL-2.1-or-later). The original C source is
// reproduced under review/libsmacker-reference/ outside this repository; see
// THIRD_PARTY_NOTICES.md and LICENSES/LGPL-2.1.txt for the full attribution
// and license text this port carries forward.
//
// This package is deliberately structured to mirror libsmacker's own source
// layout (bitstream, two Huffman-tree shapes, container parsing, video block
// rendering, audio DPCM decode) so a reader with smacker.c open can follow
// each function across the port.
//
// # Not ROM1 evidence
//
// This is third-party middleware under attribution, not reverse-engineered
// from ROM1 or the installed smackw32.dll. It produces no research claim and
// must never be cited as proof of original-engine behaviour. See
// docs/DIVERGENCES.md DIV-1257..DIV-1262 for the policy choices this port
// makes where the reference library itself leaves room (audio track
// selection, timing bounds, allocation hardening).
package smacker
