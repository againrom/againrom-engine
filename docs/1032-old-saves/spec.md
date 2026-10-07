# Story `1032` spec — saves written by older builds load again

Canonical as of the landing. This document describes the behaviour actually shipped, not the
original intention in `contract.md`.

## Domains touched

Persistence (`pkg/sim` byte form, `pkg/game` save paths), Sim Core (`formatVersion`,
`UnmarshalBinary`), Client (the load-time notice). Three, the set the contract named.

## B1 — the version census

`cmd/savemigrate` (`cmd/savemigrate/main.go`) is one program with two modes, the `cmd/saverepair`
precedent. Given no `-w`, or given `-w` against a file already at the current version, it reports and
writes nothing. Given a file or a directory argument, it prints for each `.ags` file it reads:

- the envelope version, read by `game.EnvelopeVersion` (`pkg/game/save.go`), a new exported function
  that reads the one byte `DecodeSave`'s own header check refuses on, so a tool can report the number
  a mismatch would otherwise turn into a bare error;
- the byte-form version, the first byte of the snapshot's `World` field;
- whether this build loads it (`sim.UpgradeSaveForm` succeeds or names why not);
- what upgrading it would lose, one line per substituted section, in `UpgradeSaveForm`'s own returned
  sentences.

A directory argument is walked recursively for every `*.ags` file beneath it
(`cmd/savemigrate/main.go`'s `collect`). Given more than one path, the run prints a per-version count
and a total across every file it read (`census`), grouped by `"byte-form N"` or `"town, no world
half"` for an envelope-only save. A file this tool cannot read at all — a bad envelope, a header
shorter than the fixed prefix — is reported as an error and excluded from both the write path and the
census; it is not treated the same as a version this build cannot upgrade, which is reported and
counted.

Measured against a copy of the owner's own 65 files (`saves/` and `saves/rescued-20260822/`) in a
scratch directory, `cmd/savemigrate` reports: byte-form 41 (1 file), 45 (2), 46 (5), 53 (23), 55 (28),
56 (1), and 5 town saves with no world half — 60 world-half saves plus 5 town saves, total 65. The 8
files at versions 41, 45 and 46 predate `oldestReadableVersion` (50) and are reported as `will not
load`; the other 52 world-half saves upgrade and load.

## B2 — the reader accepts byte-form versions 50 through 57

`UnmarshalBinary` itself is unchanged: it still accepts `formatVersion` (57) alone and refuses
everything else, at the same site and with the same message. The wider range sits in front of it:
`sim.UpgradeSaveForm` (`pkg/sim/upgrade.go`) widens a form at any version from
`oldestReadableVersion` (50) through `formatVersion` into `formatVersion`'s own shape before handing
it to `UnmarshalBinary`.

`UpgradeSaveForm` refuses every other version in three arms, and the refusal is written for the
player: it reaches him unedited on the load window's message line, which holds 104 columns.
`resumeWorld` returns it unwrapped for that reason, while a byte form that will not decode keeps the
prefix naming which half of the save failed. The three sentences are:

    this save is too old to open: format 46. This build reads 50 to 57
    this save is newer than this build: format 99. This build reads 50 to 57
    this save names format 52, which no version of this game ever wrote

`pkg/ui`'s `loadFailure` puts the name of the row the player chose in front, and `clipRunes` marks a
cut with an ellipsis where one is made. The longest of the three, with the longest save name in the
owner's own corpus and a three-digit format number, is 101 runes.

Versions 51 and 52 take the third arm: `binary.go`'s own header records both as allocated and
returned unused, and no build of this tree ever wrote either, so `UpgradeSaveForm` treats them as
unknown rather than guessing at an unshipped shape.

**The one declared per-version table** is `upgradeSteps`, a `map[byte]upgradeStep` keyed by the source
version, in `pkg/sim/upgrade.go`. Each entry names: the source version's entity record width, its
spell record width, whether its casting section needs the pre-53 translation, whether it carries an
item-weight section at all, its script check-record width, and the disclosure sentences B3 owes for
what it lacks. `widenSaveForm` reads only this table; no version number is tested a second time
anywhere else in the reader. `UpgradeSaveForm` itself is the only entry point, called identically by
`resumeWorld` (`pkg/game/resume.go`) and `cmd/savemigrate` (B4).

**The substituted value for every absent section is the honest empty or zero default**, never an
invented number:

- entity fields added after the source version (protection, invisibility-sight, reaction, spirit,
  corpse-loot suppression, delayed kill-credit, carried load and capacity, the authored map placement
  id) widen with zero bytes appended to each record, the same shape `MarshalBinary` never writes a
  nonzero value into for a fresh entity;
- the item-weight section, absent before version 56, widens to a declared zero count;
- pre-53 area spell effects widen with no caster, no power, no strike shape, and their one original
  cell as their only cell — the same shape the pre-53 format wrote for a plain area burn, not an
  invention;
- the spell table's grown fields widen with zero bytes appended per record, the same as the entity
  case.

`Load` and `Capacity` are not recomputed by the widening and are not left at a sentinel. Both are
plain zeros in the widened form, and `pkg/sim/weight.go`'s `recomputeLoad` reads the item-weight
table, which the widening also writes empty: with no weights declared, `loadOf` returns zero for
every actor and a recompute cannot produce anything else. `Load` is restored at the load, from the
started mission's own weight table, by the repair described below. `Capacity` is not restored, and
its sentence stands.

**The one section that cannot be widened by appending a tail** is the casting section below version
53: the pre-53 area-effect record (6 bytes: cell key, spell id, remaining lifetime) has no tag byte
and no book or attached-cast records, where the current shape prefixes every casting-state record
with a one-byte tag (`castingAreaTag`, `castingAttachTag`, `castingBookTag`). `translateLegacyCasting`
(`pkg/sim/upgrade.go`) re-emits each pre-53 area effect as a tagged area record with every field
version 53 added at its empty default, preserving the original order (script order for casts, cell
key then arrival order for effects — `castbinary.go`'s own documented order, which `decodeCasting`
still requires). Pending casts are unchanged in shape at every version this story reads and are
copied whole.

Every section between the routes and the spell table — group, sack, carry, equipment, death-gold and
purse — is unchanged in shape at every version 50 through 57 and is copied whole, located by walking
its own declared count exactly as `decodeRoutes` and the other section decoders already do. The
script-state section (`tailbinary.go`, relation formation plus cell tails) is unchanged in shape since
it was added at version 50 and is likewise copied whole. The relation is the form's fixed tail and is
copied last, with a length check that the source form has no trailing bytes left over.

## B3 — the loss disclosure

`UpgradeSaveForm` returns one `sim.SaveFormLoss` per substituted section: the section's own
`sim.SaveFormSection` value and a sentence written for the player rather than for the format. The
list is derived entirely from `upgradeSteps`; nothing hand-maintains a second copy of it. Two
accessors read the table without a form to widen: `sim.SaveFormLossesAt(v)` returns the losses a
version-`v` form would carry, and `sim.SaveFormReadableVersions()` returns the table's keys in
ascending order. Both exist so that a witness above `pkg/sim` can measure the disclosure of every
version.

Each sentence is in the reader's units and names the version that added the section. The two the
item-weight table costs are:

    no item weights and so no carried load for any character (added at version 56): every carried
    and worn item weighs nothing

    no carry capacity for any character (added at version 56): nobody is slowed by an overloaded
    pack for the rest of this session

Neither promises a later recompute. An earlier draft of the first sentence said both values were
recomputed the next time an item is picked up, dropped, worn or stored, which is measurably false:
`Entity.Load` is written only from the item-weight table, and the widened form declares that table
empty.

`resumeWorld` (`pkg/game/resume.go`) forwards the sentences of the sections it could not repair, and
only those. `frontend.go`'s mission-open closure collects them and, after residue and fog are applied,
calls `mw.openLoadNotice(view.DialoguePages(loadNoticeParagraphs(loadNotices)))` when the list is
non-empty.

**The disclosure is paged, and every page is drawn whole.** `loadNoticeParagraphs`
(`pkg/game/resume.go`) returns the header and one paragraph per sentence, as separate strings. It no
longer joins them: `ui.noticeBreak` treats a newline as a wrap opportunity rather than a hard break,
the atlas has no record below 0x20, and every such byte draws as a space, so the bullet layout the
first draft composed never reached the screen and left literal newline bytes inside drawn lines.
Measured on the real EN font at the shipped dialogue geometry, a version-53 save's disclosure needed
16 wrapped lines where `ui.NoticeLayoutOf` draws 8, and the other 8 were unreachable.

`ui.NoticeMaxLines(l, f)` states `NoticeLayoutOf`'s own clamp: how many wrapped lines that layout's
text area draws. `ui.NoticePagesOf(l, f, paras)` packs paragraphs into the fewest pages of at most
that many lines. Paragraphs are packed in order and joined with a single space; a paragraph too long
for one page on its own is split at its own wrapped line boundaries. Nothing is dropped and nothing
is reordered. `ui.Viewer.DialoguePages` applies this to the viewer's own dialogue layout resolved
without a portrait, which is the shape `openLoadNotice` pushes.

`mapWorld.openLoadNotice` (`pkg/game/world.go`) takes the page list and reuses the existing notice
window: it sets the same `missionNotices` fields `showOutcome` and `openDialogue` already set and
calls `ui.Viewer.SetDialogue`, the same call a mission-script dialogue makes. It carries no event
payload. `advanceNotice` shows the next page on each dismiss and closes the window after the last
one, so a disclosure longer than one page is reachable in full by the same gesture a multi-part
mission dialogue already teaches.

`ui.Viewer.DialogueClip(s)` reports how many wrapped lines `s` produces and how many of those the
dialogue window draws. The two numbers are equal for text that fits and differ for text that does
not. Nothing else in this tree reports the second one: `NoticeState` hands back the raw string that
was pushed, which is complete whatever the window drew.

`FrontEnd.LiveNotice` and `FrontEnd.LiveNoticePages` (`pkg/game/resume.go`) read the open map
screen's notice back for a developer tool with no window. `cmd/savecheck load` prints the page on
screen, the page count, and for each page its produced and drawn line counts.

A save whose byte form is already at `formatVersion` produces no losses and opens no notice.

## B4 — upgrade is one road

`sim.UpgradeSaveForm` is the one function between an older form and the current one. `resumeWorld`
(`pkg/game/resume.go`) and `cmd/savemigrate`'s `one` function (`cmd/savemigrate/main.go`) both call
it and nothing else widens a byte form anywhere else in the tree. Neither reimplements any part of
the widening; `cmd/savemigrate` additionally round-trips the result through `sim.World.UnmarshalBinary`
before writing, the same verification `cmd/saverepair` already performs before its own `-w` write.

## The load-time repairs

Four of the ten substituted sections are re-declared at the load, from the mission the save is
resumed into, before the disclosure is composed. The data is taken from `ms.World` before
`UnmarshalBinary` replaces it: `StartMissionScripted` has already resolved the item-weight table off
the definition tables, compiled the script off the map file, resolved the spell table off the
`Spells` collection, and constructed the map's own structure list off the class table, and all four
are overwritten the moment the save's own bytes land.

| Section | Restored by | Source |
|---|---|---|
| `SaveFormItemWeights` | `sim.World.DeclareItemWeights` | the mission's own item-weight table |
| `SaveFormCheckItemRef` | `sim.World.RestoreCheckItemRefs` | the compiled script's own checks |
| `SaveFormSpellRuleTail` | `sim.World.RestoreSpellRuleTail` | the mission's own spell table |
| `SaveFormStructures` | `sim.World.DeclareStructures` | the mission's own structure list |

`sim.RepairableFromMission(s)` is the predicate naming those four, and it is the one declaration of
which sections a mission can supply. `DeclareItemWeights` recomputes every actor's `Load` against the
restored table, which is the whole of what that section costs a save: `Entity.Load` has one writer
and it reads this table.

Each of the first three repairs withdraws its own sentence and no other, and each refuses rather than
guess when the started mission's records do not correspond to the save's: a repair that refuses
leaves its sentence standing, so a load that could not put a section back still says so.

`DeclareStructures` is different: it never refuses. No version below 58 (the version this story's own
57-to-58 rung reads from) carries any structure bytes, so there is nothing on the save's side to
compare the mission's fresh construction against, and the repair always succeeds. Its own disclosure
sentence, `lostStructures`, is declared for `cmd/savemigrate`'s report — which opens no mission and so
cannot repair the section itself — and is withdrawn every time a mission does the repair. No on-screen
notice for this section reaches a player at load: the decision follows the item-weight table's own
precedent, where a repair that always succeeds carries no disclosure the game can ever show. See "The
57-to-58 rung" below for why the substitution is safe.

## The 57-to-58 rung

`upgradeSteps`' version-57 table entry (`pkg/sim/upgrade.go`) is the only rung this story owes the
`1033` merge: a version-57 form is otherwise `formatVersion`'s own predecessor with no table entry,
which takes the third refusal arm and will not open. It repairs `SaveFormStructures` from the started
mission rather than leaving the section zeroed, on the same (a) identity / (b) immutability criteria
`upgraderestore.go`'s own header states for the other three repairs (`DIV-254`, F-1's resolution).

