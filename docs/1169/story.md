# Imported Human return through city SAV

The imported `game0010.sav` party can complete mission30 with earned spell XP,
save an ordinary city SAV, load it in a separate process after deleting the
private source copy, and move and cast in mission40. A second ordinary SAVE is
also SAV. The result covers the existing source Humans with unchanged holdings
and book membership. Arbitrary retained new items still require AGS.

## Authority and implementation

The public knowledge pin supplies `PARTY-ENDCULL-026`: expire attached effects,
then restore kept actors' HP/Mana from their maxima and reset placement/action
fields; inventories remain. `NormalizeMissionSurvivors` performs the supported
inverse/derive on an isolated candidate before `CarryRoster` loses the live
mapping. It restores pools after removal. Failed inverse, scroll cancellation
or removal that fells an actor refuses completion before rewards or town changes.
The complete mission remains available for native SAVE.

`SAV-REGENWIRE-532` establishes independent residue bytes. Their first-use
lifetime remains Unknown. The narrowed `SAV-EQUIPORDER-552` establishes local
range arithmetic and literal timing writes, not restoration of a former value.
Return therefore carries current reach, charge, relax and regeneration residues;
it does not infer defaults or erase them with action state. Older native party
snapshots default omitted residues to zero. The source import version advances
from 5 to 6, with independent legacy baseline validation.

The current return is a separate checked representation. Its producer joins
source character identity -> binding PartyID -> mission party -> Start.IDs ->
surviving live entity. Type/role, owner, hero marker and current load agree at
that boundary. Fresh minted mission actors retain this engine's existing
`HumanTypeID` compatibility policy; the source Human TypeID stays independently
checked and preserved. Names, party order and coincident runtime IDs are not
alternate joins.

The semantic source and reconstructed baseline stay immutable. The city writer
projects current fourteen Human words, six XP values, separate aggregate,
Attack/Base/Defence/Modifier blocks, mover speed, equipment timing, residues and
existing spell parameters. It applies the established stage/reference reset.
Historical Saved placement, action, effect references and world item ObjectIDs
do not enter the returned member. Town progress remains source-bound. Native
AGS persists the separate return and independently validates its source baseline
before a later ordinary SAV export.

Admission requires the same roster, source graph, item metadata/order/counts,
equipment slots, container bookkeeping and book membership. The normalized
modifier must equal the source modifier. Unknown source attached effects,
changed modifier, prior sale replay, legacy baseline, changed roster or topology
keep the existing AGS refusal. No new Human, item graph or world writer is added.

## Proof

`TestReleaseImportedReturn1169SAVColdNextMission` reads the preserved city with
SHA256 `89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4`
on EN and RU and uses the real App mission30 opener. Ordinary Light12 and
Shield18 commands change the mage's XP from `[0,0,1593,0,0,0]` to
`[0,0,1593,3,6,0]` and skills from `[0,0,10,0,0,0]` to `[0,0,10,1,1,0]`.
Shield absorption 3 is active at completion and returns to 0; the earned values
remain. HP 20/22 and Mana 127/137 return to 22/22 and 137/137.

The reach fixture teleports the hero to `(64,15)`, and `KindDamage` supplies
bounded injury controls. This is not a whole-map walk or a combat witness.
The installed victory script consumes its own quest item `0x0e1e` and wins.
The test proves exactly that quest addition/removal, unchanged other holdings
and equipment, and unchanged earned XP through the trigger. It calls actual
`FinishMissionWithRoster` with the live party, IDs and roster; no test drops
loot or synthesizes the positive outcome.

The output oracle reads primitive CArchive fields and raw blocks independently
of the current-Human projector. The cold process has only the new SAV and a
derived expected-value file. It proves campaign40, a second SAV, exact current
World values, next-map placement, movement and Light mana spend. An additional
AGS round trip checks the persisted return representation. The existing1168
campaign/NPC acceptance witness remains, and its controlled zero-tick imported
return witness now expects SAV.

Single-cause controls reject lost identity, current XP, cleanup, producer,
equipment, world handles, historical placement, forged baseline/identity and a
missing actor. Retaining the ordinary mission30 quest grant without its victory
consumption explicitly keeps AGS. Sim tests distinguish current residues and
timing, prove source inverse cleanup and atomic failure, including death/action
side effects. Format tests check byte-wide 157/255 residues and unchanged source.

Focused EN/RU tests and the manifest guard pass. The seat owns the sole review
and the final full Go, paired release, M2, asset and exact-main build gates.

The candidate includes published6f65204 and k13. SnapshotCityReturn adds141
descriptor bytes even when its value is absent. Existing historical fixture
bytes remain frozen; an actual synthetic form92 envelope emitted by6f65204 is
additionally preserved. Only three current-descriptor hash controls are updated.

## Remaining scope

DIV-1162's former missing scalar return producer is closed. DIV-1169 keeps the
new-item/roster/modifier and source compatibility refusals explicit. Retained
earned loot needs an incremental item graph producer with preserved identities,
ownership, ordering, counters and effect children; the generated-city graph
builder cannot be substituted into an imported source. Original executable
acceptance, unknown field lifetimes, campaign150 and world SAV writing are
separate work. Consumed native companion grants are delivered by inherited6f65204.
