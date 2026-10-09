# Town square built by one town composer

## Intent

The ROM1 town square is built by a data-driven town composer from an embedded
ROM1 town description. Nothing the player sees, hears or saves changes. The
composer is the one builder for town screens (owner): later ROM2 towns and a
mod town are further descriptions for it, not further code. The rooms behind
the square (tavern, shop, school, world map) stay the existing room pages and
are reached through a named page and campaign hooks; a later story moves them.

Base: `5d22d8cc` (game 0.102.0), merged with `51cccab7` (game 0.104.0).
Knowledge pin: k208.

## Authority

- Owner: one composer with no per-town branch; one builder per kind; a ROM2
  town is not the ROM1 town, so the composer holds no game's facts.
- ROM1 values: every value of `pkg/game/towns/rom1.json` carries the claim IDs
  and divergence rows the replaced code cited (TOWN-*, AI-RAND-058,
  SHOP-*, SAV-CAMPAIGN-081, VIDEO-MUSIC-065, TEXT-HOVERROOM-051 and the
  DIV rows of `docs/divergences/town.md`), or "owner" where the replaced code
  carried an owner direction. No new ROM1 fact is asserted.

## As built

- `pkg/town` imports only the standard library. It decodes a description
  strictly (unknown fields and trailing data refused), validates every
  reference (art, actor, hotspot, slot, random source, phase, room, step
  verb) with a named error, loads art through a `Loader`, and runs a `View`
  over a `Host` that supplies art, the clock, host draws, the generator seed,
  conditions, sound and loop requests, and named campaign hooks.
- Kinds built: view; layer (copy or over, with `when` over hover, an active
  actor or a condition); mask map (byte to hotspot); hotspot action (room,
  menu, hook, conditional `if ... then ... else`); hover reaction (sound
  latch, stop set, arm with chance, drive); episode (rewind or hold end, with
  a tail counter); loop; ping-pong (`pendulum`, shared enable); stepper (rest,
  hold, hold-unless condition, toward-first on hover); `driven` (direction
  set by pointer and hover); scheduled family with position table and
  unequal pick (`families`); route as group-and-prefix (`flock`); clock
  (period, strict or inclusive compare, paint-advanced, first-paint stamp);
  random pick (host source or LCG, `scaled` and `masked` forms); condition;
  sound cue (slots, latched one-shot, count cues, frame cues, entry loop);
  music; tip.
- `Process` keeps the state that outlives a view: the paint clock, the
  generator, the flock wait latch and the star tail counter. A game keeps one
  on `Presentation`, replacing the process-lifetime fields the square code
  kept on the town screen and the presentation latches.
- `pkg/game/townsquare.go` embeds the description, implements the host, the
  art loader, the conditions `gate-open` and `save-admitted`, and the hook
  bodies. The hooks are the bodies of the replaced square and room-entry code,
  run in the description's order: `dialogue-reset`, `release-audio`,
  `set-room`, `talk-clear`, `tip`, `tavern-leave`, `shop-reset`,
  `school-leave`, `tavern-interior`, `tavern-detail-clear`, `faces`,
  `tavern-selection`, `shop-shelf`, `shop-interior`, `offer-on-entry`,
  `school-reset`, `navigate`, `inn-queue-commit`, `trade-cancel` and
  `gate-closed`. Room entry and exit run as ordered step lists in the
  description, so the S19/S20 order is data.
- The town screen builds its square view when its services are bound, so a
  reader such as a refused LOAD's save check leaves the screen unchanged.
- `pkg/ui` no longer knows the square: `TownSquareView` carries a
  `TownSquareScene` (size, paint, control and tip at a point), and the ui
  draws the message line and the tip panel over it. The music request takes
  an optional track from the game.
- Deleted: `pkg/game/townexterior.go`, `townfamilies.go`,
  `townexteriorart.go`, `townsquareart.go`, the square painter, art types,
  mask table and family tables in `pkg/ui/townsquare.go`, and the ui tests of
  that painter. There is no fallback path.
- DIV-1940's implemented-behaviour cell now names the description and the
  composer instead of the deleted view-origin helper. DIV-2688..2691 are
  unused.

## Format

JSON, decoded by `encoding/json` with unknown fields refused, then validated.
Reasons: the standard library reads it on every target without a dependency;
a mod loader reads the same format; `DisallowUnknownFields` turns a misspelt
key into a named load error instead of a silent default; a cite is an
ordinary field, so a test walks every object. The format has no expression
language: `when`, `if`, chance and pick are fixed objects. An expression
language waits for a description that needs one.

## Proof

- `TestReleaseTownSquareTraceIsUnchanged` (`pkg/game`), committed before the
  move on unchanged code: 1005 ticks per root of frame hash, sound and music
  requests, room, actions and SAV hashes over a scripted pointer path across
  every hotspot and each room entry and return. After the move it is
  identical on EN and RU; the recorded files are unchanged.
- `pkg/town` `TestComposerHoldsNoGameFact`: the non-test source imports only
  the standard library and holds no integer literal outside
  {0,1,2,3,4,8,10,16,32,64}, no float or rune literal, and no string literal
  with a dot, backslash or slash other than the entry-name formats.
- `pkg/game` `TestTownDescriptionEveryValueIsCited`: every object that states
  a value cites; each cite is a claim heading in the pinned `knowledge/`, a
  ledger row, or "owner" (187 objects, 305 cites).
- `pkg/game` `TestTownSquareSceneControlsFromTheMaskBytes` and
  `TestTownSquareLabelsCopyAtTheirOrigins` replace the deleted ui tests of
  the mask table and label origins.
- The existing town, tavern, shop, school, city, gate and world-map release
  tests, the seven town headless scenarios and the milestone-2 acceptance run
  unchanged in their assertions; the unit tests that read square state now
  read it through the composer's actors.

## Open debt

- `ui.TownTipRect` and the ui music scene table still hold the square's tip
  rectangle and track list; the description also carries both. The ui values
  go when the room pages move.
- The trace witnesses campaign hooks through their effects (room, sounds,
  SAV hash), not by hook name, because it had to run on the code before hooks
  existed.
- Room pages, their state graphs and their clocks are the next story.
