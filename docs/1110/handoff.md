# Story1110 equipment checkpoint

## Ownership and publication boundary

Continue only in `wt-story-1110-actor-container-load`, branch
`story1110-actor-container-load`. Same story, same allocation and no new
review round. DIV762..769 are reserved; only DIV-762 is used. Never modify the
seat, master or another lane's worktree. Never consume the explicitly denied
1107 correction `df47dfb8`, including through a proxy or repackaging.

Parent requested a coherent checkpoint and normal push, then serialized later
composition. Equipment work started from `399c7918`, already reconciled with
published implementation `4dbd7fa1`. Code checkpoint `22f8ea2e` was reconciled
with exact published `e20d054ef1c91ec51847278e6613ac7400ee5dff` (current-facing
hotfix) in `0578d8d47cc0745a8900f9206148fe6cecb389c3`. The sole conflict was the
explicit World method/writer lists: retained-source methods and the new facing
import method were both kept.
Research moves forward from `1172d41a` through EXP0288 and its operand-source
clarification `d469bd46` to promoted `659c3f69d4cb579006d4b584e0246cf309a40fc7`.
No original executable, install or owner save was written.

## Implemented after the retention-only checkpoint

- `sim/sourceequipmove.go`: actual atomic source equip/remove command and
  per-class item virtuals; exact direct fields, load events, ordered Effect
  calls, same-slot return, owner-policy shield displacement, cadence/range,
  Weapon-owned Spell lifecycle and terminal death/container reset. Command22
  final L(0) is not inserted into each death-removal virtual.
- `sim/sourceitemeffect.go`: local state-0 dispatcher with class gates,
  source-width arithmetic and one derive per Effect. Unknown Token/subclass
  callbacks are not flattened into this path.
- `sim/sourceeffect.go`, `effect.go`, `sourceactor.go`: generic supported
  attached effects and expiry use source modifiers and nominal operands;
  callback failure retains the record/timer. Direct source stores do not
  refresh the spellbook; completed Human derive does. Current defence and
  absorption are overlaid into SourceNow so death/revive stores remain inputs
  to the next equipment producer.
- `mapload/sourceequipment.go`, game shop/town handlers: isolated clockless
  world uses the same source producer for pack/shelf/mine-table/merchant-table
  equip and doll removal. Member, shop source and purse commit together.
  Source school training continues after equipment/export-basis retirement.
- `mapload/sourceconstructor.go`, sim weight/prefix: native-generated items
  acquire reached definition-derived operands, persisted in the existing
  code/weight population's optional source-equipment records. Explicit saved
  weights/concrete classes block constructor refill. Partial scale tables
  do not invent source identity factors. Constructor records are not saved
  item overlays and conflicting declarations fail atomically.

Selected permanent/timed potion, book consumption, scroll reservation/refund,
source transfer on both actors, sack pickup, carried drop, scripted inventory
changes and city purchase/return transactions were implemented in the preceding
checkpoint. Equipment work preserves those paths.

## Canonical and scheduling seams

Unpublished form75 remains form75. SourceActor is now 209 bytes, actor-load
record 224 bytes; EquipmentRuntimePresent discriminates saved Reach/Charge/
Relax bytes, including reach zero. Decoder reach validation runs after source
actor records restore that presence. SourceEquipment is 77 bytes: class/row/
own-kind, W52, W6A/A52/S50, reached Weapon columns5/14/12/13/15, owned Spell
ID/range/defensive/mana and unsupported-state flag. Sparse constructor records
follow live item owners in the same optional prefix; the ordinary code/weight
wire remains six bytes per row. No source data means no optional prefix.

Earlier public forms initialize absent source state. Current gob-descriptor
hashes changed; original frozen public-envelope fixture bytes did not. Version
tests remain version-free. Generic attachment expiry now leaves its countdown
unchanged on a refused source callback; terminal decay does not advance past a
failed source-loot transaction. No saved world-clock restoration was added.

## Coordination with1111

1111 owns source actor registry, source record-to-EntityID binding, effective
Player owner precedence, Group reconstruction and any additional identity/
definition/facing fields. This lane does not change independent MapUnitID joins.
The public current seams are sav.ActorBasis, game.originalActorBasis,
game.originalActorLoad, sim.OriginalActorStock.LoadState and
sim.ImportOriginalActorStock. ActorBasis retains class/stat words/XP and
CityHuman raw blocks plus equipment runtime bytes. SourceActor has no saved
Token row, raw U49/U4A/U4B/U4C, saved name, owner ID or runtime identity.
HasOwner/ManaReservePercent still come from the actor Token Reference's Player;
1111 must compose effective-owner-derived inputs separately. Root owns the
published current Facing restoration and its next world-clock inquiry.

