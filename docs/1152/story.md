# Independent SpellEffect acceptance

An original mission's SpellEffect state is checked against its source bytes on
LOAD and after ordinary menu SAVE into `.ags`, fresh FrontEnd LOAD, and twenty
continuation steps. The test distinguishes typed World members from the complete
retained Document. No save format or effect simulation rule changes.

## Authority and boundary

`SAV-DOC-053` places the u32-counted SpellEffect list after Buildings.
`SAV-STREAM-013` and `SAV-ARCHREL-253` define its shared class/object counter,
nulls and back-references. `SAV-TOKEN-034`, `SAV-EFFCHAIN-046` and
`SAV-CLASSSER-173` through `SAV-CLASSSER-175` give the Token and class bodies.
These claims were read from the pinned public knowledge snapshot.

The expected side reads decompressed bytes independently from
`World.BuildingsEnd` to the terrain start. The new
`File.SpellEffectArchiveLocation` exposes only the preceding archive cursor,
class indices and prior object body locations. Earlier actor/Building framing
and container decompression remain trusted. No expected count, field or edge
comes from `File.SpellEffects`, decoded Record fields, the Document decoder, an
importer, or export over source bytes.

Typed comparison requires every represented member and nullable reference site.
Document comparison requires all Token fields, class members and a bijection of
ordered roots and named references. It distinguishes a shared child from two
distinct objects with identical values, without equating native local indices
with original archive indices. Cross-links from earlier, unrelated ownership
graphs are outside this rooted graph comparison.

## Proof

`TestMilestone2SpellEffects`, selected by the milestone-2 gate, discovers all
preserved `.sav` paths. It counts raw sources before LOAD and names each refused
file separately. Focused EN/RU runs found 102 paths: 62 world documents, 39 city
documents and one unreadable input. All 62 world documents resumed with zero
mismatches or refusals. Three source lists are nonempty: three roots, eight
objects, seven reference slots and zero alias edges; 59 lists are empty. The
393 source field bytes comprise 97 typed member bytes and 296 Token bytes
retained only in Document. Class totals are two SpellTransport, two PointEffect,
two Effect_DirectDamage, one AreaEffect, one Effect and zero bare SpellEffect.
These are discovered-path counts, not a claim of original-runtime provenance or
SHA-distinct population. The gate prints every nonempty source path.

The registered, untagged `TestReleaseMilestone2SpellEffects1152` uses the
SHA-pinned `game0018.sav` identified by `SAV-EFFECTGRAPH-366`. Its one root and
three objects occupy a 220-byte counted span. Forty-one member bytes reach the
typed World; all 152 field bytes, including 111 Token bytes, reach the retained
Document. Both original LOAD doors, ordinary menu SAVE, fresh FrontEnd LOAD and
twenty paired continuation hashes pass on EN and RU.

Literal synthetic controls cover all six classes, root aliases, a shared child,
distinct equal-valued children, prior-object references, nulls, dropped roots,
lost or changed scalar/raw fields, split sharing, collapsed identities, corrupt
counts/tags and every truncated prefix of the constructed list. Both DAG forms
also pass both LOAD doors, ordinary menu SAVE/fresh LOAD and twenty steps.

## Open debt

`DIV-938` keeps PE44 as an opaque source key; `SAV-CLASSSER-174` leaves its
repaired simulation meaning Unknown. `DIV-939` records typed alias expansion
and the absence of an effect driver. Token bytes and DAG identities already
survive in the complete Document, which this story now checks directly. World
hashes cover the typed projection; they do not substitute for the Document
comparison. No AreaEffect tick, transport delivery, original-runtime write or
original-world SAV writer is introduced. New divergence IDs are unnecessary.
