# Actor roots and retained timed effects

Original mission LOAD now restores retained Unit-owned timed effects without
applying their saved modifiers twice. The ordinary simulator owns countdown,
expiry, action gates and effect marks. Native SAVE and fresh LOAD keep the
timer. Snapshot projects its current operand and owned U144 bit; expiry
retires the exclusive child and remaps object bindings without changing keys.
`publishSource` also publishes the current MoverSpeed as RotationSpeed after
a live derive. Loading itself still does not derive.

The branch starts at 7a2b2f44238e8e4c9440f57de7c47f267a5085c2 and reconciles
published main c42aa113ebfbbf26c6b01a1fb82b8132f90a3cfb before final gates.
The public knowledge pin remains k6, 84f328362e6c946303da3d15dddfcefbcfec4946.

## Authority and finite disposition

MAGIC-CONSUME-144 preserves already-modified actor values and remaining
counters at LOAD. MAGIC-ATTACH-016, including its potion-id-0 and continuous
expiry amendments, and HERO-EFFECT-019 bound the existing timer and source
effect consumers. SAV-UNITLEN-045, SAV-UNITPROG-156, SAV-SPELL-044 and
MAGIC-BOOK-002 supply actor roots, child widths and sparse book membership.

| Retained surface | Disposition |
|---|---|
| Held weapon/shield, twelve worn references, pack | Independent Body-to-Document edges and complete reached Item/Effect/Spell fields; live equipment, pack values and exact item identities compared |
| Inventory presence, insertion index, accumulator | Complete Document and live container compared; nonzero presence flags stay nonzero |
| Spellbook presence/header/count/slots | Complete Document and live book parameters/membership compared; absent, present-empty, sparse slots and aliases remain distinct |
| U68 and Humanoid Diary references | Exact raw archive-to-Document references retained; no invented reference consumer |
| Unit final17 bytes U5C/U64/U44/U40/U48 | Every value retained and independently compared; no invented semantic consumer |
| Unit Effects | Supported bare, exclusive mode1/2 effects restore current timers, with nominal signed operands, original child bindings and native persistence |
| Late-dead actors | Independent HP/stage decides raw-only exclusion; their complete actor/child graph remains compared |

`TestMilestone2ActorRoots` is the sixteenth registered M2 family. It reads
Body values and counts before Snapshot, reusing 1158's population/identity
proof and 1151's child byte reader. Decoder fields and imported survivors
never supply expected actor values or admission. The independent corpus
instrument reports 62 worlds,39 city documents,2,837 actors,4,957 child rows,
2,435 live actors,311 raw-only actors,42 live books,192 non-null spell slots,
four live timed effects and one U68 reference. It names refusals and has a
nonempty population guard. Both EN and RU mismatch/refusal counts are 0/0. One further discovered file,
the invalid-magic game9000.sav control, is named unreadable by the shared
corpus reader; no resumed world is omitted.

## Proof

The baseline original importer lost game0007 actor34/Effect36 entirely.
The unchanged lawful file has magnitude100 and remaining776; its SHA256 is
a7cb35ea5d87c9c7a9b8cfd70f8a1f61b6927ca089e468a9d6fe983da777156e.
`TestReleaseActorEffectsRestored1161` is registered in the release population
(231 to232). A private derivative changes only its counter776 to4. Title
and map-menu LOAD, changed menu SAVE at3, private source removal, fresh
FrontEnd native LOAD and three paired advancing ticks prove expiry removes
the modifier, owned mask bit, owner edge and exclusive child. Post-expiry
native LOAD does not resurrect it. Separate current World-to-Document
assertions check the operand and binding, in addition to paired World hashes.

Permanent controls reject65 actor field/count/edge mutations,88 child
scalar/identity mutations, nine reader population/bounds faults, living and
dying omissions with a survivor, wrong live identities/books/classes, eight
native effect metadata/operand/mask losses, implicit retirement and aliases.
They also prove continuous expiry skips unapply, the frozen counter boundary,
action blocking, no derive during LOAD, and atomic invalid import. A live
expiry control first failed with Source MoverSpeed18 / RotationSpeed25 and
now requires18/18. Legacy native Documents never re-import historical effects;
their missing exact bindings become explicit projection unavailability.
Effect-free predecessors keep nil metadata and unchanged Documents. Genuine
frozen1115 LOAD/Snapshot/continuation fixtures pass without regeneration.
The three current-descriptor envelope checks account for the additive gob
field; the inner World form remains90. The exported method inventory marks
the new import method as an atomic writer.

The sole independent pass returned R1: Haste24(+7) expiring while Slow28(-7)
remains can derive a zero turn rate during an admitted turn. Its single
correction settles that turn using the existing native re-arm compatibility
policy; positive-rate changes retain the admitted interval. MOVE-TURN-044
leaves ROM1's zero-rate lifecycle Unknown, so this is a native saveability
guarantee. The permanent App test uses real `mapload.BindSourceDerive`, fails
before the correction at tick3 with a zero-rate8/10 turn, then proves ordinary
menu SAVE both before and after expiry, two fresh source-free LOADs, subsequent
ticks, and independent current Document speed/facing/rate/mask/operand checks.
No second review, fixture regeneration or additional divergence ID is used.

Focused receipts are private under `review/story1161/`. The EN/RU App and
new corpus family pass. Executed no-import and stale-projection Go overlays
both turn the registered App witness red (missing attachment; counter4
retained after World advances to3). The actual built missionrun executable
resumes unchanged game0007 and prints id0/magnitude100/remaining775 after one
tick. Its mission10/20 UNSUPPORTED counts are0/0, unchanged from the main
baseline. No desktop events or original process were used. The seat owns the
remaining full EN/RU release/M2 chain, sole independent review, serialized
landing and builds/current publication. Candidate Go/gofmt/assets receipts
are returned with the exact pushed SHA.

## Remaining boundaries

No full world SAV writer or generic object lifetime is added. Unsupported
Effect classes/kinds/modes, aliased/shared actor children and duplicate ids
leave that actor's original effect subtree retained but unarmed and explicitly
unavailable for current projection. New/replaced native attachments have no
original child identity and name that projection gap. Caster attribution is
absent from the original actor Effect record. Native legacy nil bindings do
not imply current Document effects. Extended CString forms and unobserved
reference constructors remain outside this bounded acceptance reader.
Offset-only root/tail semantics, exact Humanoid runtime acceptance, fourteen
original equipment roles and superseded motion projection remain separate
debts; this story closes the demonstrated DIV-1106 publication seam, records
remaining actor attachment projection as DIV-1107, and leaves DIV-1092 open.
It does not claim complete SAV fidelity.