## Focused proof

All expectations are literal source-width values or fixed source offsets;
production Human derive is not an expected-value oracle.

- `sourceequipmove1110_test.go` (sim/mapload): replacement callback trace,
  late failure rollback, partial and duplicate stacks, source death including
  suppression and parameter15-zero weapon retention, actual armor/melee/ranged
  commands, fresh-native next removal and noncommuting Effect order.
- `sourceconstructor1110_test.go`: exact constructed block values, explicit
  saved BaseItem/nonzero concrete block refusal to refill, missing table,
  generated weapon/armor after fresh native decode, positive onehand shield
  attach/removal with literal live values. sim constructor conflict test.
- `sourceequipcity1110_test.go`: four actual city origins, native city
  continuation, fifteen late/admission refusal cases preserving all owners.
- `sourceeffect1110_test.go`: nominal source operands, generic kind routing,
  callback-failed expiry/continuous timer preservation.
- Existing source tests cover original LOAD doors, source Human/Unit load,
  literal skill/potion producers, source transaction failures and corruption.

The natural valued staff witness in originalholdings_release_test.go uses the
read-only corpus `gameversions/saves/2026-08-15/game0017.sav`, SHA256
`eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0`.
It independently anchors saved Weapon/Armor blocks and Spell values, then runs
both App LOAD routes, native continuation and controlled death/loot/re-equip.
No original-runtime equipment cycle was observed. A synthetic existing attack
fixture now explicitly writes source reach1/charge8/relax4; source zero is not
coerced into that fixture's intended attack range.

Progress logs outside Git are under `review/story1110/`:
`equipment-current-packages.log` records data2.261s/sav0.595s/sim11.374s/
mapload0.693s passing; game initially failed only the current descriptor hash.
After updating that expected current hash, `equipment-game-packages.log` passes
game6.473s. Focused source-constructor/equipment tests pass all three packages.
Earlier natural EN producer checks pass in `equipment-natural-producer-en.log`.
At reconciled code `0578d8d4`, `go test -trimpath -count=1 ./...` passes and the
asset guard prints clean. Focused natural EN and RU each pass both cases; the
valued staff native hashes agree across roots, before re-equip
`795813289d2c5782`, after `8e9fac852d493841`. The logs are
`equipment-reconciled-go.log` and `equipment-reconciled-natural-{en,ru}.log`.
`verification.md` records the headless mission census. Final pushed SHA is the
doc-only successor of this tested code, to be reported by the lane.

## Explicit remaining work and debt

No acceptance or final review is claimed. Parent controls final1107/1109
dependency publication, final reconciliation, risk-selected gate chain and the
existing one-pass review budget. Full Go trimpath/count1 and the asset guard
pass at this reconciled checkpoint, not a later composition. Remaining final
checks: relevant EN/RU release through one paired gate, relevant
headless carry/equipment scenarios, claim/divergence guard, clean diff/trailers
and final outgoing-payload verification on the composed candidate. Mission10/20
one-tick UNSUPPORTED counts are 0/0; the script populations match the seat
baseline. This slice is not intended to change that population. No GUI window
was opened or driven.

Unfinished fidelity boundaries are concrete:

- Saved Weapon-owned Spell is retained and its equipment lifecycle executes,
  but existing combat cache still derives spell ID/power from kind41. Arbitrary
  saved nested-Spell/effect disagreement is not a complete next-cast proof.
- Generated constructor records contain reached operands only. Unknown
  original attack-block tail bytes and whole original first-SAVE defaults are
  not authored. The canonical zero in unused fresh fields is not ROM1 evidence.
- Actor-Token kind1, arbitrary Effect Token states/subclasses/list mutation,
  unsupported signed teach-Spell removal, invalid database IDs and allocation
  failure recovery refuse or remain outside the admitted model.
- Notifications/global effect state, original session clocks, whole original
  object identity/aliases and transitive Token/Position closure remain open.
- Full spell-release cost/cancellation/observation atomicity under every source
  callback failure is not proved by the item transaction tests. The untimed
  payload now propagates source callback refusal; a complete casting audit is
  not implied. Actual combat/book skill-feed matrices beyond existing source
  sink tests need separate final coverage assessment.

Use GOCACHE=<seat>/.cache/go-build. Use full Git Bash for seat
guards. Do not stash, reset, force-push, write installs or route around a
security rejection.
