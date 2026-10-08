# SAV source atlas

`join_sources.py` partitions every physical leaf in a bounded written SAV.
Decoded archive offsets and file offsets are separate. Word-codec packets retain
their decoded-to-file relation. Parent object ranges are metadata, not leaves.

The committed catalog contains guarded source candidates. Its source and overlay
pins are historical facts, not a claim about a later executable. The compact
catalog omits sample counts and offsets; each input's JSON is the offset authority.
Archive transport candidates can overlap narrower field candidates. Logical JSON
panel rows remain logical and do not subdivide a physical JSON payload.

```
python -B scripts/sav_atlas/join_sources.py --input <written.sav> --output <atlas.json> --source-map scripts/sav_atlas/source_catalog.json --decisions <receipt.json> --source-pin <producer-sha> --knowledge-pin <knowledge-sha> --capture observed=<capture.json>
python -B -m unittest discover -s scripts/sav_atlas -p test_sources.py
```

A receipt names the exact input and source-catalog SHA256 and the exact producer
and knowledge revisions. The command supplies both expected full revisions;
nonempty strings alone do not establish provenance. Each decision supplies `space`, a leaf
`selector`, relative half-open `start`/`end`, `source`, `guard`,
`producer_callsite`, `witness`, `source_pin`, `knowledge_pin`, `catalog_row` and
`selected_arm`. The selected arm equals its source. `guard_result` and
`source_present` are true for that selected arm. `prior_sources_absent` names
the preceding sources in the current/rule/Document/constructor order. The four source
classes are `current`, `rule`, `Document` and `constructor`. A current decision
also names its captured `carrier`. Rule and constructor decisions name their
`authority`. A Document decision names its debt. `meaning_unknown: true` needs
an authority and reason, and its exact matched catalog span must be classified
unknown-meaning; source absence and known missing basis never imply it.

`capture_sha256` binds every named JSON input. `capture_pin_paths` names each
capture's producer and knowledge revision JSON pointers. `guard_assertions` and
`source_assertions` contain true predicates on captured operands;
`prior_assertions` names a false eligibility conjunction for every preceding
source. Each predicate names its capture, JSON pointer, operation and comparison
value when applicable. Supported operations are `eq`, `ne`, `present`,
`nonzero`, `nonempty` and `any_nonzero`; equality distinguishes booleans from integers.
`carrier_branch` selects base, overlay, dead or terminal, and `catalog_guard`
equals that route's exact catalog guard. `expected_leaf_count` is positive and
must equal the selector's actual population. Boolean endpoints/row IDs fail.
Evaluated predicates alone establish a declared inventory. An implemented
`binding_id` adapter must also bind the exact selected arm, semantic operands,
carrier and written bytes. `source_inventory_joined` reports the declared byte
partition. `source_classification_joined` remains false for unsupported adapters.
The current adapters cover `World.RawSessionMid` and the nonzero
`World.RawSessionHead` arm. All400 middle bytes include zero. An all-zero head
selects a later arm and does not pass the nonzero-current adapter.
Detached session comparisons prove values only. Main-dispatch session binding
also requires exact input/capture/layout hashes, expected pins and concrete
module identity. A changed session producer file leaves classification
unverified even when its current raw bytes match.

`registry_binding.py` checks the24 sealed Item/Effect/Spell literal families.
The join passes an internal binding context containing actual SAV bytes, raw
writer capture bytes and independently supplied expected producer/knowledge
pins. This context is separate from CLI captures. The adapter reconstructs
ordinary records with `raw_reader.Walker` and field spans with `AtlasWalker`.
It reads Ownership/Bindings only from the framed YA1 path
`/CurrentState/AgainromActions`, through the typed JSON decoder. Marker scans
and caller-provided numeric object maps cannot select a physical member.

