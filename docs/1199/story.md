# A typed SAV name is encoded, not refused (SAV-ENDGAME M5)

A player who typed anything past printable ASCII into this build's own SAVE dialog
got a refusal, on both roots: `SAV labels support printable ASCII only; choose
AGS or a Latin save name`. A Russian player naming a save in his own alphabet
had no SAV option at all. That rule was this build's own invention and
stricter than the original's:

- `TEXT-SAVELABEL-057` (High for the range read) establishes that the SAVE
  label's own copy primitive filters no byte value but `0x00` within the
  disassembled range; every byte `0x01..0xFF` it copies is stored unchanged.
- `TEXT-SAVELABEL-055` establishes that `TEXT-NAMEIN-024`'s capped input class
  — the one that drops every byte below `0x20` and caps at ten — is not the
  SAVE label's input class; its vtable is referenced zero times in the EN
  image (High for the search itself; Medium that this rules the class out
  entirely, per the claim's own grade).
- `SAV-SAVELABEL-1018` finds a non-printable byte, `0xE0`, as live
  pre-terminator label content in a lawfully-produced save already in the
  preserved corpus — the owner's own saves already violate the rule this build
  used to enforce.

The typed name is now encoded into Windows-1251, shifted by the selected
install's own language selector, instead of refused, through
`textinput.EncodeRune` — the exact rune-to-install-byte mechanism DIV-019
already draws every other installed string through, reused here rather than a
second encoder. THIS IS ONE SHARED WINDOWS-1251 TABLE, NOT A DISTINCT PAGE PER
INSTALL (correction, adversarial pass): a Cyrillic name is therefore accepted
on the EN install exactly as on RU, and a Latin name Windows-1251 has no byte
for at all (`café`) refuses on both, EN included. The earlier wording here
("the running install's own single-byte code page") overstated what the code
does; this file and DIV-1339 are both corrected. That refusal now names the
ASCII code point rather than the rune itself, so the message stays legible
even when the refused rune cannot be drawn: `SAV label cannot represent
U+00E9; choose AGS or a different name` (was `%q`, which put the undrawable
rune inside its own error message). NUL and every control byte still refuse,
matching the original's own one filtered value.

AT SELECTOR 1 ONLY, a further 47 Windows-1251 code points are refused too —
correction-pass finding, not part of the original landing: every value in
`0x80..0xAF` except the one Windows-1251 itself leaves unassigned (`0x98`),
Cyrillic `Ё` among them. Each one's own unshifted Windows-1251 byte sits
exactly where `render/text.Convert`'s own glyph shift lands a letter from
`0xC0..0xEF`; storing one unshifted, as the pre-correction code did, silently
took over that letter's own slot. A Russian player cannot give a SAV label
containing `Ё` (lowercase `ё`, at `0xB8`, sits outside the colliding range and
is unaffected).

## In-scope behaviour

- `textinput.DecodeByte` (`pkg/formats/textinput/input.go`) is `EncodeRune`'s
  own exact inverse for every byte this build's own `EncodeRune` can still
  produce. `EncodeRune` REFUSES, at selector 1 only, every one of the 47
  Windows-1251 code points named above (correction pass;
  `TestEncodeRuneRefusesTheSelectorOneCollisionSet` enumerates all 47 and
  asserts the count). For a byte this build did NOT write — a genuinely
  original `.sav`, or data from before this fix — `DecodeByte` still cannot
  tell a shifted letter apart from an unshifted foreign Windows-1251 byte and
  always reads the shifted letter
  (`TestDecodeByteStillPrefersTheShiftReadingForAForeignByte`, replacing the
  deleted `TestDecodeByteHasTheDocumentedSelectorOneCollision`, which pinned
  the same numeric fact before `EncodeRune` refused the colliding runes on
  write); that remains DIV-1339's own pinned choice, not a round-trip defect,
  since `EncodeRune` can no longer produce the colliding byte itself.
  `TestDecodeByteInvertsEncodeRuneAtSelectorZero` and
  `...AtSelectorOneOverTheShippedAlphabet` check the round trip over the full
  stored-byte range at selector 0 and over ASCII+Cyrillic (`ё` included, `Ё`
  excluded for the reason above) at selector 1.
- The SAVE dialog's own browser and the game-menu LOAD window now show the
  SAME label text for the same `.sav` row (correction pass; previously they
  disagreed). Both read through `OriginalStore`/`SaveStore.List()`, which
  decode through `textinput.DecodeByte`, but the LOAD window's own row list
  (`ConfigureSaveSeams`'s list closure, `pkg/game/savedialog1173.go`) passed
  every row through `drawableLabel` (`pkg/game/resume.go`) first — which
  ASCII-mangles any non-ASCII rune to `?` — then restored the unmangled
  decoded label only where `!IsOriginal(row.Name)`, i.e. only for `.ags` rows.
  Every `.sav` row stayed mangled in that one window while the SAVE dialog's
  own browser, which never calls `drawableLabel`, showed the same file's label
  correctly. The override now rebuilds the labels map for every row
  unconditionally, tokenizing a `.sav` name through `localOriginalSaveToken`
  first so it matches the key `list()` already produced.
  `drawableLabel`'s own doc comment, which said no substitution beyond `?`
  would be honest for a local `.sav` row, is corrected: that was true before
  `DecodeByte` existed and is false now that a `.sav` row's own label decodes
  to real runes.
- `asciiLabel`/`OriginalSaveLabel` (`pkg/game/originalsave.go`) decode a label
  byte through `textinput.DecodeByte` under a caller-supplied selector instead
  of turning every byte past ASCII into `?`. For a label this build wrote
  itself, the round trip is exact by construction. For a genuinely original
  `.sav` this build never wrote, the selected page is still a CHOSEN DEFAULT,
  not a decoded fact — the question the pre-story doc already carried ("which
  single-byte page these labels use is NOT decoded") is narrowed only for this
  build's own writes; see DIV-1339.
- `SaveStore.Selector` and `OriginalStore.Selector` (`pkg/game/savestore.go`)
  thread the install's language digit through to every label reader
  (`List`), the same value `text.Font.Selector` already carries for drawing.
  `ConfigureSaveSeams` (`pkg/game/savedialog1173.go`) is the one seam that
  knows which install is actually running and sets both from
  `f.textSelector()` (`pkg/game/font.go`, new: pulls the running FrontEnd's
  own selector out of `f.Font.Selector` once).
- `savLabelRefusal1173` (`pkg/game/savedialog1173.go`) is unchanged in name
  and call sites — story1198 factored the rule into exactly this one function
  so this story would have a single place to replace — but its body now
  encodes and its signature changed from `func(label string) error` to
  `func(label string, selector int) (string, error)`, returning the encoded
  label. It stays the ONE function every SAV producer the dialog can reach
  calls before building anything:
  `playerCitySave1173` calls it with the typed label; `playerMissionSave1198`
  calls it once for `ExportCurrentWorldSave` (which assigns its label argument
  straight into the document with no encoding of its own) and separately lets
  `playerCitySave1173` re-derive the same encoding from the same typed label
  for its own fallback producers — same function, same selector, deliberate
  redundancy on purpose, not a second copy of the rule.
- AGS keeps the typed name as Unicode; only the SAV path narrows it, because
  only SAV is the original's own single-byte format.

## Authority

Every claim above is from `EXP-0372` (landed, published at k51; current
knowledge pin is k54). No other experiment is cited. `TEXT-SAVELABEL-054`
(Medium) — used in the Unknowns below — is from the same experiment. The
correction-pass findings (the 47-code-point collision, the LOAD-window fix,
and the `EncodeRune`/`Convert` shift-composition check) are all derived from
this repository's own already-shipped code — `textinput.EncodeRune`,
`textinput.DecodeByte`, and `render/text.Convert` — read and, where stated,
executed directly; none of them cite a new claim.

## Touched surfaces

- `pkg/formats/textinput/input.go`, `input_test.go` — `DecodeByte` (new, prior
  pass) and its tests. Correction pass: `EncodeRune` refuses the 47-value
  selector-1 collision set instead of storing over a shifted letter's slot;
  both functions' doc comments corrected — `EncodeRune`'s for the collision
  and for the `Convert` composition (see the Unknown below); `DecodeByte`'s
  for the now-accurate refusal boundary.
  `TestEncodeRuneCoversEveryRepresentableStoredByte` updated for the refusal
  branch; `TestDecodeByteInvertsEncodeRuneAtSelectorOneOverTheShippedAlphabet`
  extended with `ё`, `Ё` excluded with reasoning;
  `TestDecodeByteHasTheDocumentedSelectorOneCollision` deleted, replaced by
  `TestEncodeRuneRefusesTheSelectorOneCollisionSet` (enumerates all 47) and
  `TestDecodeByteStillPrefersTheShiftReadingForAForeignByte` (keeps the same
  numeric pin for a byte `EncodeRune` no longer writes).
- `pkg/game/font.go` — `(*FrontEnd).textSelector()` (new, prior pass).
- `pkg/game/originalsave.go`, `originalstore_test.go` — `asciiLabel`,
  `OriginalSaveLabel` take a selector and decode through it; tests updated for
  decoded (not `?`-marked) Cyrillic output at both selectors.
- `pkg/game/savestore.go` — `SaveStore.Selector`, `OriginalStore.Selector`
  (new fields), threaded into both `List` methods.
- `pkg/game/savedialog1173.go` — `ConfigureSaveSeams` sets both stores'
  selectors; `savLabelRefusal1173` re-typed to encode; `playerCitySave1173`
  uses the encoded label. Correction pass: the list-override closure inside
  `ConfigureSaveSeams` now rebuilds decoded labels for every row, `.sav`
  included (was `.ags`-only); `savLabelRefusal1173`'s doc comment and its
  refusal message corrected (framing and `%U` in place of `%q`).
- `pkg/game/resume.go` — correction pass: `drawableLabel`'s doc comment
  corrected to say a local `.sav` row DOES reach it and IS now restored
  afterward, not left at `?`; no behaviour change in this file itself, the fix
  is in `savedialog1173.go`'s override, above.
- `pkg/game/missionsave1198.go` — `playerMissionSave1198` encodes once for
  `ExportCurrentWorldSave`, passes the original typed label to the
  `playerCitySave1173` fallback.
- `pkg/game/savenamed1173.go`, `savenamed1173_test.go` — `listSaveDirectory1173`
  takes and forwards the selector.
- `pkg/game/release_integration_test.go`, `savformations1159_test.go` — call
  sites updated for `OriginalSaveLabel`'s new selector argument (`0`, matching
  the fixtures' own EN-corpus origin).
- `pkg/game/savlabel1198_test.go` — `TestSaveLabel1198RefusalIsOnePlace`
  rewritten for encode-not-refuse (still-refused names, still-writable ASCII,
  a Cyrillic name that must now write and must differ by selector);
  `saveLabel1198MissionBranchRefusesBeforeProducing` extended with a Cyrillic
  case through the dialog's mission producer, closing the round trip with
  `OriginalSaveLabel` on the bytes actually written, not a synthetic fixture.
  Correction pass: doc comments corrected for the same framing fix as above;
  the final assertion changed from `strings.HasPrefix(decoded, cyrillic)`,
  which also passes on trailing garbage, to an exact match against
  `OriginalSaveLabel`'s own known suffix for this snapshot's mission number.
- `docs/DIVERGENCES.md` — DIV-1230 amended in place (correction paragraph,
  following story1198's own established convention); DIV-1339 opened, then
  corrected directly in this same still-open correction pass (not
  append-only: it is this story's own row, not a past landing) for the same
  framing overstatement and for the 47-collision consequence.
- `pkg/game/savedialog1173_release_test.go` —
  `TestReleaseSaveDialog1173TownPairOverwriteAndRefusal`'s stale
  "Cyrillic must refuse" assertion split into a genuinely-unrepresentable
  refusal case and a Cyrillic success case, found by this lane's own
  `check-release-tests.sh` run; see Proof. Correction pass, three further
  fixes to that same rewrite: (1) the unrepresentable-name case moved to its
  own `unrepresentableDir` and, after its refusal, retries with an empty AGS
  edit to restore the witness that a refusal keeps the player's typed
  directory and name — lost when the prior pass moved the Cyrillic success
  case to a directory of its own; (2) the Cyrillic case's decoded-label check
  changed from `strings.HasPrefix` to an exact match against
  `OriginalSaveLabel`'s own known town-save suffix; (3) a LOAD-window
  regression witness added — after the Cyrillic SAV write, opens the
  game-menu LOAD window and asserts its one row's text equals the same
  decoded label the SAVE dialog's own browser showed, then returns to the
  SAVE screen before the test continues.

## Proof

Unit level, asset-free, `go test -trimpath -count=1 ./...` (all packages
pass, no assets needed):
`TestDecodeByteInvertsEncodeRuneAtSelectorZero`,
`...AtSelectorOneOverTheShippedAlphabet`,
`TestEncodeRuneRefusesTheSelectorOneCollisionSet`,
`TestDecodeByteStillPrefersTheShiftReadingForAForeignByte`,
`TestDecodeByteRefusesWhatEncodeRuneNeverProduces`,
`TestEncodeRuneCoversEveryRepresentableStoredByte` (`pkg/formats/textinput`);
`TestALabelIsMadeDrawableThroughTheInstallsOwnPage` (`pkg/game`);
`TestSaveLabel1198RefusalIsOnePlace` (`pkg/game`).

Release level, run explicitly on both roots
(`AGAINROM_ASSETS=.../gameversions/en` and `.../ru`,
`AGAINROM_SAVE_CORPUS=.../gameversions/saves`):

- `TestReleaseSaveLabel1198MissionBranchRefusesBeforeProducing` — both
  subtests pass on EN (measured `textSelector()` = 0) and on RU (measured
  `textSelector()` = 1): a name outside Windows-1251 (Arabic) still refuses
  with `SAV label cannot represent` at either root; a Cyrillic name writes
  through the battlefield branch with no fallback notice, the written bytes do
  not contain the typed UTF-8 text, and `OriginalSaveLabel` applied to those
  exact written bytes at the same selector recovers the typed name exactly
  (exact-match, not prefix — see Touched surfaces).
- `TestReleaseSaveDialog1173TownPairOverwriteAndRefusal` — re-run on both EN
  and RU after the correction-pass fixes above, all subtests including
  `selected_repeated_suffix/{AGS,SAV,BOTH,multiple}` pass on both roots. This
  is the test carrying the LOAD-window regression witness and the
  refusal-keeps-typed-name witness.

`gofmt -l .` — clean. `go build ./...` — clean. `scripts/check-no-game-assets.sh`
— clean.

Milestone census (`pipeline/milestone-baseline.txt`'s canonical row; not
required for this story, since it touches no script/simulation execution
code, run anyway as this lane's own "run the game" check):
`go build -o <scratch>/mr1199.exe ./cmd/missionrun`, then `-mission 10 -trace
-ticks 1` and `-mission 20 -trace -ticks 1`, grep-counted for `UNSUPPORTED`,
on both roots:

| root | mission | UNSUPPORTED (measured) | baseline |
|---|---|---|---|
| en | 10 | 0 | 0 (no "cannot run" row for en/m10) |
| en | 20 | 0 | 0 (no "cannot run" row for en/m20) |
| ru | 10 | 0 | 0 (no "cannot run" row for ru/m10) |
| ru | 20 | 0 | 0 (no "cannot run" row for ru/m20) |

Unchanged on both roots, both missions, matching `pipeline/milestone-baseline.txt`'s
content exactly (no "cannot run" line for any of the four): expected, since
this story touches no script compilation or execution path. Not re-measured
this correction pass — nothing in it touches script or simulation execution
either.

This lane's prior pass ran `check-release-tests.sh` itself once, before this
correction, and fixed what it found (the stale "Cyrillic must refuse"
assertion). This correction pass does NOT start a second
`check-release-tests.sh` run of its own: the seat runs the full chain over EN
and RU on the merge commit itself, and a partial run this lane could not
finish would be worse than none. The two `TestRelease*` tests named above were
each run directly and explicitly on both roots instead, per this correction's
own brief.

## Unknowns carried forward, not answered here

1. **How the original DRAWS the label, and which page a genuinely original
   `.sav`'s label was actually written in.** `TEXT-SAVELABEL-054` (Medium)
   establishes only that the byte-indexed converter `R0793` is not
   called by any of the nine traced SAVE/LOAD dialog functions, directly or
   through its one wrapper, with vtable dispatch excluded by the same scan; it
   does not establish what mechanism, if any, draws the label instead, and no
   claim establishes an original label's own source encoding. `asciiLabel`'s
   choice of the running install's own selector for a label this build did not
   write is an authored default, not a decoded fact. Filed as DIV-1339
   (UNKNOWN). A further experiment on this exact question exists and is
   presently under its own correction pass; this story does not cite it, its
   claims, or its figures, and carries the question as Unknown rather than
   assuming its outcome.
2. **Whether a non-ASCII label survives an original resave.** Not a static
   read — only an owner run establishes it. Requested below.
3. **Whether `EncodeRune`'s selector-1 shift lands on the RIGHT atlas glyph
   through `render/text.Convert`, for uppercase Cyrillic and lowercase а-п
   (48 of the 64 code points selector 1 shifts).** Correction-pass finding,
   checked by direct execution, not assumed: `render/text.Convert`
   (`pkg/render/text/text.go`) is the one site that converts a stored byte
   before it selects an atlas record (`(*Font).index`,
   `pkg/render/text/text.go:156`); its own doc states the shift exactly —
   `0x80..0xAF -> +0x30 -> 0xB0..0xDF`, `0xE0..0xEF -> +0x10 -> 0xF0..0xFF`,
   every other byte untouched. Composing `EncodeRune`'s own selector-1 shift
   with `Convert`'s, for every Windows-1251 code `v` that selector shifts:
     - `v` in `0xF0..0xFF` (lowercase р-я, 16 values): `EncodeRune` stores
       `v-0x10`; `Convert` shifts that back by `+0x10`, returning exactly `v`.
       Checked for all 16 values — this branch composes to identity.
     - `v` in `0xC0..0xEF` (uppercase А-Я and lowercase а-п, 48 values):
       `EncodeRune` stores `v-0x40`; `Convert`'s own shift on that stored
       byte is `+0x30`, not `+0x40`, and returns `v-0x10`, not `v`. Checked
       for all 48 values — a flat, uniform `-0x10` offset across the whole
       range (`0xC0`→`0xB0` through `0xEF`→`0xDF`), not a partial or
       edge-case mismatch.

   This does NOT make Againrom's own display self-inconsistent: nothing in
   `render/text` calls `textinput`, or the reverse, and drawing a decoded
   label re-runs the same `EncodeRune` on the same rune before `Convert` ever
   sees it (`ui.load_draw_test.go`'s own wiring, `textinput.EncodeRune(r,
   font.Selector)`, is the general shape every text draw uses) — so a label
   reads back and redraws as the same glyph it was typed and displayed as,
   live, every time, regardless of this shift's own relationship to `Convert`.

   What is NOT established is whether the shipped atlas's own glyph record at
   the landed index — `Convert(EncodeRune(v,1),1)-FirstChar` for one of those
   48 values — is genuinely the letter Windows-1251 assigns to `v`, some
   other legible symbol, or something else: that needs the atlas's own glyph
   bitmap data, which this lane did not inspect (inspecting it would not
   itself violate the no-game-asset rule, since nothing needs to enter a
   tracked file to look, but doing so was outside this correction's bounded
   scope and this lane did not attempt it). `EncodeRune`'s selector-1 shift is
   DIV-019's existing mechanism, reused — not introduced — by this story for
   SAV labels; this lane did not change its constants, since doing so without
   the atlas evidence above risks trading a proven arithmetic inconsistency
   for an unproven one, on a shift several other installed-string call sites
   also depend on. `EncodeRune`'s own doc comment names the composition
   precisely (corrected, this pass) rather than the "stores the inverse of
   Convert's own glyph shift" claim it made before, which was true only for
   the 16-value branch and false for the other 48.

   The existing owner-run request below is the most direct way this resolves:
   its step 3 already asks whether the ORIGINAL game draws an
   Againrom-written Cyrillic SAV label correctly, incorrectly as different
   letters, or as garbage (Levels A/B/C) — a Level B result for a name using
   uppercase or а-п letters would be consistent with this finding. Recorded as
   open debt below; proposed as a candidate for its own separately scoped
   story or experiment once the atlas or the owner run gives evidence either
   way, not attempted blind in this correction pass.

## Owner-run request (M5's own proof half)

This lane cannot drive the original game. Requested, in
`pipeline/SAV-OWNER-RUNS.md`'s vocabulary:

1. On the RU install, open this build's own SAVE dialog (from the town or mid-
   mission) and type a save name that mixes Cyrillic letters with ordinary
   ASCII. Confirm.
2. Where this build's SAVE dialog previously refused any non-Latin name
   outright, it should now accept this one and write a file — note whether it
   did.
3. Load that exact `.sav` file in the ORIGINAL, unmodified RU game (not
   Againrom). Read what the original's own chooser shows for that save's row.
   Level A: the Cyrillic letters render correctly. Level B: they render as
   different, wrong characters (a wrong page was chosen for drawing). Level C:
   they render as garbage or placeholder glyphs. If the typed name mixed
   uppercase and lowercase Cyrillic, note whether the two cases disagree in
   level — Unknown 3 above predicts they could.
4. From inside the original game, resave that exact file (or save over a new
   slot from the same state) and load the new file back in the original.
   Confirm whether the name is unchanged.
5. Report the level reached and whether the resave preserved the name. No
   save name, install byte, or screenshot enters any tracked file — describe
   what was seen in chat only.

## Open debt

- DIV-1339 stays OPEN: which page a genuinely original label uses, and what
  draws it, are Unknown; the owner-run request above and any future decode of
  the drawing path are what close it.
- The owner-run half above is unperformed. No claim that the shipped RU game
  itself accepts and correctly redraws a dialog-written non-ASCII label should
  be made until it runs.
- Unknown 3 above (the `EncodeRune`/`Convert` shift-composition mismatch for
  48 of 64 selector-1 code points) is new this correction pass, unresolved,
  and not one of the four items this pass was returned for. It is recorded
  here as debt and proposed as a candidate for a separately scoped story or
  experiment; it was not fixed blind in this pass.
- DIV-1340 through DIV-1343 (reserved for this story) are returned unused —
  only DIV-1339 was needed; DIV-1230 was amended in place rather than given a
  new row, following story1198's own established convention.
