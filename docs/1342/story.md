# 1342 — Typed Cyrillic font records

## Result

No drawn glyph changes on EN or RU. The recorded composition debt (selector 1
stores a typed Windows-1251 byte `t` 0x40 or 0x10 lower and `Convert` then adds
0x30 or 0x10, a flat `-0x10` over 48 of 64 code points) is the original's own
arithmetic, not a defect. The records it reaches hold the typed letter. The
knowledge pin moves from k168 to k173. Tests now state the rule per selector.
`EncodeRune`'s comment no longer calls the composition a mismatch.

## Authority

- `TEXT-108` (High): input conversion at selector 1 stores `0xC0..0xEF` as
  `t-0x40` and `0xF0..0xFF` as `t-0x10`; with the display conversion the typed
  byte reaches record `t-0x30` and `t-0x20`. A shipped string byte passes the
  display conversion alone.
- `TEXT-109` (High for font1 and font2, Medium for font4 and font5): those
  records draw the typed letter; 17 of 18 Latin/Cyrillic homoglyph pairs are
  pixel-identical in font1 and font2. The generator's name field draws with
  font4.
- `TEXT-110` (High / Medium): at selector 0 a typed byte reaches record
  `t-0x20`; 16 of the 64 bytes `0xC0..0xFF` draw their own letter, 32 another
  Cyrillic letter, 16 a blank record.
- `TEXT-COLL-025`: a typed `0x80..0xBF` is stored unchanged by the original.
- `HERO-CHARGEN-083` (amended) and `SPR16A-FONT-020`, `-022` (narrowed) touch
  this story only through the RU `font2` difference, which blanks 35 non-Cyrillic
  records. No engine citation depends on a retracted clause.

k169..k171 changed no claim this story cites, and k173 leaves TEXT-108..110 unchanged. The citation gate resolves 2414 ids except
`TOWN-508`, which also fails at k168. `check-div-claims.sh` flags three more
rows than at k168 (DIV-1276, DIV-532, DIV-121); each cites a claim narrowed to a
vtable address or a field name that the row does not use.

## As built

- `pkg/formats/textinput/input.go`: comment only; no code constant moves.
- Behaviour kept where a claim is Medium or Unknown: the keyboard code page
  (Windows-1251 on both selectors), the font4 and font5 letter identity, the
  refusal of typed bytes `0x80..0xAF` on selector 1.
- `docs/divergences/character-generation.md`: DIV-2330 (refused `0x80..0xAF`,
  `Ё` among 47), DIV-2331 (record bound in a 64-record font), DIV-2332
  (keyboard code page), DIV-2333 (font4 and font5 letters).
- Player-visible change on EN: none.

## Proof

- `pkg/game/typedrecord_test.go`: over `0xC0..0xFF` at selector 1 the stored
  byte and the record equal the `TEXT-108` rule; at selector 0 they equal
  `TEXT-110`; every shipped string byte `0x20..0xFF` reaches the display
  conversion record; typed `0x80..0xBF` stores as stated.
- `pkg/game/typedrecord_release_test.go` (EN and RU installs): each of the 64
  letters typed into the production generator's name field stores one byte that
  reaches the claimed record in the name font, the generator font and the
  default font; 17 of 18 homoglyph pairs match in font1 and font2 on RU and 4 of
  18 on EN; a SAV written with all 64 letters keeps the label bytes the rule
  gives and decodes to the same text on reload.

## Open debt

Keyboard code page (DIV-2332); letters of font4 and font5 by pixel identity
(DIV-2333); which surface draws with font3 (DIV-2331); whether original EN text
holds a byte `0xC0..0xFF` (`TEXT-110` Unknown).
