# 0066 — tasks

**Kinds:** `impl` — one implementation commit each. There are no other kinds in this story;
evidence and the build are pipeline stages and carry no entry.

## T1 the glyph rule — impl

Give `text.Font` a selector, convert every drawn byte through it, and discover the selector from
the install.

Files: `pkg/render/text/text.go`, `pkg/render/text/*_test.go`, `pkg/game/font.go`,
`pkg/game/frontend.go`, `pkg/game/font_test.go`.

Covers: FR-9, DD-4, DD-5.

Fences: `pkg/render/text` imports nothing inside the module and nothing outside the standard
library — the selector is a plain integer field and the discovery of it does not live here. Do not
remove or loosen the existing subscript bound. Do not touch the panel or readout resolvers.

Done when: `Convert` is exported and total over all 256 bytes at every selector; a `Font` whose
selector is 0 draws and measures exactly what it drew and measured before, with the package's
existing tests unedited and green; the front end sets the selector it read on the font it loads;
an install with no selector entry, and one whose entry ends in no digit, both yield 0.

## T2 the event text — impl

Compose the address, read the entry, and resolve one part out of the payload.

Files: `pkg/game/eventtext.go`, `pkg/game/eventtext_test.go`.

Covers: FR-3, FR-4, FR-5, DD-6, DD-7.

Fences: no error return anywhere on this path and no logging — absence is a boolean. Read nothing
at map load. Handle no markup tag but the part tag; leave every other literal alone rather than
stripping or interpreting it.

Done when: the address is composed with the event number two-digit; a missing entry returns the
not-found answer with no error; part lookup finds part 1 and a later part, returns nothing for a
payload with no matching tag, takes the first matching tag in file order, and returns the part-10
body for a payload whose first tag is `<part=10>` when part 1 is asked for; a part that runs to the
end of the payload with no following tag is returned whole.

## T3 the raises — impl

Derive, from a world and its compiled script, which announcements a pass raised.

Files: `pkg/game/announce.go`, `pkg/game/announce_test.go`.

Covers: FR-6, P-1, P-2, DD-1, DD-2, DD-17.

Fences: **no file under `pkg/sim` may be modified by this task or any other in this story.** Read
the world only through its existing exported surface. Hold no simulation type in anything that
crosses out of `pkg/game`. Do not read the win and lose counters.

Done when: over a driven world the raises equal the raise-message parameters of the triggers that
fired, for a one-shot trigger and for a repeating one; a trigger carrying several raise-message
actions yields each in action-list order; an inert trigger and a trigger whose conditions never
hold yield none; `git diff --name-only` names no path under `pkg/sim`.

## T4 the notice — impl

One box of text over the map screen: wrap, layout, paint, present, and the three inputs that
advance it.

Files: `pkg/ui/notice.go`, `pkg/ui/notice_test.go`, `pkg/ui/viewer.go`, `pkg/ui/app.go`,
`pkg/ui/flow.go`.

Covers: FR-7, FR-8, P-5, DD-3, DD-8, DD-9, DD-10, DD-16.

Fences: `pkg/ui` may name no simulation type — the seam carries a string, an integer kind and
booleans. Widen the map-loader tuple by **appending**, so no existing position moves. Where the
viewer has no font, draw nothing. Advance on the press edge only. Leave every other map-screen
input reaching the map.

Done when: text wraps at the last space that fits, breaks inside a word only when one word alone
does not fit, and is clamped to the lines the area holds; the layout is a pure function of text,
font and geometry; the button, RETURN and ESCAPE each advance once per press; ESCAPE with a notice
open advances it and does not leave the map screen; ESCAPE with none open unwinds as it does today.

## T5 the mission driver — impl

Hold the open text, its part, and the outcome already announced; push notices and answer advances.

Files: `pkg/game/world.go`, `pkg/game/mission.go`, `pkg/game/world_test.go`,
`pkg/game/mission_test.go`.

Covers: FR-10, FR-11, P-4, DD-11, DD-12, DD-15.

Fences: settle the outcome before the raises on every advance. Read the outcome from the world's
own outcome and never from the counters. One test decides discard, in this file. Push nothing into
the simulation and read no clock on the push path.

Done when: a world whose outcome becomes won, or lost, produces exactly one outcome notice and it
replaces an open dialogue notice; a raise arriving while any notice is open is dropped and is not
shown when that notice closes; advancing a dialogue notice pages then closes; advancing an outcome
notice selects the menu destination when lost and the map-list destination with its own message
when won; the tick and digest sequences over an advance are identical to those of the same advance
with no notice produced.

## T6 the door — impl

Open a campaign mission from the command line into the map screen.

Files: `cmd/againrom/main.go`, `cmd/againrom/main_test.go`, `pkg/game/frontend.go`,
`pkg/ui/app.go`, `pkg/game/frontend_test.go`.

Covers: FR-1, FR-2, DD-13, DD-14.

Fences: without the flag, no existing path changes — the menu, the map list and the map screen
behave exactly as they do today, and their tests stay unedited. The asset root still comes from the
flag or the environment and no install path is compiled in. Leave the menu and the map list intact
behind the opened mission.

Done when: the flag opens the named mission's map screen at startup with a party of one carrying
the fixed class key; the three startup failures each exit non-zero with a message naming the
address, and no two of the three messages are equal; the headless check mode accepts the flag and
reports without a window; starting without the flag opens the main menu.

## T7 the join — impl

Give a started mission's world the mission's own compiled script, so a trigger can fire at all.

Files: `pkg/mapload/start.go`, `pkg/mapload/start_test.go`, `pkg/game/mission.go`,
`pkg/game/mission_test.go`, `internal/synth/synth.go`.

Covers: FR-12, DD-18.

Fences: **no file under `pkg/sim` may be modified and no serialized form may move.** The plain start
and its own tests are untouched — add beside them. The compile keeps its present home and its
present error handling: a script that will not decode still starts the mission and its reason is
still carried, not returned. Add no second compile.

Done when: a scripted start over a map with a holding trigger yields a world whose script is not
nil, whose latch is set after one pass and whose instants ran; the same start with no program
yields the plain start's world and report, digest for digest, for a party of none, one and several;
an undefined difficulty is still refused before anything is attached; the door reaches the same
result over a synthetic archive, and the announcer raises the map's own event; `git diff
--name-only` names no path under `pkg/sim`.

## Traceability

| Task | FR | DD |
|---|---|---|
| T1 | FR-9 | DD-4, DD-5 |
| T2 | FR-3, FR-4, FR-5 | DD-6, DD-7 |
| T3 | FR-6 | DD-1, DD-2, DD-17 |
| T4 | FR-7, FR-8 | DD-3, DD-8, DD-9, DD-10, DD-16 |
| T5 | FR-10, FR-11 | DD-11, DD-12, DD-15 |
| T6 | FR-1, FR-2 | DD-13, DD-14 |
| T7 | FR-12 | DD-18 |
