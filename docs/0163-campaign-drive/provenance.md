# 0163 — provenance

## Research claims

None. This story decodes nothing and adds no format knowledge. Every game fact it
touches is already implemented here: the generation screen and its point-buy
budget (0119, 0140, 0153), the mission start path and its compiled script census
(0155), the town screen and the mission-to-town crossing (0142, 0143), roster
carry across a won mission (0147, 0159), and the document surface (0152). The
research submodule pin is unchanged for the duration of the story
(`e1b27fd5f682d8e6f29ba3e8683b4f7cb82e7e97`).

## Ours by choice

The scenario language is this project's own design and nothing in it is derived
from the original game.

- The `create_character` step and the `character` block are authored here. They
  name the same rows the generation screen shows — `Sex`, `Class`, `Skill`,
  `Body`, `Reaction`, `Mind`, `Spirit` — because those labels are already this
  project's (`pkg/game/chargen.go`, `ChargenSetup`), not because the original
  names them so.
- The four combined pictures are ordered male fighter, male mage, female fighter,
  female mage. That order is `ui.Chargen.Forward`'s own, already shipped; this
  story reads it rather than restating it.
- The front-end `wait_until` forms (`screen`, `notice`) are authored here.
- The per-stage version table (`stages`) is a refactor of 0155's command table.
  The versions it records for existing commands are the ones 0155 assigned.
- `scripts/campaign-sweep.sh` is an instrument of this repository. Its
  per-mission tick budget is chosen here and is not a property of any mission.

## What the sweep measures, and what it does not

`unsupported` and `reached_unsupported` are properties of **this build**: how
many arms of a shipped mission's compiled script this implementation cannot run,
and how many of those a drive arrived at. Neither is a statement about the
original game. A change in either is a change in this implementation.

An outcome in the sweep table is the outcome of **an unattended drive**, which is
not play. `lost` and `undecided` are recorded and are not defects (owner,
2026-08-11).

## Open

- The sweep drives each mission in its own process with the default party. It
  does not carry a roster from one mission to the next, because most campaign
  missions are not winnable unattended, so there is nothing to carry. Campaign
  continuity is witnessed by the chaining scenario instead, on the one crossing
  that can be driven.
- `create_character` reaches the two-stage generator only when the front end
  resolved `ChargenAssets` from an install. Against a root where those assets do
  not resolve, the legacy one-stage model is what the screen holds; the step
  refuses rather than guessing, and no scenario in this repository drives that
  model.

## Nothing removed

No behaviour was withdrawn. Version-1 and version-2 scenarios keep their
vocabulary and their meaning; every file in `scenarios/` that existed before this
story is unchanged and still validates.
