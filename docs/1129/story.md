# Story 1129 — a hired squad persists as its own actor record in the native town SAV

## Status

On the exact candidate commit: `gofmt -l .` clean; `go test -trimpath -count=1
./...` (whole repository) clean, including `internal/gatedtests`' checked-in
`testdata/population.txt` census (net +1: three renamed/new release witnesses
added, two stale refusal-witness names removed); `scripts/check-no-game-assets.sh`
clean; `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed
at this worktree) against both lawful roots — counts and per-root result in
"Proof" below.

**Correction pass** (`pipeline/reviews/story1129-pass1.md`, verdict RETURN,
findings F-1..F-4): all four corrected. `gofmt -l .` clean; `go test
-trimpath -count=1 ./...` clean, `testdata/population.txt` net +3 (190
total: the three F-3/F-1 witnesses named below); `scripts/check-no-game-assets.sh`
clean; `pipeline/check-release-tests.sh` against both roots: 8 packages, 190
gated tests, 190 of 190 ran and 0 lacked a subject, on EN and again on RU.

## Player result

A player who hires a squad in the tavern and saves in town (native SAV) now
gets a file whose hired mercenaries are stored the way ROM1 stores a
confirmed hire: each as his own actor record, in a second group under the
Player, beside the campaign record's own hire-flag dword. Loading that file
back in a fresh process restores the same mercenaries — identity, worn
equipment, carried items, spellbook, health, derived combat block
(damage, to-hit, defence, protection, speed, sight, rotation speed) and
party order — from the persisted record itself, not from a freshly rebuilt
tavern template. Journal, current mission, mercenary status and money read
back unchanged, on EN and RU. A siege engine (Catapult/Ballista) and a file
written before this story are unaffected: both still take the pre-existing
rebuild-from-count fallback. A squad hired alongside a siege engine round-trips
natively in the same order master itself supported (siege before Human); the
opposite order still falls back to the lossless `.ags` envelope, as it did on
master.

## Authority

Research EXP-0309 (research master `c0b24f1`, pinned this story) supplies
four promoted claims:

- **SAV-614** (High): a confirmed hire debits the money field and flips the
  campaign record's own per-type hire-flag dword.
- **SAV-616** (High): the confirmed hire's own new actors reproduce classKey
  58's already-published corpus signature field for field — class Human,
  typeWord `0x000a`, six worn slots, exactly three per file in this witness,
  health 120 at hire — and retracts SAV-608's "not evidence of hired
  mercenaries" reading of that signature for classKey 58 specifically
  (`research/claims/retracted.md`). classKey 54 is untouched by this
  retraction; SAV-608's own reading stands for it.
- **SAV-617** (High): the hire creates exactly one new group under the
  Player, holding exactly the new actors; the Player's own record and group 1
  are unchanged. Address-level cross-check against MERC-HIRE-003/
  MERC-LEVEL-005 (spawn) and PARTY-ROSTER-002 (container structure) finds no
  gap.
- Both graded High for the narrow clause each refutes or confirms, not for
  every classKey-58 file in either corpus: SAV-616 states outright that the
  pre-existing 55-file corpus's own 14 classKey-58 files are not retroactively
  certified as hires by this witness (no before/after pair exists for them),
  and neither claim reads a Name field, so a hired mercenary's own recognizable
  name (`NPC%02d_%d`) is this project's own convention, not read from either
  claim — see "Open debt."

Owner direction: reuse the existing actor-record writer (story 1109/1115 item
paths, story 1096 spellbooks); do not add a second record encoder.

## As-built behaviour

**1. Write side: `nativeCityData` (`pkg/game/nativecity.go`) walks roster and
hired members through the identical per-member construction once.** `combined
:= append(roster, hired...)` is one slice, one loop, one call each to
`nativeCityUnitData` and `nativeCityAttachItems` per member — the same
producers group 1 (the ordinary roster) already used before this story, so a
hired member's Human object, worn/carried items and known spells are built by
code this story does not modify. Only the post-loop actor-index split is new:
`actors[:len(roster)]` stays group 1's own `Actors` list (unchanged);
`actors[len(roster):]` becomes a second `sav.CityGroupData` (`Raw80: make
([]byte, 80), F44: nativeCityPlayerIdentity`, the same owner convention group
1 uses), appended only `if len(hired) > 0`. `nativeCityHumanHiredMembers`
(new) is the write-side filter: every party member with a nonzero
`MercenaryType` that is not `nativeCitySiegeMember` (tavern type 1 or 2,
Catapult/Ballista — a different wire class, `Unit(0x198)`, that SAV-616's
retraction does not cover), preserving live relative order — no rebuild, so
no reordering rule needed for this group.

**2. `nativeCityDefRow` resolves a hired member's installed Humans-table row
by name, not by the many-to-one TypeID fallback.** `data.FindHumanByName
(table.Humans, member.Name)` is the identical row `tavern.go`'s
`buildMercenarySquad` itself found by name at hire time
(`fmt.Sprintf("NPC%02d_%d", typ, level)`). The pre-existing `data.FindHumanByType`
fallback is many-to-one over the Humans collection (its own doc comment
already said so, for the hero/companion case) and would have landed a hired
member on a sibling row with the same class art but a different Face,
HiredRotationSpeed or Book — dormant before this story, since a hired
mercenary never reached this writer at all until now.

**3. Read side: `mercenaryHireTypeFromName` (`pkg/game/originalparty.go`)
recognizes a persisted hire by his own Name field, reusing the exact template
string `buildMercenarySquad` writes rather than re-deriving it.** Neither
SAV-616 nor SAV-617 reads a Name field — this recognition is this project's
own read-side convention, disclosed as such in the function's own doc
comment (see "Open debt"). `restoredMember` sets `MercenaryType` from it and,
for a recognized hire, skips `data.HeroAppearance`'s cloaked-body composition
(the PLAYER-CHARACTER appearance rule): Class comes from the row's own raw
`TypeID` with no composed Body/BodyDir, mirroring `buildMercenarySquad`'s own
design (owner, 2026-08-25: a mercenary is never turned into a hero with a
composed body).

**4. `restoreHiredMercenaries` (`pkg/game/nativecityrestore.go`) is now a
per-type fallback, not the unconditional rebuild.** `present[typ]` — true
once `RestoreParty` has already installed a member of that type in
`f.Carried` from his own persisted record — skips the rebuild for that type.
It still zeroes the pool cell for every hired type (the `mercenaryHire`
invariant, so a later in-session return does not double-count), and it still
fully rebuilds a type with no persisted member: a siege engine (never
individually addressable, any save), or any type in a file written before
this story (a set hire flag with no persisted record at all — a save that
old never set the flag in the first place, since the writer refused outright
whenever one was hired, so this is also the deterministic default for a
still-older save).

**5. `nativeCityHiredEquipmentMismatch` and the renamed
`nativeCitySiegeHireOrderMismatch`** (was `nativeCityHireOrderMismatch`,
DIV-907) **narrow to the siege-only remainder, in both writers.**
`ExportNativeCitySave` (`nativecity.go`) and `marshal` (`originalsave.go`,
the second-and-later-SAVE writer, DIV-910) share the exact two functions;
both now iterate only `nativeCitySiegeMember`. A Human-type hire's own live
equipment, inventory, spellbook and party order all round-trip through his
own persisted record, so neither guard needs to score him against a rebuilt
template first — the mismatch he would have tripped no longer exists to
trip. In `marshal` specifically, a Human-type hire is now one of
`state.bindings` like any other restored character (`originalsave.go`'s own
call-site comment); `bindCityHuman`'s pre-existing `member.Hired()` guard
(`pkg/game/originalhuman.go`) still keeps his `OriginalHuman` basis nil
regardless, so the sale/training replay machinery a genuine imported city
needs never activates for him — the ordinary per-binding `reflect.DeepEqual`
mismatch arm refuses a changed hired member's second SAVE instead.

## Correction pass (`pipeline/reviews/story1129-pass1.md`, verdict RETURN)

**F-1 — a reloaded hire's combat block.** `restoredCarry`
(`pkg/game/originalparty.go`) populated `Carry.LiveLoad` for every restored
character unconditionally, including a hired mercenary whose save format
carries no raw Attack/Defence/Modifier bytes at all. `originalHumanSpawn`
reads `Carry.LiveLoad.Inventory.Source.Class == 2` as its first branch, so a
reloaded hire derived his mission entity from those absent bytes instead of
falling through to the generic recompute a fresh hire always uses — an
all-zero combat block, and a RotationSpeed that fell through
`RotationSpeedBase`'s own legacy TypeID lookup instead of reading the row's
own column. Fixed two ways: `restoredCarry` now gates `Carry.LiveLoad` on
`!hired`, so a reloaded hire takes the same generic-recompute path a fresh
hire does; `restoredDefinition` now also returns the row's own
`RotationSpeed` column, and `restoredMember` sets `HiredRotationSpeed` from
it whenever `hired` is true (this also closes F-4's `HiredRotationSpeed`
and `Temporary` regressions below, both set from the same corrected
literal).

Closing the all-zero block exposed a second, previously latent gap: the
generic recompute reads each worn item's `Effects` list
(`mapload.ApplyItemEffects`, Kind 12/17/19/21..25 = ToHit/Speed/ScanRange/
Protection), and `nativeCityAttachItems`'s own Item/Weapon/Armor/Shield
writer (`pkg/game/nativecityitems.go`) has no byte position for that list at
all — it writes `SourceEquipment` (Attack/Defence/OwnKind) and nothing else,
by construction, so a round-tripped worn item always decoded with
`Effects == nil` regardless of what the live item carried. This was silently
correct only while the disabled Class-2 branch (which never reads `Effects`)
answered for every hire. Adding a wire position for it would be
unresearched byte-format work this correction does not attempt (B1);
instead, `restoredHireWornEffects` (new, `originalparty.go`) backfills a
worn slot's `Effects` by re-resolving the hire's own Humans row
(`mapload.HumanRowEquipment`, the identical resolution a live hire's own
equip already runs) whenever that slot's restored Code still equals the
row's own template Code for it — true at the moment of hire by construction.
A slot a shop visit changed since hire keeps its file-decoded, Effects-less
item exactly as it did before this correction (DIV-889's existing
changed-equipment scope, not narrowed here).
`TestReleaseNativeTownSaveHiredMercenaryCombatBlockRoundTrips` (new,
`nativecity_release_test.go`, EN and RU) proves a solo type-10 hire's full
derived combat block (`mapload.PartyDisplayWithTable`) equals a freshly
hired squad's own after one native SAVE and reload; type 10 was chosen
because the review's own census found two other tavern types (6, 14) already
agreed by coincidence pre-fix.

**F-2 — the wrong typeWord for a hired actor.** `nativeCityToken` wrote the
roster/hero marker `0x21` (`nativeCityHeroTypeWord`) for every actor,
hired or not. `nativeCityHumanTypeWord` (new, `nativecity.go`) returns
`uint16(member.Class)` for a hired member instead — the Humans row's own raw
TypeID, already carried on `member.Class` since `restoredDefinition`/
`def.TypeID` (item 3 above; `tavern.go`'s own `buildMercenarySquad` sets the
same value at hire time). SAV-616 (a confirmed tavern-type-14 hire, typeWord
`0x000a`) and SAV-627/SAV-628 (a directly-authored, non-hire placement,
classKey 54, typeWord `0x0003`) independently agree the marker equals the
placed actor's own row TypeID; cross-checked empirically against the
installed EN table through this project's own `data.NewHumanDef` decoder
(NPC14_1 row 58 decodes TypeID 10, NPC10_1 row 54 decodes TypeID 3 — both
match). Confirmed at exactly these two of thirteen Human tavern types; the
rest is this project's own generalization, disclosed as DIV-918 (new).
Roster (hero/companion) keeps `nativeCityHeroTypeWord` unchanged.
`TestReleaseNativeTownSaveHiredMercenaryEmitsNativeSAVAndRoundTrips`
(extended) reads each hired identity's own decoded typeWord back through
`(*CityProvenance).Human` and asserts it equals `uint16(member.Class)`,
keyed by identity so three same-named hires are checked as three distinct
actors.

**F-3 — a siege-then-Human hire order refused where master succeeded.**
The first landed candidate's `nativeCitySiegeHireOrderMismatch` scored any
non-siege member appearing after a siege member, not only a Human hire — a
stricter guard than master ever enforced, since master's own ascending-type
rebuild put every siege type first by construction (siege types 1/2 always
sort below every Human type 3..15). Hiring a siege engine and then a Human
type in one visit was native on master and refused-to-`.ags` on the
candidate; the reverse order was `.ags`-only on master and, by the same
over-broad guard, accidentally native on the candidate — capabilities
exactly swapped. Fixed to reproduce master's own asymmetry, not either
side's accident: the guard (renamed to keep, `nativeCitySiegeHireOrderMismatch`)
now scores only a siege member against an already-seen Human hire or an
out-of-order siege type, and `restoreHiredMercenaries`
(`nativecityrestore.go`) now inserts a rebuilt siege engine at the position
of the first already-restored Human hire (`insertAt`), not the party's tail,
so a Human hire already occupying that position is never rebuilt behind him.

Fixing the insertion point exposed a stable-ID staleness bug testing found:
`mapload.NameParty` mints a hired member's ID from his raw array position
and never renames an already-ID'd member, so a Human hire's own ID (minted
by `RestoreParty`'s earlier `OwnParty` call, before any siege member existed
to shift him) went stale once a later siege insertion moved his position.
`restoreHiredMercenaries` now clears the shifted tail's own ID before
`OwnParty` runs again, so `NameParty` re-derives it against the array's true
final shape — the same ID a fresh hire in that visit order would carry.
`TestReleaseNativeTownSaveHiredMercenarySiegeThenHumanOrderRoundTrips` and
`TestReleaseNativeTownSaveHiredMercenaryHumanThenSiegeFallsBackToAGSOrderPreserved`
(new, EN and RU) prove both directions match master's own asymmetry, order
and stable ID intact either way.

**F-4 — `Temporary`/`HiredRotationSpeed` lost, and a stripped weapon
disappears.** `Temporary` and `HiredRotationSpeed` are both fixed by F-1's
`restoredMember` literal above (`Temporary: hired`, `HiredRotationSpeed:
hiredRotationSpeed`) — confirmed via the reviewer's own held two-cycle chain
reproduction (`tmp/review-1129/held/rev1129chain_test.go`, reused as a
witness, not committed): every member's `Temp`/`HiredRot` now matches
across the reload where it did not before. The remaining finding — a
stripped mercenary's `Weapon` goes from `{Iron Mace...}` to `<nil>` across a
native SAVE/LOAD where master's `.ags` fallback preserved it — is not
code-fixed, per the review's own "Honest direction": the live value is
itself already stale (`Worn[0]` is already 0 while `Weapon` is still set,
because the shop strip route never clears it), so the reloaded `<nil>` is
arguably the more correct value, not a loss this correction should paper
over with a second, independent staleness. Left as disclosed, accepted debt
(below). A weapon's own display-name text (`Iron Mace` -> `Common Iron
Mace`) reproduces identically on the hero's own weapon on master's native
route too — pre-existing, unrelated to a hire specifically, out of this
correction's scope.

## Divergence rows

Amended (`docs/DIVERGENCES.md`), each with a STORY 1129 paragraph so no live
row leans on SAV-608's retracted classKey-58 clause without saying so:

- **DIV-887** (generated character DefRow): records the name-first branch
  (item 2 above) and narrows the row's own revisit condition — the TypeID
  collision gap it names is live now only for a hired name the table no
  longer carries.
- **DIV-889** (equipment/inventory/spellbook contents): records that its own
  equipment-mismatch refusal is now siege-only; a Human-type hire's
  equipment round-trips through his own record with no refusal and no `.ags`
  fallback for that shape.
- **DIV-893** (Item fields block byte positions): confirms the same writer,
  unchanged, now also runs for a hired member's own record; its own open
  question (the four columns' true ROM1 byte positions) is unaffected.
- **DIV-902** (MercenaryWorking cell reuse at SAVE time): records that the
  cell is still written the same way for every type but is load-bearing on
  LOAD only for a siege type now — a Human type's own value still
  round-trips but is no longer consulted as a headcount (`present[typ]`
  skips it).
- **DIV-907** (hired mercenary restore order and stable ID): records that
  this row's own gap is closed for a Human-type hire (his own record
  preserves true live order with no rebuild) and stays live only for the
  renamed siege-only function.
- **DIV-910** (second-SAVE writer after a native LOAD): records that the
  shared guards named in this row moved with their native-writer
  counterparts (item 5 above); the row's own subject — one writer for a
  native campaign's second SAVE — is unchanged.
- **DIV-884, DIV-903, DIV-904** (Markers/FirstMapPoint, Markers Field0/
  Field1, selectedMarkers policy): confirmed unrelated — each is a
  campaign-record field this story's own actor-record group does not touch —
  and noted as such rather than left silent.

**Correction pass** (`pipeline/reviews/story1129-pass1.md`) further amends,
each with a STORY 1129 CORRECTION paragraph:

- **DIV-907** (hired mercenary restore order and stable ID): records F-3 —
  the first landed candidate's own siege-order guard and rebuild-insertion
  point over-corrected past master's own asymmetry; both are now fixed to
  reproduce it exactly (siege-before-Human native, Human-before-siege
  `.ags`), and the stable-ID staleness bug the fix exposed is closed
  alongside it.
- **DIV-889** (equipment/inventory/spellbook contents): records the F-1
  residual — `nativeCityAttachItems`'s own item writer has no wire position
  for a worn item's ordered `Effects` list at all, closed on the read side
  (`restoredHireWornEffects`) for an unchanged-since-hire slot only; a
  changed slot's reduced fidelity is this row's own pre-existing,
  unnarrowed scope.

**DIV-918** (new, of the reserved DIV-918 through DIV-925 range,
`pipeline/ALLOCATIONS.md`) records F-2: the hired-actor typeWord rule
(`uint16(member.Class)`, the Humans row's own TypeID), confirmed at exactly
two of thirteen Human tavern types from two independent origins (SAV-616, a
confirmed hire; SAV-627/628, a directly-authored non-hire placement), and
this project's own generalization, disclosed rather than assumed, to the
other eleven. **DIV-919 through DIV-925 remain unused and retired**: no
other fact either the first landing or this correction pass needed was a
new, unclaimed owner-direction choice of its own.

## Proof

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...`: all packages pass. `pkg/game` alone:
  123s (EN assets) / 126s (RU assets), both clean.
