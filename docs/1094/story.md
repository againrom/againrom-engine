# Original mission actor pools

Loading an original mission save must retain a wounded non-party map actor's
saved HP, maximum HP, mana and maximum mana before the first tick. Fresh map
construction and equipment rearming must not overwrite those four values.

## Scope and authority

- Read every Player's actor lists by bounded archive traversal. Null references
  add no actor; repeated references add no second actor. Do not use the head scan
  to read statistics or choose a target.
- Restore only stage-zero actors with positive signed health, an on-map saved
  cell, and a nonzero MapUnitID that uniquely names a living, on-map, non-party
  entity. Duplicate eligible source records or target IDs refuse the load.
- Keep unmatched, dynamic, off-map and dead populations outside this projection.
  Count exclusions rather than refusing an ordinary save for their presence.
  Existing party restoration remains the sole owner of party pools.
- Apply four values after materialization and Rearm, inside the existing
  prepare/commit boundary. Validate the whole batch before writing any pool.
  Native mission SAVE remains `.ags` and must roundtrip the world hash without
  a byte-form version change.

The pinned research is `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`.
`SAV-UNITFLD-049` names the four words; `SAV-ROSTER-024`, `SAV-DOC-053`
and `SAV-UNITPROG-156` establish the Player graph and record programme.
`SAV-ID-015` distinguishes MapUnitID from runtime ID. Amended `SAV-DEATH-051`
and `SAV-DEADLOAD-124` distinguish living owner lists from the exact corpse
list. `SAV-HUMRUN-444` keeps the adjacent raw runs outside this work.

The unique authored-map join is this bounded importer, not ROM1's object-graph
loader. A matched record with HP above maximum HP or mana above maximum mana is
unsupported and refuses the load without clamping. This is importer policy,
not a claim that ROM1 cannot write such a state. Health uses signed-word
interpretation for the living test; the other three words retain their u16
magnitudes. These boundaries are recorded in DIV-634.

## Proof

Independent synthetic archive bytes cover all Players, sparse and repeated
references, four distinguishable pool values, duplicate identities, unmatched
and excluded actors, malformed/truncated records, and old-session retention.
Production App LOAD and SAVE/LOAD cover published values and native world hash.
A lawful source with literal raw values is imported against both EN and RU
roots. Focused tests, the asset-free suite and asset guard are recorded in
`verification.md`.

## Open debt

Corpses, actor creation/removal, saved modifiers, orders, effects, spellbooks,
regeneration periods, and other raw runtime fields are not imported. Existing
position and stock joins retain their previous boundaries. This is not a
fresh-world original writer and does not widen the city `.sav` writer gate.
The exact reader supports null/repeated top-level Player references, but the
existing `sav.Open` envelope scan still requires the declared Player count to
equal its located distinct Player bodies. Such top-level envelopes remain
unsupported end to end; sparse/repeated group actor references do work through
the normal LOAD route. Replacing that envelope index is separate codec work.
