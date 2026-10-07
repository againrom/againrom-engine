# DIVERGENCES.md -- the divergence ledger index

Introduced by the owner's pipeline-v2 ruling; the full text and this file's own
allocation history live in `docs/1203/story.md`, moved there when the ledger
was split into `docs/divergences/`. Every known mismatch between the
implementation or the owner's intent and researched ROM1 behaviour is a row
somewhere under `docs/divergences/`. A mismatch may not live only in a spec cut
list, a story doc, a code comment, or chat history.

Optional or additive improvements beyond ROM1 belong in `docs/QOL.md` instead. A
QoL row also belongs here only when it replaces original behaviour or changes
the default route; the QoL label is not a way to hide a divergence.

## Two truths, held separately

ROM1 truth is what research says the original does; againrom intent is what the
owner requires of this implementation. Where they diverge, the implementation
follows the owner and the divergence is recorded here. Authority for ROM1
truth: promoted claim > provisional research > inference > this code. Againrom
code and tests are never evidence of ROM1 behaviour.

## The row format

Every area file under `docs/divergences/` carries the same nine columns, in
this order:

| Column | Meaning |
|---|---|
| ID | `DIV-NNNN`, allocated once and never reissued, even if returned unused |
| Subsystem | free-text area / topic, informal, not a controlled vocabulary |
| Owner directive | the owner's own instruction, when one exists (`-` otherwise) |
| ROM1 behaviour (claims) | what research establishes, with claim ids; may state research is silent |
| Implemented behaviour | what this build actually does |
| Type | `UNKNOWN` (research absent or insufficient) &middot; `CONFLICT` (owner directive contradicts a promoted claim) &middot; `DEVIATION` (deliberate difference from ROM1, including the owner's own instruction) &middot; `HOTFIX` (temporary implementation; the commit row stays in `docs/hotfix/LEDGER.md`) &middot; `FIDELITY-DEBT` (correct behaviour known, implemented otherwise for now) |
| Reason | why the row exists in that Type |
| Revisit condition | what would close or narrow the row |
| Status | `OPEN` (divergence stands) &middot; `ACCEPTED` (intentional, owner-ruled, no revisit planned) &middot; `CLOSED` (matches ROM1 truth or the claim was retracted) |

A row cites claims, never experiments: no cell names an `EXP-` id as authority.
`TestLedgerRowsCiteNoExperiment` refuses a row that does, except the rows
listed in `internal/divledger/testdata/experiment-citation-allowlist.txt`, a
list that only shrinks.

Within each area file, rows are grouped under two headings, carried over from
the single-file ledger's own earlier split into these same two tables:

- **Divergences** -- implementation and researched ROM1 behaviour differ, or the
  owner ruled against a claim. This is the debt. Types here are `CONFLICT`,
  `DEVIATION`, `HOTFIX` and `FIDELITY-DEBT`.
- **Authored where research is silent** -- no claim answers the question, so the
  implementation authored one. This is a research backlog; a row's own
  `Revisit condition` names the claim that would retire it. Most rows here are
  `UNKNOWN`; a minority are `DEVIATION` (a disclosed difference from a claim
  that does exist, or the owner's own instruction) or `FIDELITY-DEBT`.

`DIVERGENCES-CLOSED.md` (unchanged by this split) holds rows whose status
became `CLOSED` before this split. A CLOSED row found during this split's own
audit was deleted outright instead, on direct owner instruction, rather than
moved there; `docs/1203/story.md` names the three and what they said. Nothing
else is deleted: a row moves between the two headings when its Type changes,
and to `DIVERGENCES-CLOSED.md` when its Status becomes `CLOSED`.

## Escaping

A cell may contain a literal `|` written as `\|` (GFM's own table escape). Both
`pipeline/check-div-claims.sh` and `internal/divledger` split on a `|` NOT
preceded by `\`, so an escaped pipe reads correctly. This replaces an earlier,
now-wrong instruction in this file's own prior text that called an escaped pipe
unreadable; `DIV-254`, `DIV-237` and `DIV-337` already use `\|` correctly and
predate the correction. A cell must not contain a bare, un-escaped `|` --
that genuinely shifts every cell to its right, which is what
`pipeline/check-div-claims.sh` and the `TestLedgerRowsUnique`/
`TestLedgerCitationsResolve` guards in `internal/divledger` both check for.

## An UNKNOWN row's ROM1 cell goes stale silently

It states what research did not say on the day the row was written, and a
later experiment can publish exactly that fact without anything failing.
`pipeline/check-div-claims.sh` reports a row citing a claim that has since
carried a retraction. It cannot see the other half: a row whose cell asserts
silence that a brand-new, uncited claim has since broken. That half is read by
hand, area by area; `docs/1203/audit.md` is one such pass and names what it
checked and found.

## Finding a row

Search by id across every area file:

```
grep -rn "DIV-<id>" docs/divergences/
```

Or build a census (row/type/status counts, per-area breakdown) with
`go run ./cmd/divcensus`, which reads every `docs/divergences/*.md` file.
`docs/1203/story.md` names the controlled vocabulary this split used to assign
an area from each row's own free-text Subsystem cell.

## Allocation

Ids are allocated in the brief that will use them and spent whether or not a
row arrives; a returned number is never reissued. Do not guess the next free
id: run `pipeline/next-div-id.sh`, which scans `docs/DIVERGENCES-CLOSED.md`,
`docs/hotfix/LEDGER.md`, every `docs/divergences/*.md` file and
`pipeline/ALLOCATIONS.md`'s own floor.

## Area file size ceiling

An area file may not pass 60000 bytes; `TestLedgerAreaFileSizeCeiling` fails
the build that does. A file within one row of the ceiling (rows run to about
5600 bytes) is split by Subsystem text into a new `docs/divergences/<area>-<topic>.md`
in the same commit that adds the row, and the new file is added to the table
below; `TestIndexListsEveryAreaFile` fails when the table and the directory
differ.

## Area files

Run `bash scripts/divergence-area-counts.sh` for each file's physical and
open/accepted row counts and byte size. The index lists file ownership without
copying those counts.

| File |
|---|
| `docs/divergences/audio.md` |
| `docs/divergences/campaign.md` |
| `docs/divergences/character-generation.md` |
| `docs/divergences/client-cursor-and-selection.md` |
| `docs/divergences/client-documents-panel.md` |
| `docs/divergences/client-input-and-keys.md` |
| `docs/divergences/client-mission-screen.md` |
| `docs/divergences/client-presentation-other.md` |
| `docs/divergences/client-room-presentation.md` |
| `docs/divergences/combat-and-ai.md` |
| `docs/divergences/combat-and-ai-attack-and-hold-orders.md` |
| `docs/divergences/cutscene.md` |
| `docs/divergences/dialogue.md` |
| `docs/divergences/healing.md` |
| `docs/divergences/hover-help.md` |
| `docs/divergences/inventory.md` |
| `docs/divergences/magic-area-movement-cost.md` |
| `docs/divergences/magic.md` |
| `docs/divergences/magic-spell-geometry.md` |
| `docs/divergences/magic-spellbooks-and-quick-spells.md` |
| `docs/divergences/misc.md` |
| `docs/divergences/mission-screen.md` |
| `docs/divergences/mods.md` |
| `docs/divergences/persistence.md` |
| `docs/divergences/persistence-city-sav-and-imports.md` |
| `docs/divergences/persistence-current-sav.md` |
| `docs/divergences/persistence-current-sav-actions.md` |
| `docs/divergences/persistence-current-sav-player-save.md` |
| `docs/divergences/persistence-entities.md` |
| `docs/divergences/persistence-native-city-sav.md` |
| `docs/divergences/persistence-original-saves-corpus.md` |
| `docs/divergences/persistence-original-saves-format.md` |
| `docs/divergences/rendering.md` |
| `docs/divergences/rom2.md` |
| `docs/divergences/school.md` |
| `docs/divergences/shop.md` |
| `docs/divergences/simulation.md` |
| `docs/divergences/simulation-cast-and-pursuit.md` |
| `docs/divergences/tavern.md` |
| `docs/divergences/town.md` |
| `docs/divergences/ui-and-settings.md` |
| `docs/divergences/world-map.md` |