- `scripts/check-no-game-assets.sh`: clean.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 187 gated
  tests, 2 roots; **187 of 187 ran and 0 lacked a subject, on EN and again on
  RU.**
- **Field-by-field release witnesses**
  (`pkg/game/nativecity_release_test.go`), EN and RU:
  - `TestReleaseNativeTownSaveHiredMercenaryEmitsNativeSAVAndRoundTrips`:
    hires a squad, SAVEs, reloads in a fresh `FrontEnd`. Compares each
    restored hired member against the live pre-save member — Name, Class,
    Profile, FigureDir, FigureFace, Hero, worn item codes, carried item
    codes, KnownSpells — the same fields the pre-existing hero/companion
    round-trip test compares, not a rebuilt-template comparison.
  - `TestReleaseNativeTownSaveHiredMercenaryChangedEquipmentRoundTrips`
    (renamed from `...ChangedEquipmentRefusesAndAGSFallbackRoundTrips`, which
    proved a refusal before this story): hires, strips and re-equips
    different gear, sells an item, SAVEs, reloads. Now proves the native
    round trip — worn, carried, KnownSpells and derived health
    (`mapload.PartyDisplayWithTable`) all equal — instead of the refusal.
  - `TestReleaseNativeTownSaveHiredMercenaryAscendingOrderRoundTrips` and
    `TestReleaseNativeTownSaveHiredMercenaryDescendingOrderRoundTrips`
    (renamed from `...DescendingOrderRefusesAndAGSFallbackRoundTrips`): hires
    two types in ascending and in descending order; both now round-trip
    natively with true live order and stable ID preserved, where the
    descending case previously refused and fell back to `.ags`.
  - `TestReleaseNativeTownSaveHiredMercenaryFlagOnlyFileStillLoads`: sets a
    campaign record's hire flag and pool count directly, with no persisted
    actor record (the pre-story-1127 and pre-story-1129 shape), and confirms
    `restoreHiredMercenaries`'s own fallback still rebuilds the correct
    count from the tavern template, the starting hero untouched, and the
    `mercenaryHire` pool-cell invariant holding afterward.
  - `TestReleaseNativeTownSaveSecondSaveAfterReloadStaysNativeWithHiredSquad`
    (story 1128's own chain, unmodified by this story other than its
    comment): hire, SAVE, reload, SAVE again with nothing further changed.
    Both SAVEs native; roster length and `mercHired` equal at both
    checkpoints — proves the hire/save/reload/save chain stays native with
    the new record shape.
  - `TestReleaseNativeTownSaveSecondSaveAfterReloadHiredEquipmentStrippedFallsBackToAGS`:
    a reloaded Human-type hired member — now Carry-backed by his own
    persisted record for the first time, taking the source-actor shop path
    (`mapload.HasSourceActor`) instead of the plain code-splice path a
    never-yet-saved hire takes — has every worn slot stripped and sold
    through production `shopUnequipToTable`/`shopSell`. Still falls back to
    `.ags` on the next SAVE and still reads back the stripped `Worn` array
    and post-sale purse exactly, but the mechanism moved: `marshal`'s own
    generic per-binding `reflect.DeepEqual` mismatch now catches the changed
    equipment (item 5 above), not the old mercenary-specific guard, which no
    longer runs for a Human-type hire at all. The strip loop itself needed a
    fix this story exposed: unequipping a weapon auto-displaces a worn
    shield into the pack (the pre-existing DIV-445 owner policy,
    `pkg/sim/sourceequipmove.go`) on the source-actor path only, updating
    `Carry.Equipped` immediately but not the outer `Worn` snapshot the loop
    used to re-check — the fix re-checks the live `shopWornItemSlots` view
    instead, unrelated to production `sim`/shop code, which is unmodified.
