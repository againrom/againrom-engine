# story1203 -- splitting docs/DIVERGENCES.md

## Intent

The owner's standing request: review the divergence ledger, delete closed
entries found during review, audit the open ones, saw the single file into
smaller granular files, and make the index thin. Quoted in full in the task
brief (Russian): "divergences надо отсмотреть, удалить все закрытые вхождения, открытые аудиторать и
распилить на более мелкие файлы, гранулярно, а индекс сделать тонким".

## As-built

`docs/DIVERGENCES.md` (794 826 bytes, 694 lines, 510 rows in one file) is now
an index of 7 753 bytes. The 510 rows moved to 32 files under
`docs/divergences/`, one per area, none over 58 657 bytes (ceiling ~60 000).
Three rows found `CLOSED` during the split's own read were deleted outright,
on the owner's direct instruction relayed in the brief, rather than moved to
`DIVERGENCES-CLOSED.md` as that file's own convention would do; Git keeps them
and this document names all three below. 507 rows moved into the split files.
Sixteen rows that failed strict parsing were repaired (not restated) by
relocating non-canonical content, not by inventing new text. `docs/1203/audit.md`
carries the owner-facing audit report this split was also asked to produce: it
makes no status or type change on its own authority.

## Proof

- `internal/divledger` is a new escape-aware ledger reader shared by
  `cmd/divcensus` and its own guard test, `TestLedgerRowsUnique`,
  `TestLedgerCitationsResolve`, `TestLedgerAreaFileSizeCeiling` in
  `internal/divledger/divledger_test.go`; each is described with what it
  checks and what it caught on a deliberately broken copy in the "Guard test"
  section below.
- `cmd/divreconstruct` rebuilds the (id, 9-cell) tuple set from the new split
  files and compares it against the same set read from `07cddcf`'s single
  `docs/DIVERGENCES.md` (extracted with `git show 07cddcf:docs/DIVERGENCES.md`
  to a scratch path and run as `go run ./cmd/divreconstruct -old
  <scratch-path> -new docs/divergences`). Its output is committed at
  `docs/1203/reconstruction-proof.txt`: 510 old rows, 507 new rows, `only in
  OLD (3)` naming exactly the three deleted ids, `only in NEW (0)`, and
  exactly the 16 repaired ids under `cell-level differences`, each naming the
  same column(s) "The 16 repaired rows" above names (`Type` alone for the ten
  Type fixes, `Reason, Status` for the six Status fixes) -- 491 common ids
  identical.
- Building `cmd/divreconstruct` against `07cddcf`'s own single-file ledger
  found a real, pre-existing structural quirk `internal/divledger`'s own
  first version did not handle: that file's "Divergences" section opens with
  15 individually spotlighted rows (`DIV-1329`, `DIV-1295`, `DIV-805`,
  `DIV-804`, `DIV-803`, `DIV-802`, `DIV-786`..`DIV-792`, `DIV-778`,
  `DIV-779`), each its own blank-line-separated one-row fragment, BEFORE that
  section's own `| ID | Subsystem | ... |` header line -- a header-driven
  parser silently drops all 15 rather than raise an error, since it reads
  each as "not yet the header, keep scanning" and never revisits them.
  `pipeline/check-div-claims.sh` never had this defect: its own awk reads
  `Type` and `Status` from hardcoded field positions (7 and 10) regardless of
  whether its own `want` variable has been set from a header line yet.
  `internal/divledger.ParseFile` now matches that reference behaviour --
  every row is mapped onto `Columns` by fixed position, and an in-file
  `| ID |` line is read as ordinary prose, never as a prerequisite -- which
  is what let `cmd/divreconstruct` see all 510 old rows. This was found and
  fixed during this split's own gate-running, not left as a discovered but
  unfixed defect.
- `pipeline/check-div-claims.sh` (seat file, unmodified) reads
  `docs/DIVERGENCES.md` alone, which this split turns into an index with no
  `DIV-` rows: run against this worktree's own final state, it now reports
  `EMPTY SELECTION`, exit 2. Its own escape-aware parsing algorithm is not
  the problem -- run mid-split, while `docs/DIVERGENCES.md` still held the
  pre-split 510-row content, it read the row format cleanly (507 live rows,
  745 distinct cited claim ids, exit 0). The seat patched it to read
  `docs/divergences/*.md` at seat `8c9ce16`, and it runs clean there; see
  "What the seat's scripts need" below.

## Open debt

- `pipeline/check-div-claims.sh` and `pipeline/next-div-id.sh` are seat files
  this lane could not edit. The seat patched both to read
  `docs/divergences/*.md` at seat `8c9ce16`, and both run clean.
- Four pre-existing `.go`-comment citations resolve to no row in either ledger
  (`DIV-067`, `DIV-101`, `DIV-223`, `DIV-1262`). They predate this split (the
  relocated allocation narrative below explains three of the four); this split
  left the `.go` comments untouched and instead recorded the gap in the new
  guard test's documented allowlist and in `docs/1203/audit.md`, since silently
  editing the comments would erase the only trace of the defect.
- `docs/1203/audit.md`'s three passes are bounded, not exhaustive, and it says
  so: part (a) read a sample of the silence-assertion candidates against a
  keyword search of the research claims corpus, not all 201 `UNKNOWN` rows
  against the full corpus; part (c) spot-checked one of 39 lexical candidates.
  Every unsettled row is listed with what was checked, per the brief's own
  "owner decides" instruction -- nothing was closed or re-typed by this split.

## Controlled vocabulary

The original ledger's `Subsystem` cell is free text: `<leading name>` or
`<leading name> / <detail>`. 142 distinct leading names existed across 510
rows. None is a typo needing correction; the split's job was to decide which
distinct names name the same area under a different word, and which distinct
`client` / `persistence` rows (two names covering 80 and 45 rows respectively,
too large for one file) split by their own detail text instead.

### Exact-alias merges

Merges are same-subject, different-word, and each is defensible by reading the
rows under both names -- no row's content changed to make a merge work. The
recurring patterns:

- `sim` and `simulation` -- literally the same word, abbreviated in some rows.
- The three SAV families the brief named: `current sav` / `native save` /
  `native saves` / `player save` all mean the file this build itself writes
  and reads (-> `persistence-current-sav`); `native city sav` alone stays its
  own file at 21 rows, already near the ceiling, and names the specific
  native-format city SAV rather than the general native-save family;
  `generated city sav` / `generated/current sav` / `town sav` / `current
  physical sav` / `native saved document` / `native original document` /
  `current original document` / `native area form92` / `generated first
  world` / `native mission` / `native campaign` / `source-backed application`
  / `imported city return` / `imported mission` / `imported human` / `load
  game` / `source city` all describe the export/import boundary between a
  generated city and its SAV form (-> `persistence-city-sav-and-imports`);
  `original sav` / `original mission load` / `original world load` /
  `original load` / `original mission` / `original diary` / `original
  non-party current profiles` / `original actor effect restoration` /
  `original-dead resume` / `canonical stacks` / `saved container and actor
  load state` all describe the original game's own save format and decode
  path, not this build's writer (-> `persistence-original-saves-format`).
