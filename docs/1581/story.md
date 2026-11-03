# Animate Dead

## Intent

The owner requires the second game's Animate Dead to raise the corpse as the
original does. The first game's Control Spirit keeps its result and World hash.

## Authority

Knowledge pin k406 is `c6410e6d8023e4e5a695f9ec0761185c2159493b`.

| Claim | Decision |
|---|---|
| R2-ENGINE-780 | Kind priority 2, 3, 4; first corpse in that kind; typeID outside 82..98; consumed health -10001 and kind 5 |
| R2-ENGINE-781 | F_Zombie.1, F_Skeleton.1, Ghost; seven corpse stores, six for Ghost; signed division toward zero; current health equals the new maximum |
| R2-ENGINE-782 | Radius-zero positioning at the clamped corpse cell; consumption survives failed placement; caster owner and a separate Group after success |
| R2-ENGINE-783 | No summon key; suppressed death rewards; bounded lifetime and RU pool-class Unknowns |
| R2-ENGINE-776 | Corpse ageing and the summon cap; a raised actor with no key is outside that cap |
| R2-ENGINE-777 | Units typeID is the corpse filter word; row values supply the remaining template statistics |
| R2-ENGINE-531 | Placement clamp 8..dimension-9; two attempts and no square scan at radius zero |
| R2-ENGINE-535 | The runtime reward mark is separate from the name-derived container policy; suppressed gold, items and bag; human experience skip is Medium; creature experience is Unknown |
| R2-ENGINE-788 | Shared runtime id and owner Group construction; unread post-construction callees keep the bounds |
| R2-ENGINE-789 | Shared plane-mask and cell-record placement facts, with remaining terrain and layer-zero Unknowns |
| R2-ASSET-202 | Unique template names and Units rows; template identity and type exclusion |

The R2-ENGINE-026 retraction rows replace its old inclusive undead filter,
Body rule and unordered kind selection. No retracted clause supplies policy.

## As-built behaviour

The arm table still routes second-game row 22 to the Control Spirit arm. Its
second-game profile supplies three corpse kinds, their names and body/health
rules, the type exclusion, reward mark and radius-zero positioning. The same
raise builder applies both profiles. ROM1 keeps its bones-only admission,
preferred target, Ghost definition, inherited Group and minimum health of one.

`raisedGhost` overlays the corpse statistics on the declared template. A
source-backed Units actor receives the same stores in its current SourceActor
blocks, so the normal SAV producer reads the changed values. Ghost retains its
template Body. The consumed corpse passes through the existing removal builder
and remains represented by its terminal tuple, including failed placement.

Both profiles call `createActor`. ROM1 supplies its inherited placement and
Group profile. ROM2 uses the same generated source binding, placement and new
Group construction as Summon. The raise profile skips the whole-World check
that ordinary creation runs. No cast-tail check replaces it.
Ordinary creation retains its validation. Runtime allocation belongs to the
candidate World, so a failed placement publishes no new actor allocation.

`createActor` sets `SuppressRewards` from the second-game raise profile. This
runtime mark discards death holdings and, under the second-game reward profile,
skips gold rolls and human experience. `SuppressCorpseLoot` retains its separate
name-derived container policy; either field discards the bag. Living people
with only that name policy still give experience. Optional byte form 132 carries
marked entity indices through the shared wrapper codec. Unmarked Worlds keep
their bytes; older forms default the mark to false. `ActorValues` carries the
mark through current SAVE and cold LOAD, with false omitted from its record.
No summon key or timer is attached. The viewer's body question resolves the
selected spell id through the same game arm table.

| Kind | One builder |
|---|---|
| Corpse selection and raise | controlSpiritCorpseAt, raiseControlSpirit and raisedGhost, with corpseRaiseProfile data |
| Spell arm | spellArms and its secondArmRow profile |
| Units creature | CheatActor, then the existing NewActor constructor |
| Runtime actor and Group | createActor, with actorCreationProfile data |
| Placement | secondPlacement, radius 0; the inherited profile supplies ROM1 placement |
| Death loot and experience | dropTerminalLoot, deathGold and awardSkill using SuppressRewards; discardsCorpseLoot combines the two bag policies |
| SAV actor | materializeCurrentWorld and the existing current actor record projection |
| Viewer corpse eligibility | gameFiles.spellTakesBody delegates to the arm table |

## Proof

The test-only commit precedes the implementation. Synthetic ordinary tests cover
each kind and its statistics, source-row stores, excluded types and boundaries,
kind priority, failed placement, the clamp, the cap, death rewards, body picking
and a second-game SAVE followed by a cold LOAD. Template declaration is tested
through the existing Units constructor. The static guard rejects actor-list or
Group construction in the raise builder.

The correction's test-only commit pins living NPC-named people's experience
before the fix. Ordinary tests also check the mark's hash, byte-form round trip,
malformed indices, absent default and separation from the name policy.

`TestFirstControlSpiritKeepsRecordedWorld` pins the base World hash
`038cb620815d4b71`. Existing Control Spirit, raised Ghost and second-game Summon
tests cover the shared builder's callers.

`cmd/animatecheck` reads a supplied install and prints its three template rows
and one corpse-derived raised actor per kind. It opens no window and writes no
install file. Its build was denied by the sandbox at build-tools.sh's mkdir;
there is no EN or RU witness measurement from this lane. The seat must build
and run it on both second-game roots before acceptance. Gate receipts and
remaining sandbox restrictions are in the owner return.

At the next release, run on EN and RU:

- TestReleaseSecondGameSpellTableLoads
- TestReleaseSecondGameMissionTenMageHealsTheHero
- TestReleaseSecondGameMissionTwentyOneMageCastsIceMissile
- TestReleaseSecondMissionTenHealSaveContinuation
- TestReleaseSecondGameAttachedEffectsActAsTheirOwnArms
- TestReleaseSecondCampaignStoneCurseAndSummonSurviveSAVEAndColdLOAD
- TestReleaseSecondGameBlizzardCast
- TestReleaseSecondGameBlizzardSaveContinuation
- TestReleaseSecondGameSpellsDrawAndSoundTheirOwn
- TestReleaseSecondGameGeneratedMageBook
- TestReleaseSecondGameLoadedSpellBehavesAsBeforeSave
- TestReleaseRaisedGhostSurvivesSAV
- TestReleaseRaisedGhostTakesItsWholeUnitsRow
- TestReleaseRaisedGhostCarriesTheInstalledUnitsRowRotationSpeed
- TestReleaseConsumedCorpseSAVLoad

No release test runs in this lane.

## Open debt

DIV-2616 and DIV-2622 are narrowed by row 22. DIV-4429 records pool reuse,
corpse-list order and identity limits. DIV-4430 records shared placement limits.
DIV-4431 records lifetime and creature reward Unknowns. DIV-4432 records the
bounded constructor, Human fields and message claims. DIV-4433 and DIV-4434
remain unused. No original-runtime equivalence or lifetime measurement is
claimed. Push and the sole adversarial review belong to the seat.
