# Provenance — a dialog window stops the world

## Backing

| spec anchor | source | confidence |
|---|---|---|
| FR-1, FR-6 — an open dialog window pauses the world and darkens what is behind it | the owner, as this product's **author**, ruling on what we ship | Authored — not a claim about the original, and not offered as one |
| FR-4 — the rule belongs to *a notice is open*, not to a notice kind | the same ruling: it was given for **any** dialog window | Authored |
| FR-2 — a resume that pays back the suspended span would burst | measured in this tree, `analysis.md` | High — the two paths were driven over the same span and counted |

Only three rows, and that is the shape of this story rather than a gap in it: the contract is an
authored ruling, so almost everything below is in *Ours by choice* by construction. The one row
that is not authored is a measurement of our own code, which is why it carries a confidence at all.

## Ours by choice

| what the spec fixes | why it is ours |
|---|---|
| That the suspension is expressed as the **stop the player's own pause already uses** | Nothing decoded says how a window and a clock are related in the original. Ours is an engineering choice between two mechanisms with different observable consequences, and it was made on a measurement rather than on a preference — see `analysis.md`. |
| The dim's colour and its strength | Nothing states either. The value shipped is black at half opacity, chosen to match the only other notion of *dimmed* this tree has — the engine margin's half-brightness — so the build has one answer to "how dark is dim" rather than two. |
| That the dim covers the map and not the panel or the readout | Nothing states what a backdrop covers. Ours: those two boxes are the front-end's own instruments rather than the mission's picture, and an instrument that dimmed with the world would be unreadable exactly when a player wanted to read it. |
| That the suspension is enforced **per advance**, so the advance that raised a notice runs out its own elapsed span | Nothing states it, and nothing could: the granularity is a property of how this front-end asks for world time, which the original does not have. It is measured rather than assumed — up to one catch-up span of world time, zero at the shipped cadence — and disclosed because no frame shows it. |
| That the three cadence keys are inert while a notice is open | Nothing states it. Ours, and it is a decision against the shipped behaviour rather than an absence — before this story all three were live under a notice. |
| That the water keeps animating while a notice suspends the world | Inherited rather than chosen here: the stop reaches the seam and not the viewer, which is a shipped rule of this tree's own pause. The original's pause is not built that way — see *Open*. |

## Open — deliberately assigned no meaning

| what | why it is left open |
|---|---|
| Whether the original stops its clock while a dialog window is up | `SESS-PACE-018` establishes that the engine's idle handler has a **run bit** whose clear branch issues neither the simulation tick nor the animation tick, and names that branch as the pause. It does **not** name what writes the bit — the claim marks that half Medium precisely because the writers were not enumerated — so nothing published connects a dialog window to it. This story asserts nothing about the original either way, and the seam is placed so a later answer moves a value rather than a structure. |
| Whether the original darkens the backdrop, and by how much | No claim states it. `DLG-WIN-001` fixes the window's own rectangles and its children; the window's art is shipped bitmap this project does not read, and nothing decoded describes what happens to the pixels outside it. |
| Whether the original's pause also stops the water | The run bit's clear branch omits the animation tick as well as the simulation one, so the original's pause plausibly holds both. This tree's does not, and the divergence predates this story and is not closed by it. |

## Removed

Nothing was dropped from this contract. One statement in **shipped code** is contradicted rather
than removed, and it is named here because it reads as a decided design point rather than an
accident: the map arm's note that "the world keeps running behind the box, notice or none". It was
0066's deliberate choice and it is the choice this story reverses on the owner's ruling; the comment
is corrected in the same commit, not left to be read as still true.