- The three shop families the brief named: `shop` / `shop interior` / `town
  shop` are the same screen under three names (-> `shop`); `tavern` /
  `tavern interior` -> `tavern`; `school` / `school training` -> `school`.
- Entity-persistence names (`saved groups`, `saved building`, `saved
  formation`, `saved player formation`, `saved clocks`, `saved and native
  turning`, `ground sack`, `ground sack art`, `retained projectile`, `signed
  poison`, `structures`, `structure light sources`, `structure art`,
  `fountain`, `fire`, `terrain`, `terrain resource admission`,
  `regeneration`) all name one saved-object kind each, none large enough for
  its own file, all sharing one concern (what a save carries for a live
  entity) -> `persistence-entities`.
- `town` / `town ambience` / `town exterior` / `town square` / `town dialogue
  and responses` / `town and generator command plaques` -> `town` (the town
  screen, not a save format).
- `campaign` / `campaign script` / `campaign ending` / `hall of fame` /
  `earned score` / `score history` / `authored player control` / `script
  execution witness` / `script command17` / `mission start` / `mission
  trailer` / `generated mission20` -> `campaign` (mission-to-mission
  progression and scripted flow, as opposed to `combat-and-ai`, which is what
  happens inside one mission).
- `actor ai` / `ai` / `ai activity refresh` / `mission ai` / `combat` /
  `mission combat` / `damage reaction` / `mover rate` -> `combat-and-ai`.