Each registry decision binds `input_sha256`, the exact raw writer
`capture_sha256`, and `layout_sha256`. The layout hash uses the current parsed
layout encoded as sorted compact ASCII JSON. `tool_pin` is the concrete
registry module SHA256, not the historical adapter revision. `prepare` builds
the exact current guard/value operands for a supplied binding context. The
current literal projector file must retain its code-bound SHA256. Changed
producer code needs a new explicit adapter and controls.

The 12 direct Item families can report `verified: true` for a bounded leaf source
join: Code, Count, F47, F48, Flags, Kind, Material, Position, Price, PublicationMask,
Shape and Type, with Code subject to the effective mod operands below. All use
the same current node/root/class branch.
The actual current registry supplies a known nonzero Token.Identity, a unique
native ID and at least one ordinary current Containers/ItemRoots occurrence.
Native IDs and wire keys are separate namespaces. The adapter rejects current
ID/key collisions and ordinary Identity/This collisions across classes, and
selects the exact typed physical leaf by its current key. Current aliases
select one node repeatedly. The direct Count 1..65535 word arm keeps explicit
current equipment class first, then the captured matching ItemWeights constructor
class when class zero is held or lacks explicit weight. Missing ItemWeights is
incomplete; a captured nil getter is an observed empty population. Its complete
ordered code population and valid constructor schema select matching-row presence
or absence. A zero constructor class reaches the installed table constructor and
then the Code.B fallback. Exact pinned code proves both branches select the same
class, whether table resolution succeeds or fails. This confluence proves class
only; constructor scalar values do not become literal source operands. Unknown
identity/count width remains unverified. The graph projector, ordinary-root
predicate, class constructor, definition binder and typed-code dependencies have
concrete code-bound hashes.

These joins use neither emitted Ownership/Bindings nor a retained document
as its source oracle. It reports `value_verified: true`,
`owner_relation_verified: false`, `holder_topology_verified: false` and
`complete: false`. It establishes which current registry scalar supplied the
written member, without classifying the raw holders of that node. Missing/unknown
keys, incomplete root/key populations, detached/session-only, keyless generated,
unsupported value and omitted branches remain incomplete. A changed dependency
hash leaves source classification unverified.

The 9 Effect literal families use an independently known child key and all
current keyed ordinary Item parents. Each parent passes the same Item node
guard. Its complete current Effects array must resolve to the exact raw ordered
edges; zero, retired or unbound siblings cannot be pruned into a new ordinal.
The raw incoming multiset must equal these independently resolved parent edges.
Shared parents and repeated edges require compatible complete current Effect
values. Unexplained incoming owners and external holders remain incomplete.

Spell.ID, Defensive and ManaCost have a source join only for independently keyed Weapon parents
whose explicit current equipment class, Spell ID and complete SourceEquipment.Spell value match the child
and exact raw OwnedSpell edge. Concrete hashes bind the late Book writer and
incoming helper: the protected Weapon child keeps its complete registry value when
current Book/World.Spells inputs request another value. ID requires 1..28;
Defensive accepts the exact uint8 carrier, including zero and 255. Shared Weapons must request the
same complete Spell value. Book-only children, keyless children and fallback
Spell values have no independent source join here.

The field table binds exact widths and per-field unknown bits. Missing carriers
do not acquire initializer values. Position arrays require 12 bytes; zero parts
are valid. Current and raw field loss, masks, missing operands, key collisions,
coherent emitted-map changes, child pruning and shared-owner controls exercise
these boundaries. All 24 direct families still report holder topology unverified
and `complete: false`.

Item.Code additionally requires the actual effective `/Capture/ModSetEmpty`
boolean. An empty set keeps the current Value.Code. A nonempty set requires the
actual ordered `/Capture/ModItemStandIns` getter, including Code, StandInCode and
StandInRow widths for every row. A captured nil slice is empty; an omitted
getter is incomplete. The final matching Code row wins. Its StandInCode supplies
F40 and its exact captured pointer and late standInModItems callsite replace the
initial projector metadata. Observed no-match retains current Value.Code.
Concrete hashes bind the final save composition, late writer, effective getters
and typed inputs. Class stays selected before substitution; final StandInCode
does not determine class, Shape or Material. No T0C or constructor arithmetic is
proved by this Code leaf.

