# Story `1035` — Valuable Documents: the panel, the collection, and the item in the player's pack

**Contract, seat, 2026-08-23.** Base `1fcf7e0f`, research pin `d7ee0c6`.
Branch `1035-valuable-documents`, worktree `wt-1035`. Stories `1034` and `1036` run in parallel in their own
branches on the owner's 2026-08-23 direction, so whichever of the three lands second and
third merges `master` first.

**Owner directive, 2026-08-23: no new experiments.**
«Новые эксперименты не запускай. То что не известно - прими решение без экспериментов.»
Every unknown this story meets is decided here or by the lane and recorded as a divergence
row. Nothing holds waiting for research. Research stays the sole authority on what ROM1
does; where it is silent, the decision is AUTHORED and the row says so.

**Owner directive, 2026-08-23: build fast.**
«Делайте истории быстро без супер тщательности. Все равно потом до 3 адверсарных ревью.»
Do not build exhaustive proof into the first pass. Evidence honesty is not on that axis:
claim less, never verify less than you claim.

**Divergence ids reserved: `DIV-297` through `DIV-306`.** Allocated with `pipeline/next-div-id.sh`
against a written reservation in `PIPELINE-STATUS.md`, re-run, and the answer confirmed
moved to `DIV-317`. **When the range is spent, stop and ask the seat.** Do not take the next
free number from inside this worktree: an unmerged branch is in none of the ledgers the
allocator scans, and two other lanes are open.

**Story `1032` has landed**, so the save paths this story extends are on `master` at the base
above. Re-measure them rather than reading this contract's description of them.

## Result

The player reads the campaign's documents. He starts a new campaign with **Valuable Documents** in his
pack, uses it, and the documents panel opens over the game: a full-page sheet with the current
document, a left and a right arrow that page through it and then move to the next document, and an OK
button that closes it. Mission 10's three text documents are there because mission 10 granted them,
and they are still there after a save and a load.

The pointable result is the screen itself, in `builds/current/`, on both language installs, with the
shipped text drawn in the install's own words.

## Why this story exists

Owner directive, 2026-08-23: build Valuable Documents — the screen itself, the recording of data into
it, the placement of that item at character creation, and the rest of what the feature needs.

`EXP-0151` decoded the whole feature and `EXP-0154` decoded what does and does not place the item.
Nothing in this build implements any part of it.

**One requirement is a deliberate deviation and the contract says so up front.** Research establishes
that no shipped map, script, `Humans` equipment row, character row or shop generation places the
access item: `ITEM-DOC-053` searched three producer families exhaustively and found zero, `DAT-DOC-021`
found no `Quest` cell in all 909 `Humans` equipment cells and no MagicItems arm in the positional
grammar, `SHOP-DOC-029` shows stock generation cannot provide it at campaign start, and `ITEM-DOC-054`
predicts the panel does not open at mission 10 start in the original. The owner has directed that the
player receive it at character creation. That is owner directive over a positive research finding, it
is what the story builds, and it carries a **DEVIATION** row with the **Owner directive** cell filled.

## The corpus

Read every row whole with `go run ./tools/claim <ID>` from the worktree's `research/`.

| Row | Subject | Note |
|---|---|---|
| `MENU-DOC-009` | the panel whole: eleven bitmaps, three hit rectangles, 21-line pages, one entry point | High for assets, geometry and page step; **Medium** for "one entry point in the image" |
| `MISSION-DOC-021` | a document's value formats straight into a resource path; the collection is campaign state a save carries | High for the path rules and the record's field set; **Medium** for the save location |
| `REG-SCN-097` | `AddTextDocument` / `AddPictureDocument` are one append onto one campaign-lifetime collection | High for the corpus and the kind assignment; **Medium** for "only grows" |
| `ITEM-DOC-053` | one item raises the panel, its identity, its gate, and that nothing shipped places it | High for the gate and the identity; **Medium** for "nothing places it" |
| `ITEM-DOC-054` | mission 10 starts with document content and no access item | **promoted**; Medium for the complete start absence |
| `DAT-DOC-021` | `MagicItems[28]` is the item's data identity and `Humans` cannot author it | **promoted**; High |
| `SHOP-DOC-029` | shop generation cannot provide it at campaign start | **promoted**; High |
| `ITEM-NAMEKEY-037` | the packed item code and the name-key formatter | cited by `ITEM-DOC-053` for `0x0e1c` |
| `TEXT-API-007` | font4, the font the panel's text is drawn with | |
| `DLG-WRAP-009` | the wrap the text document is put through | |