- **A story-1127 file with flags only still loads**: proven by
  `TestReleaseNativeTownSaveHiredMercenaryFlagOnlyFileStillLoads` above,
  which drives `restoreHiredMercenaries` directly against a campaign record
  carrying a set hire flag and pool count with no persisted actor record for
  that type — the exact shape a file written before this story (or before
  story 1127) carries — and confirms the fallback still rebuilds the correct
  squad.
- **Milestone census** (`AGAINROM_ASSETS=<root> missionrun -mission {10,20}
  -trace -ticks 1 | grep -c UNSUPPORTED`, both roots, against
  `pipeline/milestone-baseline.txt`): **0 unrunnable script nodes for mission
  10 and mission 20, on EN and on RU — unchanged from the baseline.**
  Expected: this story touches only `pkg/game`'s SAVE/city persistence
  surface, no file under `pkg/sim` or `pkg/mapload`'s script/pathing code.

### Correction pass proof

- `gofmt -l .`: clean (touched files: `originalparty.go`, `nativecity.go`,
  `nativecityitems.go`, `nativecityrestore.go`, `nativecity_release_test.go`).
- `go test -trimpath -count=1 ./...`: all packages pass, including
  `internal/gatedtests`' population census at 190 (net +3 over the first
  landing's 187: the two F-3 order witnesses and the new F-1 combat-block
  witness named below).
