# 0066 — provenance

Claims are cited by id and never by experiment, so a later amendment or retraction reaches this
story through the ledger that carries it.

## Backing — what the spec asserts, and on what

| Spec anchor | Source | Confidence | What it fixes |
|---|---|---|---|
| FR-3 (address) | `MISSION-TEXT-005` | High for the path construction | `main.res::text/battle/m<mission>/event<NN>.txt`, the event number formatted two-digit |
| FR-3 (read at fire time) | `DLG-READ-006` | High | nothing loads event text at map load; the resource is opened when the announcement fires |
| FR-4 (absent is silent) | `DLG-ABSENT-003` | High | no window, no fallback, no fault, no blocked state; 223 of 242 raised numbers ship a file |
| FR-4 (the silence is authored) | `MISSION-TEXT-005` | Medium for the number-to-file mapping | the 19 unshipped raises are no-ops rather than a defect of ours |
| FR-5 (part scan) | `DLG-MARKUP-007` | High for the vocabulary and the scan | tag bodies are lowercased and substring-searched; first match in file order wins |
| FR-5 (the substring hazard) | `DLG-MARKUP-007` | High for the mechanism, exhaustive for the absence | `part=10` satisfies a search for part 1; 0 shipped files trip it on either root |
| FR-5 (part 1 always exists) | `DLG-EMPTY-004` | High on both halves | 0 of 225 EN and 0 of 228 RU files lack a `part=1` tag, and none has a gap before its own maximum |
| FR-6 (opcode 2 raises text) | `MISSION-TEXT-005`, `MISSION-M10-009` | High | `Send message` takes a number, and mission 10's chain raises 1, 2 and 3 from triggers 2, 3 and 4 |
| FR-6 (it is the dominant action) | `MISSION-TYP-010` | Medium (corpus agreement) | instant 2 is 254 of the 797 shipped instant nodes |
| FR-7 (panel geometry) | `DLG-WIN-001` | High for the geometry and both layouts | panel `{30,120,610,360}`; child rects are panel-relative; text control `48,36-428,172` without a portrait; button `200,172-280,198` |
| FR-7 (wrap, pitch, clamp) | `DLG-WRAP-009` | High for the wrap, the clamp formula and the default pitch | pitch is font height + 2; visible lines are `min(height/pitch, line count)`; the window is given no scrollbar and no arm answering one |
| FR-8 (three dismiss inputs) | `DLG-LIFE-005` | High | the button, RETURN and ESCAPE all reach one command |
| FR-8 (paging, then close) | `DLG-LIFE-005` | High for the pager loop | a further part redraws and stays open; no further part closes the window |
| FR-8 (no timer) | `DLG-LIFE-005` | High | the class census finds 13 functions, 0 orphan bytes, and none references a timer |
| FR-8 (a second is discarded) | `DLG-LIFE-005` | High for the drop | the gate bit is set when a panel shows and the arm returns without building anything; nine clears, all in the teardown |
| FR-9 (conversion on every drawn byte) | `TEXT-CONV-001` | High | selector 1 moves `0x80..0xAF` up by `0x30` and `0xE0..0xEF` up by `0x10`; every other byte, and every other selector, is identity |
| FR-9 (the map is injective **on the shipped alphabet**) | `TEXT-CONV-001` for the rule, `TEXT-FIT-004` for the domain | High for the rule; the injectivity needs both rows | `TEXT-CONV-001`'s stated reason — that the moved blocks land where "no unmoved byte" sits — does not hold: `0xB0..0xDF` and `0xF0..0xFF` are themselves unmoved, so each of their 64 bytes shares a record with the byte that moves onto it, and the map is 64-to-1 over the full byte range. It **is** injective on ASCII plus the two moved blocks, which is the domain `TEXT-FIT-004` measured the Russian corpus to occupy exhaustively (1 952 of 1 952 high bytes inside it, 0 outside). The claim's consequence is sound for shipped data; its justification is not, and this story pins both halves rather than the convenient one |
| FR-9 (the selector's source) | `TEXT-LANG-002` | High | one dword, one writer, computed from the trailing ASCII digit of `main/id`; EN `english 0`, RU `russian 1` |
| FR-9 (identity is the EN path) | `TEXT-CONV-001`, `TEXT-LANG-002` | High | selector 0 returns the argument untouched, so an English install draws exactly what it drew before |
| FR-9 (the subscript rule) | `TEXT-INDEX-003` | High for the rule and for the absence of a bound | `record = (byte)(conv(b) - 0x20)`, unbounded, with no substitute glyph and no clamp |
| FR-9 (the fit is two-sided) | `TEXT-FIT-004` | High | 1 952 of 1 952 RU high bytes land on an inked record under the converter; 0 of 1 895 localised bytes do under the identity rule on `font2` |
| FR-10 (outcome values) | `MISSION-END-013` | High | the reporter tests `== 1` on each counter, not a threshold |
| FR-10 (the world keeps stepping) | `MISSION-STOP-016` | High | the pacer's gate is cleared only by the teardown, which is reached only after a panel is dismissed |
| FR-10 (`> 0` is the wrong predicate) | `MISSION-LATCH-014` | High for the writer and clearing sets | the counters only rise, nothing clears them inside a mission, and a repeating check drives one up without bound |
| FR-11 (win to the town, lose to the menu) | `MISSION-PATH-015` | High for the branch structure, Medium for the destination clause | the lose chain ends at the routine whose string operands are the menu's; the win chain routes to the town |
| FR-1, FR-2 (what a mission start is) | `MISSION-START-001`, `MISSION-DROP-002` | High for the positive half | the player's units are placed from the player's own list at the map's drop cell |

`0063` and `0065` are cited in the spec as **this repo's** shipped contracts, not as evidence: the
compiled script, `World.Outcome` and `game.StartMission` are consumed at their own specs.

## Ours by choice — fixed here, asserted by no source

| Spec anchor | What we fixed | Why it is ours |
|---|---|---|
| FR-1 | `-mission <n>` as the door, straight to the map screen, with no menu entry | Nothing decoded says how a campaign is entered; the original's shell is a campaign screen this tree does not have |
| FR-2 | A party of exactly one, carrying an authored class key | `PartyMember` carries a class and nothing else by `0065`'s own disclosure: what a hero *is* comes from character generation, which does not exist here |
| FR-7 | The colours, the frame, and the absence of a portrait pane | The window's art is shipped bitmap we do not read; the geometry is the claim's, the paint is ours |
| FR-7 | Breaking a line at a word boundary, and mid-word only when one word alone exceeds the rect | The engine's splitter was read as a splitter, not as a rule. The rect and the pitch are the claim's; the break policy is ours |
| FR-9 | Keeping the bounded subscript: an out-of-range record draws the space | `TEXT-INDEX-003` establishes that the original is unbounded and reads out of bounds. We diverge deliberately and safely, and FR-9 says so |
| FR-10 | The banner's words, its geometry, and that it is dismissible | The original's panels come from string-table entries 140 and 141, which we do not read |
| FR-11 | A win stops at a named seam and returns to the map list | There is no town. `MISSION-PATH-015` says where the original goes; nothing here can go there |

## Open — undecoded, and deliberately given no meaning

| What | Why it stays open |
|---|---|
| The portrait and the `npc=` tag | Needs `npc.reg` and face art. `DLG-FACE-008` records that the shipped corpus cannot even discriminate the flag's two tests |
| `sound=`, `tune=`, `tips=`, `iamfemale`, `iammale`, `iammage`, `iamfighter`, `npcalive=`, `npcdead=`, `female`, `male`, `mage`, `fighter` | `DLG-MARKUP-007` grades the effect of each conditional literal Medium — they were read as tests, not as effects — and four are never used on either root |
| `~` as a rule glyph | `TEXT-TILDE-009` is High on the mechanism and **Unknown** on what it is for and which strings use it. No census exists, so we neither reproduce nor exclude it |
| Whether a shipped mission text exceeds the clamp | `DLG-WRAP-009` is Medium on text past the clamp being unreachable. The longest shipped part body is 230 bytes EN and 211 RU |
| The three EN-silent announcements | `DLG-LANG-010` records `m100/event09`, `m130/event07` and `m150/event10` as RU-only, from scripts identical on both roots. We reproduce the silence rather than explain it |

## Removed — dropped from the contract, and why

| What | Why |
|---|---|
| Implementing instant 2 inside `pkg/sim` | It changes no simulation state in the original either, and `pkg/sim` can take none: the field sets are pinned for exact equality and the byte form's next version is spoken for elsewhere. The fact is derived from the latch array instead, and `Script.Unsupported` still names opcode 2 |
| Announcing from `ScriptCounters` | `MISSION-LATCH-014` shows the counters are the wrong predicate. The outcome is read from `Outcome()` alone |
| Queueing a second announcement | `DLG-LIFE-005`: the original drops it. A queue would show text the original never showed |
| A fallback string for a missing file | `DLG-ABSENT-003`: there is no fallback on that path. The one literal that exists is reached only by a file that ships and yields no part 1, which `DLG-EMPTY-004` shows no shipped file does |
