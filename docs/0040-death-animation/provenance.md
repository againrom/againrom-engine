# Provenance — the death animation

Pin: research `f35be34`; every claim id below was read there.

## Backing

| Spec anchor | Source | Confidence | What it carries |
|---|---|---|---|
| FR-1 (the dying block's slot length; an absent phase is no block) | `SPR256-UNIT-024`, `REG-UNITS-049` | High | the six blocks and their bases as arithmetic over the class's own phase scalars, the layout pair resolved from the class's switch, bone and idle sharing one base; and the per-key no-parent default `-1`, a declaration of absence rather than a count. Both are already carried by `pkg/data`, so this story only names a length that arithmetic computes with. |
| FR-2 (the corpse is another class's sheet, one hop) | `REG-UNITS-050` | High | the dying key is a class id, and the corpse and dying arms **replace the class**: they subscript the class table with it — once — and read that class's own layout switch and phase scalars in place of the unit's own. The corpus adds that every shipped class names one, every target resolves and every target has a positive dying phase count, so the guarded cases are ours to be total about, not observed. |
| FR-3 (the frame is the run's clock halved, two ticks each, the last one then held) | `TERR-SPR-047`, `ANIM-DEATH-007`, `ANIM-RUN-004`, `REG-UNITS-050` | High | the dying arm computes `dir * DyingPhases + clock/2` from the substituted class, the halving a named instruction; the run is sized `2 x DyingPhases` ticks from the art, exactly what that halving needs to play the block once; and when its counter reaches zero no arm runs and the draw's own corpse fork freezes on dying frame `DyingPhases - 1`. |
| FR-3 (the direction rule and the mirror) | `ANIM-DIR-006`, `REG-UNITS-051` | High | every block but the standing one halves the sixteen-way facing to eight, and the layout switch mirrors the upper half. Already carried, and shared verbatim with the live selection. |
| FR-4 (nothing on the drawing side advances a stage; the run's clock starts at the transition) | `ANIM-DEATH-007` | High | the stage is one byte the server ships, the client dispatches on the transition and advances nothing, and the run it starts there begins at clock zero. The whole-image writer census is the instrument, and its blind spot — a store through a computed pointer — is stated in the claim. |
| FR-5, FR-7 (a body holds still, facing where it died) | `HERO-DEATH-026` | High | the actor's death arm stops its movement and releases its reservation, and nothing turns it afterwards. Our tree gets this free — the simulation does not advance an entity that is not alive — so the requirement pins behaviour rather than adding it. |
| FR-8 (animation is not simulation state) | `ANIM-CLOCK-001` | High | no simulation actor carries a frame, a phase or an animation state, and the drawable that does is absent from the serializable-class table. The consumer's consequence is quoted there: the frame need not be deterministic, but must be stepped by the simulation's own tick. |

## Ours by choice

| What | Why it is ours, and what would make it the game's |
|---|---|
| **The downed state takes the fall too.** | The engine has no state between alive and dead: its death arm triggers on health at or below zero, so what we call downed is what it calls dead. Drawing both through the death path agrees with the engine on the health it can see. The downed state itself is ours. |
| **The death clock is the map screen's scene clock**, and the elapsed count carries no per-entity offset. | The live selection de-syncs entities by their id so a crowd does not march in step. A fall must start at its own first frame, so that offset is deliberately not applied here. The pin speaks to neither. |
| **The clock is set once and never cleared.** | Our simulation has no heal, and damage refuses an entity already dead, so nothing returns from not-alive. A revival rule would need a transition that does not exist to be defined against. |
| **Which class's canvas the anchor is measured in.** | `REG-UNITS-050` enumerates what the corpse arm reads out of the substituted class and the canvas fields are not in that list; `TERR-SPR-067` gives the anchor formula without saying which class object it reads from. We draw the corpse **wholly** as the substituted class — sheet, layout and canvas — one substitution rather than two half ones. Reading the arm's own field accesses would settle it. |
| **The fallback when the death path yields no frame.** | The engine's corpse arm falls back to the standing frame when the substituted class carries no bone block. We fall back to the live selection — that intent one step wider — which keeps a body on the field for a class with no dying block at all. |

## Open

| What is not decided here | Why it is left open |
|---|---|
| **Decay stages 2 to 5** — the bone frames and the vanish. | Decoded (`ANIM-DEATH-007`, `REG-UNITS-050`, `HERO-DEATH-026`) and deliberately not built: the stage is produced by a dying countdown and a health that walks downward on a schedule, which is simulation state and would be hashed. Our world can only be in the first stage. |
| **The replay of the last two frames** when a body is struck again. | The same reason: a stage arriving a second time drives it, and no stage arrives here at all. |
| **The classes that leave no corpse.** | Decoded as a movement-type column of a placeable-definition database this tree does not decode (`ANIM-DEATH-007`, `TERR-MOVE-057`). Nothing here can key on it and nothing guesses. |
| **One shipped class whose bone phase count is the absent sentinel**, whose bone index walks backwards into the dying block. | Graded Medium in the pin, and a hazard of the bone arm alone. Recorded for whoever builds it; this story's frames are proved to lie inside the dying block, which excludes that shape here. |
| **Corpse removal**, and the id release at the last stage (`MOVE-ID-016`). | Out of scope by C-1, and against the simulation's own contract, which keeps a dead unit and its id. |

## Removed

| Statement | Why it is not here |
|---|---|
| A dying and bone block layout summed from separate 16-way and 8-way direction counts, with a bone base of its own. | Superseded by the decoded layout this tree already implements, where the pair is resolved from the class's own switch and bone and idle share one base. An older baseline of our own carried it as an unresolved research item, and the pin answers that item against it. |
| `BonePhases >= 3` as the rule deciding whether a class has a bone chain. | The pin measures it as a corpus fact on 261 of 262 pairs, not a cutoff the engine applies — the engine's own guard is a different test. Nothing here consumes it. |
| Named tuning constants for the fall and the rot, and a retuning of them. | The cadence is decoded: two ticks per dying frame. A knob would be a number to defend against one we have. |
| A chain walk over the dying key, with a visited set and a cycle guard. | The arm subscripts once. |
| Gating a life-state tint on whether the death frame resolved. | There is no life-state tint in this tree. |

## Appended 2026-08-01 — pin `130bb79`: `TERR-MOVE-057` is SUPERSEDED, and the one place its ordering is real is this story's

The row above cites `TERR-MOVE-057` and `ANIM-DEATH-007` for the classes that leave no corpse, and
records that nothing here can key on them. That stands. What moved is the *gloss*: `retracted.md`
now classes "2 = Ghost/Bee (`0x44`), 3 = Bat_Sonic/Dragon (`0x82`, air)" **SUPERSEDED**, because it
reads as two grades of one thing and the two domains are not a ladder — `0x44` carries the object
bit and not the terrain bit, `0x82` neither, and they disagree on **83 203 of 880 704** shipped
cells (`MOVE-DOM-026`).

Worth recording precisely here, because this is the one consumer for which the ordering **is** the
rule: the corpse test is `movementType > 1`, one of the domain byte's six consumers, and above 1 it
slams the decay counter to −1000 so no corpse is left (`MOVE-DOM-024`). So the four classes and the
`> 1` shape both survive the overturn intact. What must not be carried out of this story is the
inference that a higher code means *more* flight anywhere else.