- `scripts/check-no-game-assets.sh`: clean.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 190 gated
  tests, 2 roots; **190 of 190 ran and 0 lacked a subject, on EN and again on
  RU.**
- **New field-by-field release witnesses**
  (`pkg/game/nativecity_release_test.go`), EN and RU:
  - `TestReleaseNativeTownSaveHiredMercenaryCombatBlockRoundTrips` (F-1):
    hires a solo type-10 squad, SAVEs, reloads. Asserts
    `mapload.PartyDisplayWithTable`'s complete `data.Derived` (Body, Reaction,
    Mind, Spirit, Skill/SkillXP, HealthMax/ManaMax, the full `Combat` block —
    DamageBase/Spread, ToHit, Defence, Reach, AttackCharge/RelaxTime,
    SpellName/Power — Protection\[0..4\], Speed, Sight, RotationSpeed,
    Capacity) plus HP and mana equal the live pre-save value, member for
    member.
  - `TestReleaseNativeTownSaveHiredMercenarySiegeThenHumanOrderRoundTrips`
    and
    `TestReleaseNativeTownSaveHiredMercenaryHumanThenSiegeFallsBackToAGSOrderPreserved`
    (F-3): hire a siege engine and a Human type in one town visit, both
    orders. Siege-then-Human: `ExportNativeCitySave` accepts, `IsOriginal`
    true, restored `MercenaryType`/ID sequence exactly matches live.
    Human-then-siege: `ExportNativeCitySave` refuses, `IsOriginal` false,
    restored order via the `.ags` fallback still exactly matches live —
    proving the fallback preserves order even where native cannot, and that
    the two orders now trade places exactly as they did between master and
    the first landed candidate (review's own measured table).
  - `TestReleaseNativeTownSaveHiredMercenaryEmitsNativeSAVAndRoundTrips`
    (extended, F-2): after its pre-existing identity/worn/carried/spell
    comparison, additionally decodes each of the three hired identities'
    own typeWord back through `(*CityProvenance).Human` (keyed by identity,
    not by name-match position, so three same-named hires are checked as
    three distinct actors) and asserts it equals `uint16(member.Class)`.
  - `TestZZDiagMercItemEffects` and the two files copied in from
    `tmp/review-1129/held/` (`rev1129_test.go`, `rev1129chain_test.go`) were
    diagnostic-only, run to confirm each fix and then removed; none is
    committed.
- **F-4 confirmation** (reused, not committed, the reviewer's own held
  `tmp/review-1129/held/rev1129chain_test.go`,
  `TestReview1129TwoCycleChainWithChangedGear`, EN): re-run against the
  corrected candidate. `Temporary` and `HiredRotationSpeed` now match
  live-to-reloaded for every member (both were `true`/`17` on both sides,
  where the review found `false`/`0` on the candidate). The stripped
  member's `Weapon` still reads `<nil>` after reload where live already
  carried `Worn[0] == 0` with `Weapon` still set — the disclosed, unfixed
  half of F-4 (see "Open debt"). A weapon display-name text difference
  (`Iron Mace` -> `Common Iron Mace`) reproduces identically on the hero's
  own unrelated weapon, confirming it is pre-existing and out of this
  correction's scope, not something F-1's item-effects fix touches or
  worsens.
- **Milestone census, re-run on the corrected candidate**
  (`pipeline/check-milestone.sh`, `AGAINROM_MILESTONE_DRIVE` pointed at a
  `missionrun` built from this exact worktree, both roots): **exit 0, "the
  script gap and the drive are where they were recorded" — every one of the
  28 shipped campaign maps' own checks/instants/triggers count matches
  `pipeline/milestone-baseline.txt` exactly on EN and on RU, and the
  unattended mission-10 drive reaches the identical recorded outcome (lost
  at tick 304, 4 of 36 units moved, 1 fell) on both.** Unchanged from the
  first landing. Expected: the correction touches the same `pkg/game`
  SAVE/city persistence surface as the first landing, still no file under
  `pkg/sim` or `pkg/mapload`'s script/pathing code.

## Open debt

**Name-based hire recognition is this project's own convention, not read
from either promoted claim.** SAV-616 and SAV-617 establish the actor
record's shape and grouping; neither reads a Name field. A hired member's
`NPC%02d_%d` name is this tree's own hire-time template
(`buildMercenarySquad`, pre-existing) reused verbatim as the load-time
recognizer (`mercenaryHireTypeFromName`) rather than a second, independent
parser — a deliberate, disclosed choice (the function's own doc comment
states it plainly), not a gap silently papered over. A future rename of that
template, or an installed Humans table whose name for a given type collides
with the `NPC%02d_%d` shape by coincidence, is unexercised by any witness
here.

**Siege engines and pre-story-1127 files are unaffected by design, not
verified against a second, independent shape this story could have chosen
instead.** SAV-616's retraction is scoped to classKey 58; a siege hire
(`Unit(0x198)`, MERC-LEVEL-005) is a different wire class no claim retracts
SAV-608 for, so it keeps the pre-existing rebuild-from-count fallback and the
pre-existing equipment/order refusal-and-`.ags`-fallback pair, unchanged by
this story.

**The pre-existing 55-file corpus's own 14 classKey-58 files are not
retroactively certified as hires.** SAV-616 states this limit itself: no
before/after pair exists for those files in EXP-0309, so this story's own
read side (`mercenaryHireTypeFromName`) recognizes a hire by name, never by
classKey alone — a file whose classKey-58 record does not carry a
recognizable `NPC%02d_%d` name falls through to `data.FindHumanByType`'s
pre-existing many-to-one fallback exactly as any other unrecognized Human
record does, not to a mercenary-specific path.

Out of scope, unchanged, per the brief: the imported-city path for a
genuinely-imported ROM1 save (no test here proves a change to it, so none
was made — `TestReleaseCitySalesUseSAVAndFreshProcesses` and the
`OriginalStore`-backed corpus tests are untouched and still pass); mission
saves; the original process's own write boundary; DIV-906 (campaign-complete
refusal).