Criterion (a) is vacuous here in the same way it is for the item-weight table: a widened pre-58 form's
own structure count is always zero, so there is no source-side record to compare the mission's
construction against, and nothing can be refused. Criterion (b) holds for a narrower and stronger
reason specific to this one rung: every version-57 form was written by a build of this tree that had
no structure section and no instant capable of changing a structure's field, because that concept did
not exist until story `1033`. A save at version 57 therefore cannot carry a play session that had
already changed a structure's state, and the mission's own fresh construction is the value a native
version-58 save of that same session would have carried.

**This argument does not generalise to a future 58-to-59 rung**, and the table entry's own comment
says so: any later widening of a save that DID carry a structure section once the concept existed
faces the same identity problem criterion (a) fails for the six per-entity fields below — an id match
is not a proof, and a fresh mission's construction is not the value a resumed session actually held.
Zeroing is not the fallback either: a value the running session held and the save format could not yet
record is exactly the case the other six unrepaired sections are already in, and their sentences stand
rather than being invented.

The other six sections are not repaired and their sentences stand. Five are per-entity fields
(`SaveFormDefenceState`, `SaveFormCorpseLoot`, `SaveFormKillCredit`, `SaveFormCarryCapacity`,
`SaveFormMapUnitID`); the sixth is the pre-53 area-effect shape. An entity field can only be matched
between the save's entity list and the freshly built mission's by entity id, and the two lists are not
the same population: a save taken after a Control Spirit raise carries entities the fresh mission
never allocates, and the fresh mission's own ids continue from `lastID + 1`. Measured on the owner's
mission-151 save, the resumed world holds 235 entities against 238 in the fresh construction, so an
id-matched restore would write one entity's state onto another. `DIV-275` carries that reasoning for
`Capacity`.

