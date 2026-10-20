# SAV continuation supplement

`/CurrentState/AgainromActions` is the Againrom continuation supplement in the
ordinary application registry. `pkg/formats/sav/native_actions.go` checks its
exact path, unique kind-6 leaf, framing version 1, byte count, 8 MiB limit and
zero padding. The payload is JSON. `pkg/game/savactioncurrent.go` checks payload
`Version` 1 and the document-dependent continuation fields. The reader rejects
unknown JSON fields and trailing values.

## Writer carrier record

`WriterCarriers` is an unsigned 32-bit JSON field in the supplement. It records
which original carriers the writer built from its World. The field is metadata;
it neither creates a carrier at LOAD nor changes World state. A missing field
has the deterministic value zero, meaning that the writer held none of the
listed carriers. The current writer emits the field explicitly. It records
`"WriterCarriers":1024` (bit 10, TerrainBlockPlane) when the saved World holds
the cell planes, which every first-game World does from mission start, and on a
town SAVE of an install whose Worlds hold them (the installed `map.reg` cost
table). A second-game SAVE, whose Worlds hold no planes (DIV-3095), records 0.
Bits 0..9 stay clear (DIV-2885, `docs/1482/story.md`).

Production LOAD rejects an ambiguous record with
`current actions: invalid WriterCarriers record` and a bounded reason. The
reader counts top-level members after JSON name unescaping and case folding.
More than one occurrence is invalid, including equal values, different letter
case and escaped names. A present member must be one nonnegative decimal
integer JSON token within `uint32` and the known mask. Explicit `null`, strings,
fractions, exponent notation, negative numbers and unknown bits are invalid.
Nested members and string content do not supply the root record. Absence keeps
the historical zero default. Ambiguous input never selects the weaker M2 rule.

The bits are stable wire identifiers. A carrier implementation sets its bit
only after its writer builds that complete carrier from state. Its story must
name the bit and prove the corresponding original-carrier byte instrument.
Loading a carrier from a source file, retaining opaque document bytes or
restoring supplementary current policy alone does not establish this writer
capability.

| Bit | Mask | Original-carrier instrument |
| --- | --- | --- |
| 0 | `0x001` | Buildings |
| 1 | `0x002` | CellRecords |
| 2 | `0x004` | DeadRoots |
| 3 | `0x008` | DyingOwnerActors |
| 4 | `0x010` | Groups |
| 5 | `0x020` | MoverRoutes |
| 6 | `0x040` | Players |
| 7 | `0x080` | Sacks |
| 8 | `0x100` | SessionBlock |
| 9 | `0x200` | SpellEffects |
| 10 | `0x400` | TerrainBlockPlane |

The known mask is `0x7ff`. Any bit outside that mask rejects the supplement;
it cannot silently weaken admission. Future carriers may extend the mask with
new stable bits. Reusing an existing bit for a different carrier is invalid.

## Milestone-2 admission

An original save has no checked continuation supplement and retains every
original-carrier instrument. An Againrom save is admitted to the writer rule
only after the framing, JSON, version, carrier mask and document-dependent
supplement checks succeed. A malformed supplement is an error, never a reason
to treat the file as an original save or to skip an instrument.

For an admitted Againrom save, LOAD, SAVE and cold LOAD must preserve the World
hash and the document bytes its writer owns; following ticks must run. Each
listed original-carrier byte instrument additionally applies when its
`WriterCarriers` bit is set. File names, paths, dates and known failures do not
select the rule. Supplement version 1 alone does not claim any carrier.

The owned document bytes are the ordered campaign Valuable Documents pairs
and the complete DIV-2342 payload envelope. They are checked before SAVE and
after cold LOAD. The M2 writer witness saves once at LOAD and once after 60
ticks, compares the World hash at each cold LOAD, and requires both Worlds
to advance through 60 following ticks. Following hash differences are reported
with their first tick; carrier continuation equivalence is not inferred from
this admission rule. Between-mission saves have no World and instead compare
their town state and owned document bytes after cold LOAD. The writer census
also changes the World through its existing route before this round-trip
proof. Original-save subtest names and comparisons are unchanged.

`AGAINROM_M2_ORIGINAL_CONTROL=1` forces the stronger original-carrier path in
the tagged tests. The loss controls copy the admitted input under a new name
and use this path to prove that the byte instruments still detect its
differences. This negative control pins its source bytes by SHA-256; admission
does not read that digest. Other zero-mask saves need not share its loss shape.
The control cannot weaken admission. This is a harness rule; it changes no
original behaviour and needs no new divergence row.

The `ARSV` campaign-document payload of DIV-2342 has a separate version, exact
length and CRC-32 envelope. Its bytes have no carrier-capability schema. Its
presence does not admit a save to the writer rule.

## Compatibility

The new field keeps the existing JSON version and all existing field names.
Historical supplements without it read as zero. Older engines that reject
unknown JSON fields cannot LOAD a new supplement containing `WriterCarriers`.
The field is not part of a gob-persisted struct and adds no World wire field.