- `ui` / `input` / `hover help` / `settings` / `sound options` / `graphics
  options` / `game options` / `smoothing` / `editor` / `data` / `front end` /
  `game` / `media menu` / `map command voices` / `.16 loader` / `shared
  dialogs` -> `ui-and-settings` (chrome and configuration, not a specific
  screen's content).
- `magic` / `quick spells` / `spellbook` / `book targeting` / `manual book
  spells` / `prismatic spray` / `spell delivery` / `combat and magic` ->
  `magic`.
- `world map` / `map view` -> `world-map`.
- `cutscene` / `movie audio` / `cutscene decoder and platform` / `cutscene
  requests and source selection` / `cutscene sidecars and placement` /
  `cutscene audio, focus and cadence` -> `cutscene`.
- `chrgen` / `character generation` / `chrgen and town` / `character
  generator` / `pre-create` -> `character-generation`.
- `audio` / `human voice` -> `audio`.
- `mission ui` / `mission rendering` -> `mission-screen` (kept separate from
  `client-mission-screen`: the two-row `mission-screen` file is the UI/render
  split for the mission HUD as a subsystem pairing, while
  `client-mission-screen` -- reached only through the `client` content split
  below -- is the mission screen's own layout elements; both are small and
  reading them together would blur a real distinction the rows themselves
  draw between "how the HUD is drawn" and "how the mission layout panel is
  built").
- `global healing target and scheduler` / `healing defaults and native
  compatibility` / `global healing across town return` / `player parameter
  transport` -> `healing`.
- `item identity` / `item-cell stars` / `runtime item construction` / `item
  inspection book titles` / `inventory` / `unit inspection` -> `inventory`.
- `research` (1 row) -> `misc`, its only member; not worth a one-row file
  under its own name.
- `dialogue`, `rendering`, `simulation` (already `sim`/`simulation` above)
  stand alone as single names.

### Content-based sub-split: `client` (80 rows)

One leading name, `client`, covered 80 rows -- far past the ceiling for one
file, and the brief required a split by actual content, not row count. Each
row's own `Subsystem` cell carries a second segment after ` / client / ` that
names what part of the client the row is about; that text was matched against
five keyword patterns, in this priority order (first match wins), against
`gen.py`'s `client_bucket()`:

1. Document-panel words (`document panel`, `documents panel`, `documents
   element`, `Ok_l_off`, `font4`, `line breaker`) -> `client-documents-panel`
   (7 rows).
2. Room-art words (`plaque`, `shipped 16-wide strip`, `navigation label
   font`, `shop button text`, `character panel body`, `tavern left column`,
   `tip panel text font`, `shop upper region`, `seam`) ->
   `client-room-presentation` (9 rows) -- the shared visual furniture of the
   town's interior screens (tavern/school/shop/chargen), as opposed to
   `tavern`/`school`/`shop`'s own files, which hold that screen's *behaviour*
   rows, not its *art* rows.
3. Cursor/selection words (`cursor`, `hover mask`, `selection rectangle`,
   `selection-summary`, bare `selection`) -> `client-cursor-and-selection`
   (10 rows).
4. Mission-layout words (`mission column`, `mission right column`, `mission
   command panel`, `minimap`, `control-panel`, `command panel`, `mission
   map`, `mission frame`, `statistics card`, `structure capabilit`,
   `edge-scroll`, `world-map cards`, `unit panel`) ->
   `client-mission-screen` (14 rows).
5. Key/input words (`key`, `keys`, `keyboard`, `accelerator`, `ctrl`, `f1`,
   `escape`, `letter`, `patrol`, `pause ... key/space`, `function-key`,
   `repeat`) -> `client-input-and-keys` (12 rows).
6. Everything else -> `client-presentation-other` (28 rows) -- the largest
   remaining bucket, genuinely miscellaneous client-presentation rows that
   matched none of the five named concerns; still under the byte ceiling
   (43 899 bytes) so left as one file rather than forced into a sixth
   arbitrary split.

### Content-based sub-split: `persistence` (45 rows)

The bare leading name `persistence` (not `native city sav`, `current sav`, or
any of the SAV-family names above, all already routed by their own leading
name) covered 45 rows. Its second segment was matched by prefix:

- Starting `original saves` (case-insensitive) -> `persistence-original-saves-corpus`
  (19 rows) -- the field-by-field census of the original saves corpus, as
  opposed to `persistence-original-saves-format`'s decode-path rows above.
- Starting `city sav` -> `persistence-city-sav-and-imports` (4 rows), joining
  the SAV-family rows already routed there by leading name.
- Everything else -> `persistence` itself (22 rows) -- versioning and upgrade
  rows that are about the persistence layer in general, not one format.

### Verification of the mapping

`namemap.py`'s run against the parsed 510-row set confirms: 142 distinct
leading names (matches the brief's measured starting state exactly); `client`
splits 28/14/12/10/9/7 = 80; `persistence` splits 22/19/4 = 45. Both sums match
each name's total row count with no row unaccounted for. `build.py`'s earlier
run over the same `ALIAS` table and content rules produced all 32 target files
under the byte ceiling before generation; `gen.py`'s actual run (the one that
wrote the committed files) asserts the row total is exactly `510 - 3` (the
three deleted `CLOSED` rows) before writing anything, and its own printed
per-file byte counts match `docs/DIVERGENCES.md`'s "Area files" table exactly.

## The three deleted CLOSED rows

Found `Status: CLOSED` during this split's own read of the single file at
`07cddcf`. Per the owner's direct instruction relayed in the brief, these were
deleted outright rather than moved to `DIVERGENCES-CLOSED.md` (that file's own
established convention for a row that closes after being read there) --
Git keeps the full row text in the `07cddcf` history and in this story's own
`docs/1203/reconstruction-proof.txt`, which reads the old single file at that
commit and preserves each row's full old text in the input side of its diff.

- **DIV-1254** -- movie audio / installed decoder. Said: cutscene audio was
  believed silent as a decoder limit; the actual defect was an instrument bug
  (a probe called `SmackGetTrackData` with its 2nd and 3rd arguments
  reversed), and once fixed, real per-frame PCM reaches the player with no
  known blocker. Closed as `FIDELITY-DEBT`; two residual differences moved to
  their own rows, `DIV-1255` (mixing/volume policy) and `DIV-1256`
  (timing/synchronisation policy), both still open in `docs/divergences/cutscene.md`.
- **DIV-880** -- generated city SAV / speed, load, capacity and regeneration
  periods. Said: the writer used to emit zero load for equipped native
  members and wrote bonus amounts as periods; fresh generation now writes
  actual worn weight, serialized pack accumulator, coherent load/speed/
  capacity, and correct modifier amounts, and `CityUpdate` preserves
  constructed values instead of recomputing them on load.
  Closed as `FIDELITY-DEBT`.
- **DIV-1304** -- global healing across town return. Said: the missing
  city-level `Player` healing-policy producer described in `SAV-726` is now
  implemented; `CityUpdate` writes the current owning `Player`'s percentage,
  imported parties retain theirs, and the policy and floors survive an
  original-city return and a cold SAV load. Closed as `FIDELITY-DEBT`.

## The 16 repaired rows

The brief's own hypothesis -- that the 6 rows without a parseable `Status` and
10 without a parseable `Type` shared a cell-count/escaping defect -- does not
match what the ledger actually contains. An escape-aware parser (this split's
own prototype, later the same algorithm as `internal/divledger`) found all 510
rows carry exactly 9 cells; running the seat's own `check-div-claims.sh`
against the worktree confirmed `badn=0`. The real defect in both groups is a
controlled-vocabulary violation, not a structural one:

**10 rows used a non-canonical `Type` value** instead of one of the five the
index defines (`UNKNOWN`, `CONFLICT`, `DEVIATION`, `HOTFIX`, `FIDELITY-DEBT`):
`DIVERGES` (1), `INTENTIONAL` (2), `OWNER-DIRECTION` (6), `AUTHORED CHOICE`
(1). Every one of these ten rows' own `Reason` cell names an explicit owner
instruction or an authored choice, never a bare unresolved research gap --
that is exactly the header's own definition of `DEVIATION` ("a deliberate,
disclosed difference from a claim that does exist, or the owner's own
instruction"). All ten were normalized to `DEVIATION`; no other cell in any of
these ten rows changed. Affected ids: `DIV-853`, `DIV-953`, `DIV-954`,
`DIV-1266`, `DIV-1269`, `DIV-1311`, `DIV-1313`, `DIV-1349`, `DIV-1359`,
`DIV-1360`.

**6 rows carried a parenthetical annotation appended to `Status`** instead of
the bare `OPEN`/`ACCEPTED`/`CLOSED` the column defines, e.g.
`OPEN (authority narrowed in story1164)` or, for the longer ones, a full
sentence describing what a later story closed. Content was not discarded: each
row's `Status` cell became the bare word `OPEN`, and the parenthetical's own
text was appended to that row's `Reason` cell as `NARROWED: <text>`, which is
where a narrowing note belongs under the header's own column definitions --
`Reason` is "why the row exists in that Type," and a narrowing is exactly a
refinement of that reason, not a new status. Affected ids: `DIV-730`,
`DIV-437`, `DIV-926`, `DIV-932`, `DIV-938`, `DIV-956`. No row's `Type` or any
other cell was touched by this fix.

`docs/1203/reconstruction-proof.txt` shows these as the only 16 cell-level
differences between the old single file and the new split, beyond the 3
row-level deletions.

## Four pre-existing phantom `.go` citations

Not introduced by this split; found while cross-checking `.go` citations
against both ledgers for `internal/divledger`'s `TestLedgerCitationsResolve`
guard and for `docs/1203/audit.md`'s part (b). Each resolves to no row in
either `docs/divergences/*.md` or `docs/DIVERGENCES-CLOSED.md`:

- **`DIV-067`** and **`DIV-101`** -- the relocated allocation narrative below
  says both were "returned unused." A returned id is never reissued (the
  index's own "Allocation" section states this), so a `.go` comment still
  citing either one predates the return and was never updated.
- **`DIV-223`** -- the relocated narrative describes this id as "spent" (a row
  was meant to exist), but no such row exists in either ledger despite its
  siblings `DIV-222`, `DIV-224`, `DIV-225`, `DIV-226` all existing. This looks
  like a genuine lost row, not a returned id.
- **`DIV-1262`** -- likely an off-by-one: a `.go` comment cites a range
  `DIV-1257..DIV-1262`, but the ledger's own ids in that neighbourhood stop at
  `DIV-1261`.

This split left every one of these `.go` comments untouched -- rewriting
`DIV-223`'s citation, for instance, would erase the only remaining trace that
a row was once meant to exist there. Instead, `internal/divledger_test.go`
carries a small documented allowlist so `TestLedgerCitationsResolve` stays
green without hiding the gap, and `docs/1203/audit.md` names all four again
under its own part (b). The owner decides whether to allocate replacement
rows, correct the comments, or leave them; this split does neither.

## Where the old header prose went

The old `docs/DIVERGENCES.md` carried 33 252 bytes of prose above its first
row (measured at `07cddcf`; this story's own extraction of the same span,
`git show 07cddcf:docs/DIVERGENCES.md | sed -n '1,129p'`, is 33 237 bytes --
the small difference is this extraction's own line range ending one paragraph
short of the exact byte boundary, not a discrepancy in content). Its still-
normative parts (what the ledger is, the two truths, the row format, the two
per-file headings, the escaping rule, corrected where it was already stale --
see below) moved into the new thin index at `docs/DIVERGENCES.md`, condensed
from prose into the table the index now carries. Its rationale and the
id-allocation chronology has no home in `pipeline/PROCESS-LOG.md` available to
this lane (that is the seat's own file); it is reproduced verbatim below
instead, exactly as extracted, as a frozen historical record. The new index
says where it went, in its own opening paragraph and its "Allocation" section.

One correction made in transit: the old header's own text (reproduced below)
claimed an escaped pipe was unreadable to the tooling. That was already wrong
by the time of this split -- `pipeline/check-div-claims.sh`'s own header
comment records that it has split on an unescaped `|` (protecting `\|` first)
since an earlier fix of its own, and three rows already in the ledger before
this split, `DIV-254`, `DIV-237` and `DIV-337`, already use `\|` correctly.
The new index's own "Escaping" section states the corrected rule and names
the fix; the false claim is not reproduced as current guidance, only
preserved below as history.

### Verbatim: the old file's header, as of `07cddcf`

The following is the exact text of `docs/DIVERGENCES.md` lines 1-129 at commit
`07cddcf` (extracted with `git show 07cddcf:docs/DIVERGENCES.md | sed -n
'1,129p'`, reproduced byte for byte, including its own outdated escaping
claim and its full id-allocation chronology). It is not current guidance;
current guidance is `docs/DIVERGENCES.md`'s own text.

```markdown
# DIVERGENCES.md — the single divergence ledger

Introduced by the owner's pipeline-v2 ruling (2026-08-15, full text in
`pipeline/archive/owner-ruling-2026-08-15-pipeline-v2.md` above this repo). Every known mismatch
between the implementation or the owner's intent and researched ROM1 behaviour is a row here. A
mismatch may not live only in a spec cut list, a story doc, a code comment, or chat history.

Optional or additive improvements beyond ROM1 are tracked separately in `docs/QOL.md`. A QoL row
also belongs here only when it replaces original behaviour or changes the default route; the QoL
label is not a way to hide a divergence.

Two truths are held separately. ROM1 truth is what research says the original does; againrom
intent is what the owner requires of this implementation. Where they diverge, the implementation
follows the owner and the divergence is recorded here. Authority for ROM1 truth: promoted claim >
provisional research > inference > this code. This code is never evidence of ROM1 behaviour.

Types: `UNKNOWN` (research absent or insufficient) · `CONFLICT` (owner directive contradicts a
promoted claim) · `DEVIATION` (deliberate difference from ROM1) · `HOTFIX` (temporary
implementation; the commit row stays in `docs/hotfix/LEDGER.md`, the behavioural divergence, if
any, lives here) · `FIDELITY-DEBT` (correct behaviour known, implemented otherwise for now).

Status: `OPEN` (divergence stands) · `ACCEPTED` (intentional, owner-ruled, no revisit planned) ·
`CLOSED` (implementation now matches ROM1 truth, or the claim was retracted).

A landing updates this file. A row cites claims, never experiments.

**Three tables, split on the owner's instruction of 2026-08-17.** The single table had reached 105
rows, and 48 of them were `UNKNOWN` — not differences from ROM1 but places where research is silent
and the implementation is authored. Reading them as divergences is what made the ledger look larger
than the debt it carries. A row is filed by what it is:

- **Divergences** — the implementation and researched ROM1 behaviour differ, or the owner has ruled
  against a claim. This is the debt.
- **Authored where research is silent** — no claim answers the question, so the implementation
  authored one. This is a research backlog, and its `Revisit condition` names the claim that would
  retire the row.
- **`DIVERGENCES-CLOSED.md`** — closed rows, moved out of this file and destroyed nowhere. A closed
  row is a record of what was once divergent and how it was settled.

No per-table row count is written here. The first version of this header carried three, and the next
landing made all three wrong within the day: it closed a row, opened two and retyped one. The tables
carry their own rows and a census is one command.

A row moves between the first two tables when its type changes, and to the closed file when its
status becomes `CLOSED`. Nothing is deleted.

**An `UNKNOWN` row's `ROM1 behaviour` cell goes stale silently.** It states what research did not
say on the day the row was written, and a later experiment can publish exactly that fact without
anything failing. On 2026-08-17 an audit found 27 live rows citing a claim that now carries a
retraction row, and four cells asserting research silence that had since been answered
(`TOWN-116`/`TOWN-117`/`TOWN-119`/`TOWN-120` for `DIV-104`, `TOWN-123` for `DIV-106`, `TOWN-118` for
`DIV-105`, `AI-CURSOR-126`/`AI-INPUT-121` for `DIV-084` and `DIV-088`).
`pipeline/check-div-claims.sh` now reports the citing rows; it cannot judge whether the retracted
clause is the one a row leans on, which is still read by hand.

A cell may not contain an escaped pipe. It renders, but it makes the row unreadable by column
position, which is how every census over this file counts types, and a row that cannot be counted
is a row nobody checks. Say the thing in words. The same applies to editing this file with a basic
regular expression: `\|` is alternation there, not an escaped pipe, and a `sed` written that way
rewrote five unrelated rows on 2026-08-15 before it was reverted.

Ids are allocated in the brief that will use them, and an id handed to a lane is spent whether or
not a row arrives: a returned number is never reissued. Returned unused so far, all 2026-08-15:
`DIV-045` (the F5 lattice key introduced no divergence, and the key itself has since been removed
at the owner's word), `DIV-049` (the shop's merge rule is part of DIV-046's own directive, not a
second divergence), `DIV-051` (`r3-orders` needed one row, not two), `DIV-060` (the shot marker was
a defect in this build's own consumer, not a divergence from ROM1), `DIV-061` and `DIV-062`
(the lattice asserts nothing about ROM1 on either of this build's own two surfaces, so moving it
from the placement surface to the corner mesh is not a divergence), `DIV-065` and `DIV-066` (the
chargen skill art ships all four states the owner named — three files plus the column's own baked
art for the fourth — so drawing all four is not a divergence from either source; the earlier row
claiming a missing fourth state was itself wrong and is withdrawn, not replaced), `DIV-067` (the
raise-after-unequip witness changed no behaviour, and both reasoned paths held), `DIV-069`
(the object and unit shadow paths carry no divergence; only the structure path does), and
`DIV-071` (routing a map-placed person through the recompute a party member already takes
closes a gap against this build's own behaviour, not against ROM1). `DIV-092` is NOT returned: it
carries a live `OPEN` row below (town shop / a direct shelf-pack drag, allocated to story 1005's
round 2 at its own adversarial review's second pass, 2026-08-16), and an earlier form of this
paragraph said otherwise — a self-contradiction this file's own allocation rule ("the next free
id is the one after the highest seen in this file or in this paragraph") could have carried
forward into a reissue of a live id, a returned id never being reissued in this project. `DIV-087`
through `DIV-091` were the five ids `1005`'s round 2 spent at its own first landing and its
first-pass review; `DIV-092` is the sixth, spent at the second-pass review below. Corrected
2026-08-16, at the round's own third-pass adversarial review. The spell-effects hotfix spent
`DIV-093` and `DIV-094`. Story `1008` spent `DIV-095` through `DIV-100`;
`DIV-101` and `DIV-102` were returned unused, needing only six of its eight-id allocation.
The teleport-explored hotfix spent `DIV-103`. Story `1007` spent `DIV-104`
through `DIV-107` for the world map, four of the seven ids it reserved; `DIV-108` through
`DIV-110` were returned unused, 2026-08-17. `1005`'s seventh pass spends `DIV-111` and its
twelfth pass spends `DIV-112`. Story `1006` spends `DIV-113` through `DIV-119`; `DIV-120` was
returned unused, and `DIV-121` replaces the school-order row that had collided with `1005`'s
`DIV-112`. The purchase-producer row that had collided with `1005`'s `DIV-111` is withdrawn:
`TOWN-GENERAL-107` and `TOWN-GENERAL-108` now establish that producer. `DIV-122` and `DIV-123` were
allocated to an over-broad reading of the staff-training and mission-20 carry claims and are
returned unused; that reading was re-audited by `EXP-0191` and authored no row. The
staff-training and mission-20 hotfix spends `DIV-124`. The mission-loss hotfix of 2026-08-17 spends
`DIV-125` and the autocast usefulness hotfix of the same day spends `DIV-126`. The ledger-audit story
`1009` (2026-08-18) is allocated `DIV-127` and `DIV-128`; both are spent — `DIV-127` for the shop
shelf animation's undecoded cadence and `DIV-128` for the world map's marker-paint-versus-selection
gap found while re-reading `TOWN-123` to settle `DIV-106`, which the same story closes as leaning on
a retracted claim. Story `1012` was allocated `DIV-134` to settle `DIV-011`'s notice-cadence
question; it is returned unused — `DIV-011` was corrected and re-cited in place, and the
measurement `1012` took did not surface a second divergence. Story `1010` (2026-08-18) was
allocated `DIV-129` through `DIV-131` for the world map's route construction; all three are spent —
`DIV-129` for the graph-unusable fallback, `DIV-130` for the marker-identity/animation gap deferred
alongside `DIV-128`, `DIV-131` for the least-length search's weight and tie-break — and the story
closes `DIV-104`, moved to `DIVERGENCES-CLOSED.md`. Its adversarial review added `DIV-135`, the
arrival mechanism `TOWN-121` decodes. Story `1011` (2026-08-18) is allocated `DIV-132` and `DIV-133`
for the shop tip widget; both are spent — `DIV-132` for the widget's own construction gate and its
unimplemented second text, `DIV-133` for its wrap rule and its layering against the pre-existing
message strip. The same story closes `DIV-018`, whose two named subjects (the shelf animation and
the tip widget) are now both drawn. Story `1013` (2026-08-18) was allocated `DIV-136` through
`DIV-139` for the world map's arrival, skip, reveal cadence and marker mechanics; three are
spent — `DIV-136` for the reveal/paint cadence, gated on wall-clock time in `pkg/ui/app.go` at a
value `TOWN-121` does not establish; `DIV-137` for the party's own current-position field, which
this build resets to the town start on every arrival in town rather than persisting it; `DIV-138`
for the selected-mission marker cache, which story `1054` now persists as a broader set containing
every selected positive mission rather than only `TOWN-123`'s picture-bearing population — and
`DIV-139` is returned unused,
having found no third technical fact distinct from the two `DIV-137` and `DIV-138` already record.
The same story closes `DIV-128`, `DIV-130` and `DIV-135`, moved to `DIVERGENCES-CLOSED.md`. Story
`1014` (2026-08-18) is allocated `DIV-140` and `DIV-141` for the RU menu accelerator's own
keyboard-to-byte mapping; both are spent — `DIV-140` for physical keyboard layout being outside
this build's own scope, `DIV-141` for two RU town rows whose labels fold to the same accelerator
byte. The same story closes `DIV-006`, moved to `DIVERGENCES-CLOSED.md`. Story `1015` (2026-08-18) is allocated `DIV-142` through `DIV-146` for the school's column art and its hit test; all five are spent, and none is returned. `DIV-142` is the sixteen shipped rotation frames of which this build draws two, `DIV-143` the five fields of `TOWN-138`'s picker-step reset that have no counterpart here, `DIV-144` the pure-black key on the thirty skill patches, `DIV-145` the two per-class rectangle arrays against `TOWN-068`'s one, and `DIV-146` the class-to-rotation-frame assignment, which is read off the shipped art because `TOWN-061` and `TOWN-067` disagree about the panel selector's polarity. At the story's landing on the same day, research `EXP-0195` answered four of the five. `DIV-145` and `DIV-146` are CLOSED and moved to `DIVERGENCES-CLOSED.md`: `TOWN-148` publishes two per-class rectangle arrays whose twelve values are this build's twelve to the pixel, and `TOWN-149` settles the panel polarity in `TOWN-067`'s favour and names the frame the selector picks. `DIV-142` and `DIV-144` were both written `UNKNOWN` because no claim then gave a rule, and both are re-typed `FIDELITY-DEBT` now that `TOWN-147` gives the column's advance rule and `TOWN-150` decodes both blits. The same story amends `DIV-121` twice — once for the hit test as well as the draw, and once at the landing for `TOWN-152`, which narrows the deviation to the internal array order. `DIV-147` is opened at the landing for the nine-frame diamond animation `TOWN-154` decodes and this build does not draw; it is outside `1015`'s contract and is not a gap in it. Story `1016` (2026-08-19) was allocated `DIV-148` through `DIV-152` for the town square's own art and its hit test. Three are spent: `DIV-148` for the mask code-to-door mapping, measured from art correlation and the shipped tip text rather than published by any claim; `DIV-149` for `town_add.bmp`'s own compositing offset being a measured no-op against the currently sampled art; `DIV-150` for the three door labels being drawn unconditionally, on no game state this build tracks. `DIV-151` and `DIV-152` are returned unused — the story's own findings did not surface a fourth and fifth distinct technical fact — and they stay retired rather than being reused: `pipeline/next-div-id.sh` counts mentions, not rows, so a returned id is already spoken for by this paragraph, and a later reader of the words `DIV-151` cannot tell which meaning was intended. At its landing the story opened `DIV-153` and `DIV-154` from the allocator instead, for the town decoration and the tip widget that its round-3 review found the ledger did not carry. Story `1017` (2026-08-19) was allocated `DIV-155` through `DIV-159` for the school, tavern, shop and character-generator button art and the once-per-mission offer rule; the contract's own premise that the shop's button art is unlocated was wrong (the art and its wiring already existed; the story only verified it). All five are spent: `DIV-155` for the shipped ON-bitmap paint's own second condition, whose writer is unread; `DIV-156` for the character generator's four measured button wells (amended round 2 from a round-1 finding of three, `DIV-156`'s own row) being mapped three to Play, Reset and Back by the existing code's own order and the fourth suppressed, since that screen ships no per-button art; `DIV-157` for the once-per-mission school and shop offer rule being owner-directed session state with no ROM1 repeat behaviour researched, amended round 2 for the decline case; `DIV-158` for the button bitmaps' own blit convention (`TOWN-150`'s two primitives), which the contract already flagged and which stays unpublished for buttons specifically; `DIV-159` for the school's and tavern's own button-to-action assignment (which shipped bitmap is Train against Exit, Hire against Talk against Exit), read off each room's own well order top to bottom rather than decoded. Round 3 (adversarial review, pass 2) amends `DIV-156` again, for the fourth chargen well's own treatment (drawn unmodified rather than covered by a fill that did not follow the plaque's own shape), and `DIV-157` again, for the mechanism gating re-offering (`Town.taken` alone, a second `Town.shown` latch found redundant at both its production read sites and removed); the behaviour either row describes is unchanged by its own amendment. Round 3 opens `DIV-165` for the school's, tavern's and shop's hardcoded English button labels, pre-existing and left alone this round per scope. Story `1018` (2026-08-19) was allocated `DIV-160` through `DIV-164` for the floating tip panel widget and its five screens. Three are spent: `DIV-160` for the tip suppression toggle's own store, a flat file beside `saves/` rather than `TOWN-186`'s registry location; `DIV-161` for the panel's close and toggle captions, authored English rather than the two `STRINGTABLE` strings `TOWN-185` names by id and leaves unread, since this repository has no `STRINGTABLE` decoder; `DIV-162` for the panel's own fill, frame and toggle art pairing and for three non-shop rects' authored height past the researched popup rectangles' own 200px, raised at this story's own landing after a regression check found the sourced heights, once this build's own close/toggle row is accounted for, silently truncated real shipped text on all five screens; the generator's own rect takes `TOWN-187`'s researched rectangle whole and carries no height deviation, corrected from round 3's own shorter rect at the landing (round-3 adversarial review, D-1). `DIV-163` and `DIV-164` are returned unused and stay retired rather than being reused, for the same reason `DIV-151`/`DIV-152` did: the story's own findings did not surface a fourth and fifth distinct technical fact, and `pipeline/next-div-id.sh` counts mentions, not rows, so a returned id is already spoken for by this paragraph. At the same landing the story closes `DIV-154` (the town square now draws the widget `DIV-154` found missing), moved to `DIVERGENCES-CLOSED.md`, and amends `DIV-132` (narrowed: the shop tip's construction is now gated on this build's own `TipsOff`, not established as the original's `[L03631]`) and `DIV-133` (the shop panel's rect now extends past `shopMessageRect` rather than clipping before it, since the panel now draws opaque and last and the bleed-through the clip guarded against cannot occur). Round 3 (adversarial review, pass 2) amends `DIV-162` again: round 2's own widened rects let a press or release fall through, at three of `app.go`'s four call sites, to a control the panel visually covered, reaching a door choice, a class commit and a skill spend the player could not see happen — story `1005`'s own shipped defect shape, found by round-3 review. The fix reverts every rect to its researched width and top-left corner and restores a uniform swallow at all four call sites; `cmd/tippanelcheck` (committed this round) replaces round 1's and round 2's own uncommitted scratch measurements and the row's own numbers are now sourced from it. Pass 3 of the same review found one further player-visible defect after the story's three-pass ceiling was spent — the swallow returns before three of the four call sites' own press bookkeeping, so a gesture crossing the panel's boundary leaves a latch armed — and it is carried as a hotfix rather than a fourth lane cycle (`docs/hotfix/LEDGER.md`); pass 3's seven documentation findings are fixed in place at the landing, `docs/1018-tips/closure.md`. Story `1018`'s own landing also opens `DIV-167`, which belongs to no story: `EXP-0199` (`TOWN-214`) and `EXP-0200` (`TOWN-222`) published the tavern button panel's own stored rectangles and their runtime origin addend after `1017` landed, and two of that story's three wells sit one row above them. At the pin bump to `202b8d5` that closes `1018`, `EXP-0203` amends two rows in place: `DIV-167`'s height convention is settled by `TOWN-258` (exclusive bottom, the school's own pair contiguous at 117), leaving only the instrument re-statement; and `DIV-166`'s school half is answered by `TOWN-257` and `TOWN-259` (the `+0x10` addend right-aligns a 160-wide bitmap in a 176-wide rect, nothing the widget draws reaches the left 16 columns, and the room's own background is what shows there), which refutes this seat's own seam-strip inference for the school and leaves the eight 16-wide bitmaps untouched. A 2026-08-19 hotfix for three owner-reported presentation defects (screenshots of the school, tavern, character generator and shop) spends `DIV-168` and `DIV-169`, both DEVIATION and both OPEN pending research: `DIV-168` for the seam-strip placement decision, made by rendering evidence (`cmd/plaqueseams`, committed) rather than by the withdrawn inference `DIV-166` records, and `DIV-169` for the character generator's navigation label font, changed to match the rooms' own on the owner's visual report with no claim naming which font object ROM1 itself binds there. The same hotfix amends `DIV-165`: its earlier reading of the shop's own `SHOP-SCREEN-035`/`SHOP-SCREEN-039` as establishing a fidelity-correct absence of a button label is withdrawn as an over-promoted negative, and the shop's four buttons now carry the same kind of hardcoded English caption the school's and tavern's already did. A same-session follow-up (2026-08-20) bumps the research pin to `6110aa1` for `EXP-0204`, whose `SHOP-050` decodes the shop button panel's own caption source and composition; the follow-up amends `DIV-165` a second time (the shop's four captions are now read through `ui.Words`'s install-text seam rather than hardcoded, on both roots, and the row's remaining debt is the school's and tavern's alone) and opens `DIV-170` for the one part of `SHOP-050` this build does not reproduce — the caption-and-number join on the Buy and Sell buttons, reverted after a rendered test showed it clips the number under this build's own text-layout primitive. The same pin bump's `TOWN-264`/`TOWN-265` (an `interface/` reference census) confirm `inn/RUOver.bmp` is one of seven of the eight 16-wide `interface/` bitmaps the original loads by literal path — `DIV-168`'s own rendering-based choice — while `inn/tav_09.bmp` is the one unreferenced entry among them; neither claim states where any of the seven is drawn, so `DIV-168` is not amended, only corroborated in its own text at the next edit. Story `1019` (2026-08-19) and the same session's plaque hotfix were together allocated `DIV-168` through `DIV-174`. Three are spent, all three by the hotfix and all three described above; `DIV-171` through `DIV-174` are returned unused and stay retired rather than being reused, for the reason `DIV-151`/`DIV-152` and `DIV-163`/`DIV-164` already record here. `1019` opened no divergence row of its own: its fix restores this build's own documented session-reset discipline at every site that installs a different game, and states no new mismatch with ROM1. Story `1021` (2026-08-21) was allocated `DIV-175` through `DIV-182` for the shared column pane abstraction, per-root tip heights and the character panel body. Two are spent: `DIV-175` for the shared character panel's own body bitmap, chosen by rendering rather than a decoded destination; `DIV-176` for the tavern's left-column bitmap order, authored from three converging signals against `TOWN-282`'s own unresolved coordinates. The story's own tip-height and seam changes amend `DIV-162` and `DIV-168` in place rather than opening new rows, since both describe the same underlying fact (an authored height with no decoded value; a rendering-chosen seam bitmap) at a wider set of call sites. `DIV-177` through `DIV-182` are returned unused and stay retired rather than being reused, for the reason `DIV-151`/`DIV-152`, `DIV-163`/`DIV-164` and `DIV-171`..`DIV-174` already record here. Story `1022` (2026-08-21) was allocated `DIV-190` through `DIV-198` for the character generator's canonical composition, nine ids because `DIV-189` was already spent by `1021`. The same story closes `DIV-189`, moved to `DIVERGENCES-CLOSED.md`. Four are spent: `DIV-190` for the four body-slot archive entries chosen by exact size match rather than a claim naming the reader; `DIV-191` for the compact card's own field arrangement, row set and font, including the WEIGHT row this build has no data source to fill; `DIV-192` for the detailed page's own transient hover and refusal copy, dropped rather than moved when its message box was removed; `DIV-193` for the DOLL/STATS toggle now replacing the doll with the same card in every room's character pane, reversing `1021`'s own design of showing both. `DIV-194` through `DIV-198` are returned unused and stay retired rather than being reused, for the reason `DIV-151`/`DIV-152` and the others already listed in this paragraph record here. Round 3 (adversarial review, pass 3, final) opens no new row: it fixes two player-visible defects in the card's own rendering — the persistent chevron/mode-box/Book chrome and the member-name row drawing over the card's now-populated rows in Statistics mode, and `AlignValues`' shared right edge overflowing the card's own width for a real leveled character's wider values — and amends `DIV-191`'s Implemented-behaviour cell in place, since that row's own ruling already covers both: "his same-day ruling that truncation is acceptable and the main information must fit" was recorded round 2 but not actually implemented until this round. Story `1023` (2026-08-22) was allocated `DIV-199` through `DIV-206` for the mission column's decoded geometry. Two are spent: `DIV-200` for the doll and worn boxes' own authored placement in the column's fourth slot (the doll kept, drawn under the character panel; the worn box dropped and never attempted during a mission); `DIV-201` for the minimap-above-control-panel assignment to ids 5 and 6, the owner's own ordering directive rather than a decoded identity. `DIV-199` and `DIV-202` through `DIV-206` are returned unused and stay retired rather than being reused, for the reason `DIV-151`/`DIV-152` and the others already listed in this paragraph record here. The same landing closes `DIV-213` and `DIV-218`, both moved to `DIVERGENCES-CLOSED.md`: `DIV-213`'s right-column-does-not-fit gap is discharged by building the strip at its decoded 160-pixel width and 158/80/242 division, witnessed by both previously-failing headless scenarios passing on both roots; `DIV-218`'s mission-panel WORN-row truncation is discharged by removing the code path it described rather than by a decode — the mission's live panel composes through `CompactPanelLayout`, which carries no WORN row, and `AuthoredPanelLayout` is left with no production caller anywhere in the tree. The same landing amends `DIV-217` in place: the mission screen's own panel composer is named as `CompactPanelLayout`, not `AuthoredPanelLayout`; the row's own substance, that no claim names the figure widget's own interactive controls, is unchanged. Story `1025` (2026-08-22) was allocated `DIV-222` through `DIV-229` for carried weight. Five are spent: `DIV-222` for the card's own fractional format and the field it reads, which is what stayed open when `DIV-209` closed; `DIV-223` for an item code no producer declared weighing nothing, where the original's weight travels with the item object; `DIV-224` for which entities carry a derived capacity; `DIV-225` for the twelve shipped weapon triples whose weight column resolves negative, reproduced rather than clamped; `DIV-226` for the penalty applying to the composed mover speed rather than to the actor's own term. `DIV-226` is the overload penalty's own place in the mover speed, opened when the spec was written against the shipped code. `DIV-227` through `DIV-229` are returned unused and stay retired rather than being reused, for the reason `DIV-151`/`DIV-152` and the others already listed in this paragraph record here. The same story closes `DIV-209`, moved to `DIVERGENCES-CLOSED.md`, and amends `DIV-191` in place for the row's own position in the order. Story `1028` (2026-08-22) was allocated `DIV-230` through `DIV-234` for the mission command panel. All five are spent: `DIV-230` for the three cells (Defend, Swarm, Retreat) this build ships disabled through the original's own skip mask, having none of those three orders; `DIV-231` for Patrol's own key, an addition outside the decoded eight-word table; `DIV-232` for the S and D letters, which collide between the panel's Swarm/Defend cells and the map screen's pre-existing Book/Doll display switches and are resolved in the switches' favour; `DIV-233` for the removal of this build's own authored WASD-style camera pan, freeing every letter for the panel's own accelerators and leaving the arrow keys and the pre-existing screen-edge pan as the camera's only input; `DIV-234` for the Cast cell, enabled and reachable but issuing no order because no decoded route lets the panel select a spell of its own. The same landing narrows `DIV-201`: id 6's own slot now draws the decoded eight-cell grid from its four shipped bitmaps, closing the drawn-content half of that row's debt, and the remainder is exactly `DIV-230` through `DIV-234`. The story's return visit after adversarial pass 1 (2026-08-22) opens `DIV-236` and `DIV-237`. `DIV-236` starts as a seat finding independent of the review, that the pick-up key's letter `B` collides with `text/main.txt`'s own naming of `B` for the shipped spellbook; the owner then ruled that a key of this project's own must not occupy or scramble a letter the original names, so pick-up moves to `F` instead of keeping `B`. `DIV-237` is the same ruling's second instance, found against the ruling itself: March, an invented order (0146), was bound to `R`, the original's own decoded Retreat key; March moves to `H` and `R` is left free for a future Retreat. Neither row disturbs `DIV-232` (Book/Doll keep S/D, ruled separately, the same day). Story `1029` (2026-08-22) was allocated `DIV-239` through `DIV-244` for the two check opcodes the campaign authors and this build could not evaluate. All six are spent and none is returned: `DIV-239` for check opcodes 4, 16 and 21, which stay unsupported and hold the whole of the milestone census's remaining 33 nodes; `DIV-240` for check opcode 9 answering 0 where the original faults on a null or stale pursuit target; `DIV-241` for check opcode 9 answering 0 for a pursued unit the map file never placed; `DIV-242` for authored map id 0 being read as no authored id, which `cmd/mapunitcensus` measures to be unreachable on shipped content; `DIV-243` for check opcode 12, decoded as byte-identical to 17 and authored zero times, left unimplemented on the story's own exclusion; and `DIV-244` for the arm's pursuit test being this build's `HasAttackTarget` rather than the original's order-object branch. Story `1031` (2026-08-22) was allocated `DIV-259` through `DIV-263` for the mission map's own cursor. All five are spent: `DIV-259` for the hostility projection reading the live session relation matrix rather than a view-side per-player record, a Medium-confidence link (`UNIT-VPLAYER-021`/`UNIT-VISBIT-044`); `DIV-260` for the minimap widget's own hit rectangle being this build's own `minimapCaptures`, since `AI-CURSOR-208`'s own widget-test rectangle is not established; `DIV-261` for `sdefend` having no armed state of this build's own to select it from; `DIV-262` for the two Ctrl/Alt modifier-key cursors (`swarm`, forced `move`) not being read by this build's input model; `DIV-263` for the `town` cursor not being implemented, both of `AI-CURSOR-209`'s own conditions being Unknown in game terms, and for the hostility mask being read as the hostility bit alone (this build's `MapEntity` carries no second bit). The same landing narrows `DIV-247`: sixteen of the 28 registered slots now reach the screen, up from three, and the row's own count is corrected there. Story `1031`'s return of adversarial pass 1 (2026-08-23) is allocated `DIV-270` through `DIV-273`. Three are spent: `DIV-270` for the hostile-hover `attack` selected without the original's three further gates, `DIV-271` for the authored pointer composition order, and `DIV-272` for the popup answering `default` rather than the decoded entry gate's no-change exit. `DIV-273` is RETURNED UNUSED: the third modifier latch it was held for belongs to `DIV-262`'s own subject and is corrected into that row in place. The same return corrects three counts in `DIV-247` and two statements in `DIV-262`. Story `1031`'s return of adversarial pass 2 (2026-08-23) was allocated `DIV-280` through `DIV-282`; **all three are RETURNED UNUSED**, because every finding of that pass is either a defect in this build's own code and tests (a draw guard that read a cursor NAME instead of which code path produced it, an unclamped marker rectangle, a test fixture that composed one of four layers it claimed to cover) or a correction to an existing row made in place. `DIV-270` is corrected: its research-basis cell said no published claim covers the second selection tree; `AI-CURSOR-209` covers part of the same tree (the `town` pick's own entry test and refinement, which reconverge with the `attack` write's own path before the write's own gates and do not gate it), and the row's revisit condition is narrowed to the one question still open, which of the two selection trees an ordinary hover reaches. `DIV-249` is corrected: its row read the mission map as an unscaled, native-resolution surface and compared a monitor-independent off-map figure (5%) against a monitor-dependent on-map one (1.7%, one named monitor); the mission map is a 1024x768 frame fitted to the window by the same mechanism as every other screen's 640x480 canvas, and the corrected, monitor-independent figures are 5% off the map against 3.1% on it, ratio 0.625. Story `1033` (2026-08-23) was allocated `DIV-264` through `DIV-269` for the three check opcodes and the paired instant that story implements. Two are spent and four are returned unused. `DIV-267` was opened at the story's first push for the structure field's initial value, on the premise that no claim gives it, and is CLOSED by adversarial pass 1 the next day: `ALM-CLS-053` and `SAV-BLDG-037`, both High and both in the story's own pin, give that value, and the row's own absence claim came from searching the ALM ledger the subject belongs to rather than the subject. `DIV-264` is opened by the same pass for the field changing only through instant 26, this build having no structure damage model. The same pass re-verified `DIV-239`'s closure, which stands, and corrected two of its cells that repeated the withdrawn absence claim. `DIV-265`, `DIV-266`, `DIV-268` and `DIV-269` are returned unused and stay retired rather than being reused, for the reason `DIV-151`/`DIV-152` and the others already listed in this paragraph record here. The next free id is the
one after the highest seen in this file, in `DIVERGENCES-CLOSED.md`, in this paragraph, or reserved to
unmerged work. Do not read that number off a summary: run `pipeline/next-div-id.sh`, which scans all
three sources and prints the file and line its answer came from.
```

## What the seat's scripts need

Applied at seat `8c9ce16`: both scripts now read `docs/divergences/*.md` and
run clean. The request as the lane wrote it follows.

Both are seat-owned files this lane cannot edit (per the brief). Both need
the change below, because `docs/DIVERGENCES.md` is now an index, not a
ledger, and neither one yet looks at `docs/divergences/*.md`.

- **`pipeline/check-div-claims.sh`**, line 95:
  `ledger="$impl/docs/DIVERGENCES.md"` -- this is the sole place the script
  names its input file. It needs to read every `docs/divergences/*.md` file
  instead (or in addition, if the seat wants the index itself scanned too,
  though the index now carries no `DIV-NNNN` rows). Run against this
  worktree's own final state as it stands (`AGAINROM_IMPL` pointed at it),
  it now reports `check-div-claims: EMPTY SELECTION -- the scan read
  nothing, which is not a pass`, exit 2 -- exactly the break this section
  describes, and the reason this is a real, not hypothetical, ask. The
  script is otherwise ready for the new files without any other change: its
  own `esplit` escape-aware split function, `want` cell-count check, and
  `CLOSED`-skip logic already match the split files' identical row format --
  confirmed earlier, mid-split, by running it while `docs/DIVERGENCES.md`
  still held the pre-split 510-row content: `check-div-claims: selected 507
  live row(s) of 510 (3 closed), citing 745 distinct claim id(s)`, exit 0.
- **`pipeline/next-div-id.sh`**, lines 33-37 (the `sources` array): currently
  `docs/DIVERGENCES.md`, `docs/DIVERGENCES-CLOSED.md`,
  `docs/hotfix/LEDGER.md`. Needs `docs/divergences/*.md` added (a glob, since
  the script's own `grep -noE 'DIV-[0-9]+'` per-source scan already handles
  multiple files for `docs/hotfix/LEDGER.md`'s own directory). Confirmed via
  `pipeline/ALLOCATIONS.md`'s own DIV floor (`grep -n "^| DIV " pipeline/ALLOCATIONS.md`,
  currently 1363) that this is not an active break today: the floor already
  exceeds the highest id in `docs/divergences/*.md` (1361), so
  `next-div-id.sh`'s answer is unaffected by the split for as long as that
  stays true. It stops being true the first time a lane allocates past 1363
  without moving the floor, which is exactly the failure mode
  `pipeline/alloc-sweep.sh` exists to catch -- but that script brackets the
  allocation floor and ledger edits, not this split's own directory move, so
  it would not by itself notice `next-div-id.sh` no longer scanning the split
  files. This is a real, if currently dormant, gap the seat should close.

## Guard test

`internal/divledger/divledger_test.go` adds three checks, run as part of
`go test -trimpath -count=1 ./...`:

- `TestLedgerRowsUnique` -- every `DIV-NNNN` id across `docs/divergences/*.md`
  is unique. Demonstrated failing: a scratch copy of
  `docs/divergences/misc.md` with its one row's id changed to `DIV-035`
  (already present in `docs/divergences/simulation.md`) makes the test fail
  with a duplicate-id report naming both files; restoring the original file
  makes it pass again. Not committed (a deliberately broken file has no place
  in the tree); the failing run's output is quoted in the final report.
- `TestLedgerCitationsResolve` -- every `DIV-NNNN` referenced in a `.go` file
  under the repository resolves to exactly one row in the union of
  `docs/divergences/*.md` and `docs/DIVERGENCES-CLOSED.md`, except the four
  documented pre-existing gaps above (`DIV-067`, `DIV-101`, `DIV-223`,
  `DIV-1262`), which are named in the test source with the same explanation
  given above rather than silently allowed.
- `TestLedgerAreaFileSizeCeiling` -- no file under `docs/divergences/` exceeds
  60 000 bytes (the brief's own "~60KB" ceiling, made a concrete constant
  so a future row addition that pushes a file over it fails loudly instead of
  silently regrowing a single-file ledger by another name).

## What was not done

- Release tests (`check-release-tests.sh`) and the milestone script-gap
  census (`pipeline/check-milestone.sh`) were not run. This story changes only
  documentation and a new Go guard/reconstruction tool; it touches no game
  screen, no shipped data, and no save behaviour, which is exactly the "Change
  / Required final evidence" table's own criterion for when those gates apply.
- `DIVERGENCES-CLOSED.md` itself was read (for size and targeted `grep`) but
  not modified. The three rows this split deletes are removed outright rather
  than appended there, which is a deliberate deviation from that file's own
  established convention, explained above under "The three deleted CLOSED
  rows."
- The audit in `docs/1203/audit.md` reports; it does not re-type or re-status
  any row. That decision is the owner's, per the brief.