### Correction pass open debt (`pipeline/reviews/story1129-pass1.md`)

**The typeWord rule (F-2, DIV-918) is confirmed at two of thirteen Human
tavern types, generalized to the rest.** SAV-616 (tavern type 14) and
SAV-627/628 (a directly-authored placement, not a hire, on the row a tavern
type 10 hire would use) independently agree the marker equals the placed
actor's own Humans-row TypeID. No claim reads a third tavern type's own
hired actor. The generalization to types 3, 4, 5, 7, 8, 9, 11, 12, 13, 15 is
this project's own inference from two agreeing origins, cross-checked
empirically against the installed EN table's own row data (not a new
claim), disclosed rather than silently assumed.

**A worn slot changed since hire keeps reduced combat fidelity (F-1
residual boundary, DIV-889).** `restoredHireWornEffects` backfills a worn
item's `Effects` list only when its restored Code still equals the hire's
own Humans row template Code for that slot. A slot re-equipped from the
shop since hire decodes with `Effects == nil` (no wire position for it
exists at all in `nativeCityAttachItems`'s own item writer, an
unresearched-byte-format question this correction does not open, B1) and so
underderives ToHit/Speed/ScanRange/Protection for that one slot exactly as
the first landed candidate did for every slot. This is the same,
not-narrowed boundary `TestReleaseNativeTownSaveHiredMercenaryChangedEquipmentRoundTrips`
already accepts for worn/carried CODE identity; it was not previously
visible as a combat-stat gap because F-1's own all-zero bug already masked
it for every slot, changed or not.

