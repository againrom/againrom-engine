# A mod mission's texts are written in UTF-8 and the game encodes them

## Intent

A mod author writes a mod mission's texts as UTF-8 files: event pages, tip
pages, title, briefing and the world map's mission text (`briefmap.txt`). The
game encodes each into the running install's own bytes, so one mod folder shows
English on the English base and Russian on the Russian base. `cmd/modmission`
copies an installed mission's texts out as UTF-8. The map editor's report warns
before it saves into a mod about a script message the mod gives no page. The
story started from main `f75abc34` (pin k275).

## Authority

Owner direction (map editor, priority; audit item I7). Claims: `MISSION-TEXT-005`
(`Send message` takes a number; the page is `text/battle/m<mission>/event<NN>.txt`),
`TRIG-MSGCORPUS-049` and `TRIG-MSGLIMIT-050` (the shipped corpus and what a
consumer may change), `TRIG-MSG-023`, `TOWN-044` (the world map reads
`title.txt` and `briefmap.txt`), `TRIG-TIPS-087` and `TRIG-TIPS-088`. Rows:
`DIV-3215` to `DIV-3220`.

Choices the claims leave open:

- Encoding. Each rune goes through `textinput.EncodeRune` under the install's
  language selector, the encoder `EncodeInstallText` already uses for `title`
  and `briefing`. A control byte (CR LF, tab) is kept, because `EncodeRune`
  refuses it. A rune with no byte is dropped (`DIV-3216`).
- Not UTF-8. A file that is not valid UTF-8 is served unchanged (`DIV-3217`).
- `briefmap`. A file of the `texts` folder, not a text key (`DIV-3218`).
- Decoder. `textinput.DecodeByte`, the exact inverse of `EncodeRune`. The
  headless report's `decodeInstallText` (`vfs.DecodeText`, CP866) is not an
  inverse and is not used for the copy (`DIV-3219`).

## As built

- `EncodeInstallTextFile` and `DecodeInstallTextFile` (`pkg/game/installtext.go`).
  `SetModMissions` serves every `texts` file through the first; `title` and
  `briefing` keep `EncodeInstallText`. There is one serving route and one
  per-rune encoder.
- `AssembleModMission` (`cmd/modmission`) writes event, tip, title, briefing and
  briefmap pages through `DecodeInstallTextFile`; a file that does not decode
  back byte for byte is copied as the install holds it.
- Rule `mod-message-text` in `mapRules` (warning): when the editor saves into a
  mod and the mod's mission entry names the map, each message number the
  compile's raise list holds (255 excluded, once per number) needs an
  `eventNN.txt` in the entry's `texts` folder. A replacing mission keeps the
  install's page for a number the folder leaves out, so the install's page also
  counts. This narrows the brief's wording ("no text in the mod's folder") for a
  replacing mission, where the install's page is shown.
- `docs/MODS.md` and `CHANGELOG.md` state the authoring rule.

## Proof

Installed corpus, `main/text/battle/m*/*.txt` through the production
filesystem, decoded and encoded back byte for byte:

| base | files | bytes | event | tips | title | briefing | briefmap | bytes above 0x7f | not round-tripping |
|---|---|---|---|---|---|---|---|---|---|
| EN | 318 | 67599 | 225 | 9 | 28 | 28 | 28 | 0 files | 0 |
| RU | 321 | 61133 | 228 | 9 | 28 | 28 | 28 | 319 files, 58 distinct values | 0 |

No installed file with bytes above 0x7f is valid UTF-8, so a legacy copy is
never read as UTF-8. Tests (`pkg/game/modtexts_test.go`,
`modtexts_release_test.go`):

- Cyrillic and Latin-1 accents encode to the install's bytes on selector 0 and 1;
  the accent is dropped; a legacy file is served unchanged and does not alias
  its source.
- A replacing mission 20 with UTF-8 `eventNN.txt` and `briefmap.txt` shows, on
  EN and RU, the install's bytes through `openDialogue` for a message the
  mission's script raises and through the world map's scroll; without the mod
  both show the installed pages.
- The copy of missions 10, 20 and 110 encodes back to the installed bytes
  (38 files on each base).
- The report names a message of mission 10's map once per number when the
  copied pages are removed and is quiet with them.

## Open debt

- An accented Latin letter has no byte on the English base; whether the English
  font draws one is not established (`DIV-3216`).
- The editor window cannot type texts; the mod author edits the files.
