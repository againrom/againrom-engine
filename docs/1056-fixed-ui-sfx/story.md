# 1056 — fixed UI SFX events

## Player result

The game now resolves and plays the shipped interface samples at the decoded events:

| slot | event | timing |
|---:|---|---|
| 1 | character/campaign pane mode, previous, next and menu routes | before transition |
| 2 | accepted common controls, including picker/menu/document actions | before action |
| 7 | mission pack/book and shop book toggles | after child mutation |
| 8 | mission command-panel message `0x40c` | on dispatch, before refusal gates |
| 14 / 16 | mission-complete / mission-failed notices | after outcome child creation |
| 100 | Sound Options `TEST SOUND` action | on action |

`VIDEO-SFX-013`, `VIDEO-SFX-015`, `VIDEO-SFX-016` and `VIDEO-SFX-020` are the authority. In particular, `0xdc` is playback priority rather than a selector. Slot 15 remains excluded: its physical labels and ordering are still Unknown.

`TEST SOUND` is this build's concise route to decoded Sound Options action 7; the research identifies that action and sample, not a physical label.

## As built and proof

`pkg/ui` uses the existing optional audio player and `SoundBank`; interface samples are centred while map sounds retain their positional path. `FrontEnd.App` receives the same production device and bank as every map viewer, so no asset bytes enter this repository.

Focused fake-sink tests pin all seven selectors and the event ordering. The release test opens `sfx.res` through `OpenSounds` and requires slots 1, 2, 7, 8, 14, 16 and 100 to decode on each lawful EN/RU root. `DIV-314` is closed; its redraw half is inapplicable to a client that recomposes every frame.