**A stripped hired member's stale `Weapon` field is disclosed, not fixed
(F-4).** Per the review's own "Honest direction": the live value was
already inconsistent before this correction (`Worn[0] == 0` while `Weapon`
stayed set, because the shop-strip route never clears it), so the reloaded
`<nil>` is arguably the more correct of the two, and this correction does
not manufacture a second staleness to match the first. A player who strips
a hired member's weapon in the shop and then saves and reloads natively will
see `<nil>` where a `.ags`-fallback reload previously showed the stale
display name.

**A weapon's own display-name text is pre-existing and unrelated to a
hire.** `Iron Mace` reads back as `Common Iron Mace` (and `Bronze Pike` as
`Common Bronze Pike`) across a native round trip for the hero's own weapon
exactly as for a hired mercenary's — confirmed by re-running the review's
own held two-cycle chain witness against the corrected candidate. Neither
F-1 through F-4 nor this correction's own item-effects fix touches weapon
name resolution; this is unmodified, out-of-scope, pre-existing display-only
text, not a combat-affecting field.

## Touched surfaces

`pkg/game/nativecity.go` (`nativeCitySiegeMember`, `nativeCityHumanHiredMembers`
new; `nativeCityDefRow` gains the name-first hired-member branch;
`nativeCityData` appends a second `Player` group for hired members through
the existing per-member producers; `nativeCityHiredEquipmentMismatch`'s call
site and the renamed `nativeCitySiegeHireOrderMismatch` narrow to
`nativeCitySiegeMember`), `pkg/game/nativecityrestore.go`
(`restoreHiredMercenaries` gated by `present[typ]`, now a per-type fallback),
`pkg/game/originalparty.go` (`restoredDefinition` gains a `TypeID` return;
`restoredMember` recognizes a hire via `mercenaryHireTypeFromName` and skips
`HeroAppearance`'s composed body for one; `mercenaryHireTypeFromName` new),
`pkg/game/originalsave.go` (`marshal`'s roster-count check and its
`nativeCityHiredEquipmentMismatch`/order-check loop both narrow to
`nativeCitySiegeMember`), `pkg/game/nativecity_release_test.go`:
`TestReleaseNativeTownSaveHiredMercenaryEmitsNativeSAVAndRoundTrips` (body
edited: an inline identity/worn/carried/spell comparison replaces the reused
`nativeCityHiredEquipmentMismatch` deep-equality call, name unchanged);
`TestReleaseNativeTownSaveHiredMercenaryChangedEquipmentRoundTrips` (renamed
from `...ChangedEquipmentRefusesAndAGSFallbackRoundTrips`; body edited the
same way, plus a derived-health comparison and the LOAD call fixed to use
the tokenized `SaveSeams` name instead of the raw one);
`TestReleaseNativeTownSaveHiredMercenaryDescendingOrderRoundTrips` (renamed
from `...DescendingOrderRefusesAndAGSFallbackRoundTrips`; same LOAD-call
fix); `TestReleaseNativeTownSaveHiredMercenaryFlagOnlyFileStillLoads` (new);
`TestReleaseNativeTownSaveSecondSaveAfterReloadHiredEquipmentStrippedFallsBackToAGS`
(strip-loop guard fixed to read live `shopWornItemSlots` instead of the
stale `Worn` snapshot). `internal/gatedtests/testdata/population.txt` (net
+1: the three renamed/new names above in, the two stale refusal names out),
`docs/DIVERGENCES.md` (DIV-884, 887, 889, 893, 902, 903, 904, 907, 910
amended).

No `formatVersion` or pinned-digest constant moved.

### Correction pass touched surfaces

`pkg/game/originalparty.go` (`restoredDefinition` return signature gains an
8th value, the row's own `RotationSpeed`; `restoredCarry` gains a `hired
bool` parameter, gating `Carry.LiveLoad` on `!hired`; `restoredMember`
threads `HiredRotationSpeed`/`Temporary` from the new returns, calls the two
functions below; `restoredHireWornEffects` new), `pkg/game/nativecity.go`
(`nativeCityToken` gains a `typeWord uint16` parameter; `nativeCityHeroTypeWord`
constant and `nativeCityHumanTypeWord` function new; `nativeCityUnitData`
selects between them; `nativeCitySiegeHireOrderMismatch` rewritten to score
only a siege member against an already-seen Human hire or an out-of-order
siege type; `ExportNativeCitySave`'s refusal message text updated to match),
`pkg/game/nativecityitems.go` (`nativeCityItemToken`'s call site updated for
the new `nativeCityToken` parameter, passing `nativeCityHeroTypeWord`
unchanged — an Item head's own marker is a different, out-of-scope field),
`pkg/game/nativecityrestore.go` (`restoreHiredMercenaries` rewritten: fixes
`insertAt` at the first already-restored Human hire before the loop,
inserts each rebuilt siege squad there instead of at the tail, clears the
shifted tail's own stable IDs so `NameParty` re-derives them),
`pkg/game/nativecity_release_test.go`
(`TestReleaseNativeTownSaveHiredMercenaryEmitsNativeSAVAndRoundTrips`
extended with the typeWord-by-identity assertion;
`TestReleaseNativeTownSaveHiredMercenaryCombatBlockRoundTrips`,
`hireForSiegeOrderTest`,
`TestReleaseNativeTownSaveHiredMercenarySiegeThenHumanOrderRoundTrips` and
`TestReleaseNativeTownSaveHiredMercenaryHumanThenSiegeFallsBackToAGSOrderPreserved`
all new), `internal/gatedtests/testdata/population.txt` (net +3: the three
new test names above), `docs/DIVERGENCES.md` (DIV-907 and DIV-889 further
amended; DIV-918 new, of the reserved range), `docs/1129/story.md` (this
section and the ones above it).

No `formatVersion` or pinned-digest constant moved by the correction pass
either.