## B5 — `cmd/savemigrate`

Reports by default. `-w` rewrites in place, keeping the original beside it as `<path>.bak`, the same
as `cmd/saverepair`. It refuses to touch a file it cannot read at all — a bad envelope, a truncated
header — leaving the file and no `.bak`, with exit code 1. A file already at the current version is
reported and left alone whether or not `-w` is given.

`-w` also refuses a file whose repairable sections hold data. Four of the ten sections are restored
at load time from the started mission, and none of them can be restored by this tool, which opens no
mission. Rewriting such a file to the current version removes the older source version declaration,
so the substituted zeros become the file's own permanent contents and the load path can no longer
tell that anything was substituted. The refusal is on content, not on version: `holdsSectionData`
(`cmd/savemigrate/main.go`) decodes the widened world and asks whether it carries a compiled check, a
spell rule, or any carried or worn item.

The structure section is the exception, and it answers unconditionally rather than by content.
`holdsSectionData` cannot ask whether the target MISSION has structures: a structure is per-map state
seeded from the class table at mission start, this tool opens no map, and a widened pre-58 world's own
structure count is always zero regardless of what the mission actually has, so counting it answers
nothing. It treats that unknown as data it must not discard, on the same "refuse rather than guess"
rule the other three repairs follow. Every version this table reads (50 through 57) loses the
structure section, so `-w` cannot currently rewrite any pre-58 file in place; report mode,
`cmd/saverepair`, and `pkg/game`'s own load-time repair are unaffected. A file whose repairable
sections are otherwise all empty is still refused for this one reason alone (1032 return 2, found
while sweeping `RepairableFromMission`'s callers for the section this return adds to that set).

## What is deliberately not in this story

Unchanged from `contract.md`: the original game's own `.sav` format (`DIV-026`); making a save this
build writes loadable by the original game; the envelope version (still read as 1 only); versions
below 50; a new byte-form value of `formatVersion` (57 is unchanged by this story — every version
this story reads is strictly older than the version it writes).

Story `1033` bumps `formatVersion` to 58 on its own branch and inserts a structure section between
the script-state section and the script section. That branch is not merged at this landing and this
story's table stops at 57. The merge of the two owes `upgradeSteps` a 57 rung: without one, a form
at version 57 is no longer `formatVersion` and has no table entry, so it takes the third refusal arm
and will not open.

## G2 — the limit this implies

The per-version table (`upgradeSteps`, `pkg/sim/upgrade.go`) is the seam. A later version becomes one
new map entry naming its own predecessor's widths and its own disclosure sentences, not a new branch
in the reader. The same table is what B1's report and B3's disclosure both read. Reading an old save
cannot change a shipped file's bytes: the only files this story's `-w` mode writes are the ones its
own arguments name, and the owner's own saves are never named by a command this lane ran (verified —
see `closure.md`).

## Decisions the contract left open

- **One program, two modes.** `cmd/savemigrate` is `cmd/saverepair`'s own shape: report by default,
  `-w` to rewrite. A directory argument additionally makes it the census tool B1 asks for; there is no
  second binary.
- **Substituted values.** Zero or empty in every case, per the table above; the one case that is not a
  literal append (the casting section below version 53) still ends at every added field's own empty
  default.
- **How B3 reaches the screen.** The existing dialogue notice window, via a new
  `mapWorld.openLoadNotice` method that composes the same fields `showOutcome` and `openDialogue`
  already do, with no event payload of its own. The disclosure is paged rather than scrolled: the
  window has no scrollbar and the original's own control scrolls, so paging through the window's
  existing dismiss is what shows a notice longer than eight lines.
