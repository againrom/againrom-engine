# Tasks — the mission trigger runtime

## T1 the type-7 leaf grammar

Add the type-7 decode to `pkg/formats/alm`: the three counted arrays, the 796-byte node and the
184-byte trigger, as a method on the decoded map. Expose the placed-unit record's two identifier
words beside the fields already there (FR-1; DD-1).

The raw body stays exactly as it is — decode THROUGH it, never instead of it — and the walk must
consume the payload exactly. The ten 64-byte parameter names are read by nobody; leave them out and
fill them in the fixture so a reader of them would be caught.

Tests: the arrays field for field with a parameter in the tenth slot and none in the fourth; five
refusals; the absent and the present-empty record told apart; the body unmoved; the two identifier
words at their own offsets and widths (SC-1).

## T2 the compiled script, its state and the pass

Add the compiled program and the runtime to `pkg/sim`: the three record types, `NewScript` with its
refusals and its inertness derivation, the register file, the latch array, the two counters and the
outcome, the pass and the reporter on their phases, and the byte-form section at the tail
(FR-3, FR-4, FR-5, FR-6, FR-7, FR-8; DD-2, DD-3, DD-4, DD-6, DD-7, DD-8, DD-9).

Two tables decide what this build evaluates and are the only place either answer is given. The
distance metric is one function with one call site per arm, carrying its own reason. Declare the new
fields in the field-set pin and the new methods in the method-set pin.

Every pinned form, length and digest in the tree moves. **Re-derive, do not re-record**: strip the
new section, confirm it is zero, lower the version byte, and require the result to reproduce the
number already pinned.

Tests: the phase and the ordering; the alphabet, both AND arms and the AND of nothing; both flags
over three passes; the register file's two id spaces, the constant preset and the bounded subscript;
the outcome table including the counter that skips one; every implemented check arm; the inert
trigger and the skipped instant; the round trip and the section's refusals; and the campaign's first
mission's win chain driven to a win (SC-2, SC-3, SC-4, SC-5).

## T3 the binder

Add the compile to `pkg/mapload`: three passes in the builder's own order, plain parameters packed in
encounter order, the three `Target_Unit` bands through an injected resolver, the id-to-subscript miss
value, the dropped trigger, the latch by map position, and a report carrying the drop table, the
dropped triggers, the discarded build-time actions and every unresolved reference
(FR-2; DD-1, DD-5).

Two entry points, one over a map and one over a decoded script, so the passes are testable without
routing a fixture through the reader. A node whose reference does not resolve is still built and
still takes its register — report it, do not drop it.

Tests: the three passes; encounter-order packing against a node whose parameters sit in slots 0, 1, 2
and 9; each band, resolved or reported; the miss value and the one case it cannot be taken; the unit
table's duplicate rule; the loudness rule on a compiled map; and one whole road from map bytes
(SC-6).

## T4 the developer verb

Add `almtool script`: the decode counts, the compile counts, the binder's findings, the unimplemented
arms tallied per opcode and the inert triggers with their map positions (FR-8; DD-10).

It resolves the map's own units and no hero — a campaign map places none — so a hero-band reference
prints as unresolved with the band that explains it. Register the verb in the usage line and the
command doc.
