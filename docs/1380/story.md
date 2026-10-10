# One random service

## Intent

Every random draw in the game comes from one service, `pkg/random`. A session
replays from one seed: the same seed gives the same frames, sounds, World and
SAV. A launch setting switches the original game's generator on: every
consumer the original runs on its one main-thread stream then draws from one
shared MSVC stream, in the original's draw form. Seeded mode is the default
and changes nothing pinned.

Base: `458e07e2` (game 0.105.0). Knowledge pin: k208, moved to k217.

## Authority

- Owner: everything runs deterministically from one seed; the original
  randomizer can be switched on without breakage.
- ROM1: `SESS-082` (one main-thread stream), `SESS-083` (seed 1, the
  item-star grids take draws 1..8192, the four reseed sites), `MAGIC-279`
  (the recurrence), `MAGIC-283` (placement on the main thread), `MAGIC-284`
  (the range and float wrappers), `MAGIC-285` (each family's form),
  `MAGIC-286` (the music list draws).
- `MAGIC-283` places 47 sites by a direct path (High) and the rest by
  exclusion (Medium). Medium placement by exclusion is enough for a consumer
  to join the shared stream.
- Divergence rows updated to state both modes: DIV-027, DIV-076, DIV-370,
  DIV-497, DIV-789, DIV-790, DIV-835, DIV-843, DIV-851, DIV-859, DIV-869,
  DIV-1022, DIV-1202, DIV-1295, DIV-1942, DIV-2038, DIV-2681. New rows: DIV-2730..2736.
  DIV-2737..2739 are unused.

## As built

`pkg/random` holds the session service, SplitMix64, the MSVC recurrence
(`Rand`, the range wrapper `Range` with no draw at width zero, `RangeFrom1`,
the float wrapper `Float`), a generic LCG step for a description that carries
its own constants, and `Go`, the existing seeded source that keeps seeded
outputs identical. It imports no clock and no `os`.

A `Session` is a seed, a mode and the shared stream's state. The front end
holds one `Service` in `RuntimeServices.Random`, replacing `MusicSeed` and
`AmbientSeed`. The session seed comes from `-seed <n>`, else the clock at
new game (`clockSessionSeed`, the one sanctioned clock read). A runtime built
without launch settings uses seed 0, so a test replays without a clock.
`-original-random` selects original mode.

### Streams

| Stream | Consumer | Seeded mode | Original mode |
|---|---|---|---|
| world | the mission World | SplitMix64 at the World seed, unchanged | the shared stream; the World holds it while the mission runs; range wrapper for every bounded draw |
| placement | mission start | `StartSeed`, unchanged | the shared stream after the mission loader's and AI manager's reseeds; continues into the World |
| command-voice | speaker chooser, command and select readers | (seed, name) | shared, raw draws |
| music | list order | (seed, name), the engine's shuffle | shared: start `rand()%n`, then 2n swaps in Random Order (`MAGIC-286`) |
| ambient-birds | mission bird calls | (seed, name) | own stream (DIV-2732) |
| town-animation | sign, vane, square draws | (seed, name) | shared, `n*rand()/32767 % n` (DIV-2731) |
| town-ambient | square bird group, count, delay | (seed, name) | shared, same form |
| town-wildlife | square families | the description's generator at the stream's seed | shared, raw draws |
| tavern | tender delays | (seed, name) | shared, same form as town-animation |
| shop-interior | merchant idle | (seed, name) | shared, same form |
| school | training draws | (seed, name) | shared, same form |
| bolt-figures | bolt, Heal and Drain figures | per-call seed, unchanged | own per-call stream (DIV-2733) |
| item-stars | the four star grids | state zero, unchanged | draws 1..8192 of seed 1 (`SESS-083`) |
| shop-stock | generation and restock | campaign seed, unchanged | shared, range wrapper |

A seeded stream's seed is a mix of the session seed and the stream's name. A
new session restarts every stream in place; a room's entry no longer reseeds.
`SetStreamSeed` is the test injection point beside the `RuntimeServices`
draw seams.

### Original mode

The shared stream starts at 1; the item-star grids take its first 8192
draws; sound initialisation reseeds it at launch, the scenario constructor at
new game, the mission loader and the AI manager at a mission start and a
LOAD. Each reseed value is the mix of the session seed, the site and the state
it replaces (DIV-2730). Between missions the service holds the stream; while
a mission runs its World holds it (`World.OriginalRand`), and the service
takes the state back when the mission ends. The World byte form marks the
mode in byte 29 (bit 0x40) and keeps the 32-bit state in the rng field.

### SAV

A SAV carries `/CurrentState/AgainromSeed` (DIV-2736): version 1, the seed,
the mode and the shared state. The zero session (seed 0, seeded, state 0),
the session of a runtime built without launch settings, writes no leaf, so
every SAV hash pinned before this story holds. A LOAD begins the saved seed
in the launch mode. An engine SAV without the leaf (it carries the
current-session supplement) begins the zero session; any other SAV without
the leaf takes `SeedOf` its bytes. A SAV of the
other mode loads: a seeded state is folded into an original one, an original
state unfolded into a seeded one. A LOAD is never refused for it.

### Guard

`internal/archtest` (`randomness.go`) forbids, in the production files of
`pkg/sim`, `pkg/mapload`, `pkg/game`, `pkg/ui` and `pkg/town`: a `math/rand`
import, the original generator's constants, a seed or session argument to a
`pkg/random` call that reads package `time`, and a seed function that reads
the clock other than `clockSessionSeed`. The debt list is empty.
`pkg/town` stays standard-library only; its description LCG takes its
constants from data.

## Proof

- Seeded pins: no pinned literal moved. The town square and room traces and
  the profile witness (`rom1-en`, `rom1-ru`, `rom2-en`, `rom2-ru`) pass
  unchanged on their installs: the zero session writes no session leaf.
- Determinism (`TestReleaseRandomSessionReplaysFromItsSeed`, EN and RU): one
  headless session from launch seed 7 covers the town square, tavern, shop and
  school with idle extras, Random Order music, a new game on mission 41, a
  selection and command voice, a Lightning bolt, a Meteor Storm, SAVE, LOAD
  and 120 further ticks. Two runs give 1599 identical lines of frame hash,
  sound log and World hash, and identical SAV bytes. The World hash is
  equal at SAVE and after LOAD. Seed 8 changes 629 presentation lines on
  EN and leaves every World hash equal.
- Original mode (`pkg/random` tests; `TestReleaseOriginalRandomSession` and
  `TestReleaseOriginalRandomMissionOne`, EN and RU):
  - from seed 1 the first ten MSVC outputs are 41, 18467, 6334, 26500, 19169,
    15724, 11478, 29358, 26962, 24464;
  - the item-star grids hold draws 1..8192 of seed 1;
  - a zero-width range draws nothing;
  - two consumers interleave on one stream, and a running mission takes the
    draws;
  - the SAV leaf records mode 1 and seed 7, and a LOAD continues the saved
    state through the load reseeds;
  - SAVs of either mode load in the other;
  - the first mission runs 2000 ticks with no refusal or panic, also on both
    ROM2 roots.
- Guard: `internal/archtest` `TestLiveRandomnessMatchesItsDebt` with an empty
  debt list; synthetic cases fail on each banned form.
- Gates on the merge: `gofmt -l` clean; `go test -trimpath -count=1 ./...`
  ok; release set ok on RU and EN (926 of 972 ran, 46 lacked a subject);
  headless scenarios 54 of 54 on EN and RU; `check-no-game-assets.sh`
  clean; `check-div-claims.sh` exit 0. `check-milestone2-acceptance.sh`
  exits 1 on both roots with the same 13 failing instruments per root as
  current main, with equal summary counts; the writer census names
  `AgainromSeed` as engine debt.

## Open debt

- DIV-2731: town and room draws use one scaled form in original mode.
- DIV-2732: mission ambient birds keep their own stream.
- DIV-2733: bolt, Heal and Drain figures keep their own streams.
- DIV-2734: the music stop's order build and the setter path.
- DIV-2735: the hall-of-fame fallback's nine draws.
- The original's runtime order of consumers between reseeds is Unknown
  (`SESS-083`); the engine's order is its own.