**Three of the rows carry a Medium that this story must respect rather than round up.** "Nothing
places it", "only grows" and the save location were each scoped to a search, and a spec that states
them as settled facts is the failure this project has recorded twice.

## The five behaviours

### B1 — the collection is campaign state, and a save carries it

`AddTextDocument` and `AddPictureDocument` in the campaign registry are one append onto one
campaign-lifetime collection of `(value, kind)` pairs, deduplicated on the pair. `REG-SCN-097` gives
the shipped corpus, identical on both roots: `[Mission10] AddTextDocument {1,2,3}`, `[Mission50]
AddTextDocument {4}`, `[Mission60] AddPictureDocument {1}` — four text elements over two sections and
one picture element over one, of 24 `[Mission<n>]` sections.

The loader runs only for a mission strictly higher than the current one, so the collection only grows.
That clause is **Medium** — the guard is one compare read in a decompiled caller and the mission
sequence was not driven. Implement the guard and record the grade.

`MISSION-DOC-021` gives the serialized form: `u32 count`, then `count` pairs of dwords, value then
kind. Its Medium is the **location** in the original's save, not the field set. This build has its own
save; write the collection into it in this build's own way and do not transcribe an unattributed
32-byte trailer.

### B2 — the panel

`MENU-DOC-009`, whole. `interface/Docs/sheet.bmp` at 640×480; left arrow at `(0,200,56,240)`, right at
`(576,200,636,240)`, OK at `(560,416,604,448)`, each rectangle equal to its bitmap's own pixel size.
Three states per arrow — idle, hover, pressed — from the `00`/`01`/`11` files; four OK bitmaps of
which `Ok_on` is selected by no arm the row read. Draw order: sheet, left arrow, right arrow, OK, then
the current document at the panel origin. OK closes and posts `0x445`.

**Paging is one control over two axes and the order matters.** An arrow steps the page by 21 lines,
bounded by the document's line count; the document changes **only when the page step returns zero**.
One arrow pair therefore walks pages first and documents second. A build that pages documents directly
is a different screen.

### B3 — the two document kinds load from their own paths

`MISSION-DOC-021`: kind 0 formats `graphics\interface\Docs\%d.bmp` and kind 1 formats
`main\text\Docs\%d.txt`, with the first path component naming the archive. The five shipped values
resolve 5/5 on both roots with no unused file and no unresolved value.

A text document is read whole and wrapped into lines; the panel draws 21 of them per page with font4,
with the pitch taken from the glyph sprite. A picture is blitted at the document rectangle's top-left
`(92,72)` at natural size — `1.bmp` is 464×344, which is **8 px wider than the 456-px rectangle**, and
that overhang is the original's own behaviour, not a bug to correct.

**The text is install text.** On a Russian install it comes from that install's own `docs/*.txt`. This
is the one screen in the story where `pkg/ui/words.go`'s membership rule does not apply, because the
words are the document's, not the program's.

### B4 — the item raises the panel

`ITEM-DOC-053`'s gate, in this build's terms: an item of class 14 whose row is congruent to 28 modulo
32, with the two other bit tests the row names. The original masks five bits, so rows 28, 60, 92, 124,
156, 188, 220 and 252 all pass for any material nibble — **reproduce the mask, not the single row**,
and say so in `spec.md`, because a build that tests `code == 0x0e1c` is a narrower gate than the
original's and the difference is invisible until content uses another row.

