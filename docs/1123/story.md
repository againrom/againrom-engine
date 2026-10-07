# Story 1123 — bird, statue-star and crowd ambience

## Player result

The playable town square now runs the installed bird episodes over its roofs,
including their rooftop occlusion strip and count-selected calls. The Gilded
Statue shows S00 on entry; pointing at it advances S01 through S08 and then
hides the star without changing the statue's click-to-menu action. The town
crowd requests its installed repeating loop while the square is active.

This is presentation only. Shop, tavern, school, gate, guards, sign, fluger,
labels, tips, navigation, native saves, world state and simulation hashes keep
their existing rules. No horse, baba, dervish or crowd visual is invented.

## Authority and policy

`TOWN-415` through `TOWN-420` at research pin
`5a9eaa5a08062dfd86fb8ecd2414aec15f5f932c` establish the nine 57-frame bird
families, three groups, one-to-three prefix, keyed overlay order, the
single-step no-catch-up progress increment, the S00..S08 selector16 machine,
the every-tenth terminal reset and the repeating Crowd request/cleanup route.
`TOWN-157` gives the strict 1000..2999 ms re-arm delay and `TOWN-404` the
strict `>67 ms` hub admission; `TOWN-415` and `TOWN-416` refine and preserve
both.

Original physical paint and pointer delivery, static alias resets, generator
identity, corrupt-art behavior, audio mixing and audibility remain Unknown.
Againrom therefore advances only from focused live App town composition and
uses explicit visible snapshots. A private bird generator cannot shift the
existing sign/fluger stream. FrontEnd retains the process bird delay and star
terminal counter; per-view episodes reset on effective entry. The final bird
paint is overlay-only, remains available through App's same-clock view
reacquire, then clears on the next distinct paint. The next delay starts after
that terminal snapshot.

Each bird sheet degrades independently; S00..S08 degrade atomically. One-shot
bird/star requests require a retained voice capability. Crowd uses a dedicated
retained loop selector, stops on focus/menu/room/reset boundaries and restarts
once on resume or re-entry. `DIV-858` through `DIV-867` disclose these client
policies. `DIV-149` is closed; `DIV-153` now covers only horse/baba/dervish and
any still-unknown indirect crowd visual.

## Touched surfaces

- `pkg/game`: optional art loading, process/view controllers, sound lifecycle,
  native/hash boundary and installed oracle.
- `pkg/ui`: explicit optional-frame visibility, bird/overlay/star layer order
  and the third retained loop selector.
- `internal/gatedtests`: the installed EN/RU witness population.
- `docs`: closed overlay divergence and narrowed remaining-wildlife debt.

## Proof

Focused tests cover per-family and atomic fallback, exact delay boundaries,
group/count selection, Birds1/Birds2 choice, all-three progress words,
one-step/no-catch-up, final overlay-only paint, inactive successor, S00 through
S08/hide, every-tenth terminal reset, next-arm S01, stationary selector16,
unchanged statue menu routing, crowd start-once/stop/restart/retry, missing
player/sample silence, explicit visibility, exact layer order and unchanged
native bytes plus simulation form/hash. Reflective FrontEnd and townScreen
censuses classify every new process or view field.

`TestReleaseTownAmbient1123InstalledBirdStarCrowdAndNative` independently
decodes literal installed paths for all nine bird sheets, all nine star BMPs,
the rooftop overlay, existing town motion and the four exact sound samples. It
drives the real App and checks bird1, bird3, terminal, post-terminal, S00, S08
and hidden frames pixel-for-pixel, plus repeating-loop lifecycle and native
save invariance on both preserved roots. Ignored witnesses live under
`review/story1123/{en,ru}` when `AGAINROM_STORY1123_FRAMES` names their parent.

The town transition scenarios remain the relevant headless routing witnesses.
No milestone census is needed because simulation, pathing and script
populations do not change.

## Open debt

Horse, baba and dervish activation/lifecycle contracts remain unimplemented.
Any crowd visual outside the bounded researched range, exact OS pointer and
paint delivery, process-static alias resets, random seed/sharing, sound overlap,
gain, loop phase and actual audibility remain Unknown.
