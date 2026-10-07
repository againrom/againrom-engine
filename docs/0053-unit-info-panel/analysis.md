# Analysis — what a selected unit can honestly be said to be

## Intensity & terrain

**Intensity: spec-first / static**, the profile's default for rendering and UI work. The panel is a
bounded delivery whose behaviour settles at once; no other repo or team consumes it, and no watcher
tool exists here, so the sync direction is static.

**Terrain:** greenfield for the panel itself — `pkg/ui/panel.go` is new and there is no behaviour to
preserve in it. **Brownfield** for four shipped seams it widens: `terrain.UnitClass`, `LoadUnits`,
`ui.MapEntity` and the snapshot builder that fills it. Every change there is **additive** — a new
field, and one assignment that fills it — so the existing suites over those packages are the
characterisation that pins what must not move, and each is run unedited.

## What we did not know

**Whether the original's panel is decoded.** It is not, and the research says so positively rather
than by silence. `UNIT-PANEL-011` is a claim whose whole subject is what was deliberately not
established: past the first cached value the display reads by **computed index**, so a displacement
sweep returns readers that are not readers, and *which value appears at which position* cannot be
settled without the index table and the drawing loop — the interface layer, scoped out of that
round. Its own sentence is that a consumer has the value set and its arithmetic on evidence and its
**layout on nothing**. `claims/unit.md`'s open list restates it, and the promoted format spec ends
the same way.

So there is no AMBER question here. AMBER is for something research can still answer this round;
this is an unopened area with a named entry point, and the project's rule for that is a verdict:
the owner rules, the divergence is disclosed, the seam is named.

**Whether anything of the panel is decoded.** Two things are. `UNIT-PANEL-010` publishes the
25-value block the display is handed and the order it is copied in, at High for the copy map — and
with it two negatives this story leans on. It reads **no `units.reg` field at all**, so the class's
own `DescText` and `InfoPicture` do not reach the drawable by that path and the original's caption
and portrait come from somewhere else. And it has **no multi-selection arm**: one subject, one code,
one face, no loop and no container.

## What the tree actually carries — measured, not assumed

A running unit is `sim.Entity`. Its whole field set is an id, a cell, an optional target, an opaque
class key, a stall count, `HP`/`MaxHP`, and a movement domain. **Health is the only stat on it.**

The stats exist elsewhere. `data.UnitDef` resolves a definition-table row into some thirty named
columns — the four primaries, mana and its period, speed, rotation, scan range, the damage pair,
to-hit, defence, absorption, five protections, five resistances, the attack cadence, XP value. Of
those, exactly one reaches a live unit: `mapload.FromALMWith` takes the resolved `HealthMax` as both
health fields and consumes nothing else.

And the game does not call that entry point. `openMapWorld` builds every playable world through
`mapload.FromALM`, which passes **no table**, so every placed unit in the running game is born at
the provisional `SpawnHP` on both fields. The definition table is reached only by `cmd/classdump`.
Two independent gaps therefore stand between the decoded stat block and a panel: thirty columns that
no entity carries, and a table the game never loads. Neither is this story's to close, and neither
may be papered over with a number that looks decoded.

What a unit's *identity* can come from is narrower still. `data.UnitClass.DescText` is the class's
own name text, present on all 34 unit classes, held as the registry's own bytes with no character
encoding applied. `pkg/game.LoadUnits` reads that registry and keeps geometry, frames and the
animation descriptor; it drops the name. So the name exists, is loaded, and is discarded one tier
below where a panel could read it.

Drawing it needs no conversion, and that is the fact that makes this story cheap. The font indexes
**bytes**: record `k` is character `32 + k`, a string is walked one byte at a time and never
transcoded. Registry bytes handed to that walk therefore reach the records the game's own text does,
in either shipped release, and there is nothing between them for a codec to get wrong.

## What we looked at

`pkg/sim/world.go` (the entity record and the three life predicates) · `pkg/mapload/fromalm.go`
(both entry points, `SpawnHP`, and which fields a resolved definition writes) · `pkg/data/unitdef.go`
and `classes.go` (the column set, and `DescText`) · `pkg/game/units.go`, `world.go`, `frontend.go`
(what the loader keeps, what the per-tick snapshot carries, where a per-map asset is handed over) ·
`pkg/ui/command.go` and `overlay.go` (the selection, its present-and-not-dead filter, and the seam
type) · `pkg/render/text/text.go` (the pen, the box, the blit) · `internal/archtest/dag.go` (the UI
tier's grant over the render tier, which this story needs and does not widen).

Two readings were taken from that and are worth writing down because both shaped the contract. The
snapshot's `Art` field is **not** the unit's class in every state: a unit that is not alive — downed
as well as dead — crosses with the **corpse** class's art substituted whole, so a caption read off
`Art` would rename a wounded unit. And the selection's existing filter already skips a dead entry
while keeping a downed one, so "what the panel says about a corpse" is answered by a rule that is
already there rather than by a new one.