`MENU-DOC-009` says the shower runs under `campaign+0x3dc == 1`, one phase gate. Establish what that
phase is in this build's terms. If it cannot be established, the panel opens wherever this build can
open it, and the gap is a divergence row rather than a guess.

### B5 — the player starts with it

A new character is generated holding **Valuable Documents** in his pack. Not worn, not equipped:
`DAT-DOC-021` establishes that the `Humans` equipment grammar has no MagicItems arm, so an equipment
slot is the wrong place for it and this build should not invent one.

Its price is `-1` and its params are `[-1,5]`. A negative price is the original's own exclusion from
every live shop pool. **Check that this build's shop and sell paths honour a negative price** before
handing the player an item he can sell for nothing or, worse, for a negative amount.

The English name is `Valuable Documents`, the Russian `Официальные Документы`, and both come from the
install's own name table through the existing key path — not from a literal in our source.

## What is deliberately not in this story

- **Authoring new documents.** The story ships the five the campaign already carries.
- **The `Ok_on` bitmap's own arm**, which `MENU-DOC-009` says is selected by nothing it read.
- **The original's save location for the collection.** This build writes its own save.
- **Any other MagicItems row.** The gate admits eight rows; only row 28 has shipped content.
- **The panel's own entry point being unique.** `MENU-DOC-009` is Medium on that and this story does
  not close it.

## Ceiling

**Three adversarial passes.** That is `AGENTS.md`'s own ceiling for a story, not a decision this
contract makes.

**The chain ends at the first pass with no P finding, before the ceiling.** `AGENTS.md` classes
every finding and only one class returns a story: **P**, the player sees it wrong or hashed
simulation state is wrong. **W** -- production is correct and the witness cannot see it -- is a
ledger row. **D** -- production and witness are correct and a document says something untrue -- is
fixed in place or is a ledger row. Neither returns the story. A pass that finds no P therefore
closes the chain, and its W and D findings are applied without another round.

The trade is deliberate and its price is named in the same rule: stopping at the first pass with no
P finding would have shipped one defect that `1005` found on its twelfth pass. A hotfix costs one
ledger row and no lane; a review round costs a lane and a review. While a round returns fewer than
about one player-visible defect in three, it costs more than it prevents.

Returning the story to apply a pass's findings is not a pass. The ceiling counts passes, not fixes.

## Domains

Client, Campaign & Session, Persistence, Data, and Inventory & Equipment. Five. The ceiling is
three, as the Ceiling section states.

## The decisions taken without an experiment

Under the 2026-08-23 directive each of these is settled by the lane rather than asked of research,
and each is a row.

- **The phase gate `campaign+0x3dc == 1` in this build's terms.** `TOWN-352` reads that field as a
  bitmask with mission = 1 and a shop opened from a mission = 3. Decide from that row; if this build
  has no equivalent state, the panel opens wherever this build can open it and the row says so.
- **The save location for the collection.** `MISSION-DOC-021` is Medium on the original's location
  and High on the field set. Write this build's own form with the row's field set. DEVIATION.
- **`Ok_on`'s own arm**, selected by nothing `MENU-DOC-009` read. Decide a state for it or leave the
  bitmap unused, and say which. UNKNOWN.
- **Whether the panel's entry point is unique.** `MENU-DOC-009` is Medium. Build the one entry point
  the row gives and record the grade rather than closing the question.

## Divergence rows

The range is `DIV-297`..`DIV-306`, reserved in `PIPELINE-STATUS.md`. At least these are expected:

- **DEVIATION, Owner directive filled** — the player starts with the access item, which no shipped
  producer places in the original. This row is the story's own headline divergence and states the
  three searches that establish the original's silence.
- the phase gate, if `campaign+0x3dc` cannot be established in this build's terms;
- the save location, since this build writes its own form rather than the original's;
- anything the 640×480 panel needs that this build's surface model does not give it directly
  (`DIV-249` is the precedent).