Historical captures lacking these actual observer operands retain
`value_verified: true` and `verified: false` when bounded comparisons pass.
Raw mod configuration cannot replace the effective getter observations. Emitted Ownership or
Bindings, equal values and same-class correspondence are not independent owner
authority. No generic retained Item/Sack bridge is implemented. Historical sealed
controls establish only the premises each case actually supplies; their old
positive counts do not establish source classification for every family.
Whole-save source acceptance remains open.

Historical captures need separately named dependency evidence for the selected
class/root/priority branches. A current classifier result on an old capture is
a regression verdict until those historical source bytes are independently
bound. Function correspondence may bind the selected branch when an unrelated
helper changed its whole-file hash; it never replaces a frozen manifest entry
or establishes compiler dependency closure. An explicitly bounded Attack-only
function delta can preserve class convergence without establishing raw function
equality or equipment arithmetic. The old sealed93 controls remain a
bounded regression population.

YA1 choices supply exact `ya1_path` and `ya1_kind`. A shared pool leaf needs
`alias_decisions` for every additional owning label, with the same physical
source class and a `carrier_value` naming its captured path and byte encoding.
Every shared owner's captured value must agree over the shared subspan.
Contradictory owner sources or values fail. Transport parents cannot replace
narrower member decisions.

The producer witness must establish the selected guard, source correspondence
and mutation/loss sensitivity. A value equal to the retained Document is not
such a witness. The tool checks the physical/hash/schema join; it does not infer
the truth of a receipt's claims from equal bytes. Unwitnessed selectors,
overlapping decisions, outside spans, missing joins and known Document debt
prevent acceptance. Every unmatched catalog row needs an explicit `absences`
record with its row, zero emitted leaf count, producer pins, scope, absence kind,
actual guard/source-presence observations and witness. `capture_assertions`
bind each absence to observed operands. These checks establish catalog
population only. Their source bindings remain unverified. `scoped_absences`
names each omitted Pack/Book by class, archive index, branch and exact presence
leaf, even when another same-class object emits that body. Captured predicates
do not discharge these scoped producer bindings. Physical coverage
does not discharge those obligations or story gates. Framing needs explicit
rule decisions; the instrument never creates them implicitly.

`typed_payload_obligations` preserves CurrentAction members and scoped JSON
omissions, captured Residue fields and BSL1/NAB1 presence/layout separately from
an opaque payload leaf. Both have canonical World capture coordinates; these are
never SAV file offsets. Count, sorted EntityID/sparse slot rows, span and base
version are bounded independently. Their current/action/raw U44 source joins
remain unverified. A later source/schema delta needs explicit adapters and
controls; the historical catalog does not silently cover new fields.
NAB1 bounds component flags,24-bit Base knowledge,64-bit Modifier knowledge,
fixed107-byte actor records and its outer footer before checking nested BSL1.
Knowledge masks do not establish initial values or later mutation history.

Each physical leaf retains its literal bytes for value checks. The output names
the join module and every reader, binding and typed-payload module SHA256, plus
capture hashes and expected producer/knowledge revisions. These hashes identify
the instrument inputs; they do not establish compilation dependency closure.

`--catalog-only` writes the physical atlas and guarded candidates without source
decisions. An `incomplete` result remains incomplete and is not SAVE closure.
All inputs and outputs are explicit; neither entry point can overwrite an input.
The reader rejects unsupported Unicode strings, 32-bit archive tags, schemas,
unframed tails and incomplete spans. These limits remain in each output.

`import_catalog.py` compacts an explicit TSV inventory while retaining its input
hash. Catalog changes do not select a runtime branch or replace its witness.
