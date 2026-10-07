# 1289 - hover help remainder

## Result

The map-selection list's description column now reads what the claim names:
the leading string of the 512-byte block at payload `+0x78`, with each line
feed shown as a `#` break. The other open hover rows were compared with their
claims and with the claims promoted since pin k118. None gained a claim that
settles one of their stated unknowns, so their behaviour is unchanged. No
simulation, save or hashed state changed.

## Authority

Knowledge pin k122. TEXT-083 (map list), ALM-META-028 (the 512-byte block),
TEXT-080 to TEXT-082, TEXT-084, TEXT-085 (compared, unchanged),
TEXT-HOVERPAINT-053 (popup layout, unchanged).

## As-built behaviour

`alm.Info.ListDescription` is the block's leading C string after every line
feed in the 512 bytes became `#`, converted from CP1251. The map list rows
carry it as their description; `alm.Info.Description`, the 64-byte field the
map document round-trips, is unchanged. A block without a terminator stops at
its 512 bytes, and the popup's existing width clamp bounds it.

Census over the EN and RU installs (38 and 34 maps, each `.alm` read from the
archive or the loose files): no description holds a line feed and none reaches
64 bytes, so the shipped lists look the same as before. The difference
appears for a map, such as a modded one, whose description is longer than 64
bytes or holds line feeds.

## Divergences

DIV-1884 narrowed: the description block length is no longer a stated
difference; the size relation stays open. DIV-1270 states the 512-byte bound.
DIV-1882, DIV-1883, DIV-1885 unchanged (see open debt). DIV-2010 to DIV-2015
unused.

## Proof

- `pkg/formats/alm/info_test.go` `TestOpenInfoListDescriptionReadsTheWholeBlock`:
  line feed, length beyond 64 bytes, unterminated block.
- `TestReleaseMapListHoverReadsTheWholeDescriptionBlock` (EN and RU, in
  `internal/gatedtests/testdata/population.txt`): an installed archive map's
  own description is repeated six times with line feeds in its block; the
  popup over the row, reached by pointer hover on the map list, equals the
  six lines composed by the shared popup composer. Loss control: the old
  64-byte field read with raw line feeds paints a different popup, and the
  test fails with the old read.

## Open debt

Each is a falsifiable question for research; none states an expected answer.

- DIV-1882: which spell record fields do `+0x9`, `+0xe`, `+0xf` and word
  `+0x10` hold, and which spell rows fill them? What does the fold skip for an
  actor with a zero `+0x7c`, or when the view's `+0x3dc` lacks bit 2?
- DIV-1883: what are the five rectangles' pixel positions per attribute row,
  and what owns the byte at `[[view+0x5c]+0x104]`? What string does `R0807`
  produce for a signed value of four or more characters?
- DIV-1884: how do the record's `+0x14` and `+0x18` relate to the decoded
  width and height?
- DIV-1885: which control receives each of `dialogs.txt` 23, 75, 117 and
  `patch.txt` 52 to 54 through the setter or constructor, and which of the 52
  unreached inherited-getter tables are shown in play?
