# Current creature slot words in SAV

## Delivered scope

The common current-world producer writes every creature slot word from
`Entity.CreatureSpells`, including an empty set. The previous empty-set branch
kept the retained order window. No simulation rule or LOAD reader changes.
This is one completed field family of audit G14. G13, G15, G7 and the other
G14 families remain open. No ordinary player route that clears these slots has
been established, so this slice adds no CHANGELOG claim.

## Authority

The owner's source order is current state, an evidence rule, loaded bytes,
then an established constructor; a missing producer is named debt.
`SAV-1066` (High / Medium) establishes the unconditional raw STORE of the
148-byte order block, including all six slot words. `AI-377` (High for the
loop) establishes that a zero ID skips its draw and a zero threshold never
matches. Those claims do not establish an ordinary runtime slot edit.
`DIV-1729` retains the existing legacy zero-block LOAD compatibility boundary.

## Field sources

| Field | Selected source | Scope / row |
| --- | --- | --- |
| U158 +0x78, slot 1 ID | Current Entity.CreatureSpells[0].ID, including zero | Present bound mission actor; DIV-1729 |
| U158 +0x7c, slot 2 ID | Current Entity.CreatureSpells[1].ID, including zero | Present bound mission actor; DIV-1729 |
| U158 +0x80, slot 3 ID | Current Entity.CreatureSpells[2].ID, including zero | Present bound mission actor; DIV-1729 |
| U158 +0x84, slot 1 threshold | Current Entity.CreatureSpells[0].Threshold, including zero | Present bound mission actor; DIV-1729 |
| U158 +0x88, slot 2 threshold | Current Entity.CreatureSpells[1].Threshold, including zero | Present bound mission actor; DIV-1729 |
| U158 +0x8c, slot 3 threshold | Current Entity.CreatureSpells[2].Threshold, including zero | Present bound mission actor; DIV-1729 |
| Original world-effect Token and payload Token | Debt: missing complete World Token/graph model; loaded bytes remain | G14; DIV-2502 |
| Imported actor U50 | Debt: shared party AI-state producer/import model remains open | G14; DIV-1492 |
| Town Human Block12 cell and fine position | Current Saved.Cell when valid; otherwise retained bytes / constructor debt | G14; DIV-1414 |
| Town head map name | Existing last-played-map rule; research on original writer remains open | G14; DIV-1412 |
| Before-first-tick body-target order words | Retained order debt pending original LOAD chronology | G14; DIV-1741 |
| Projectile spelling, leaf order and inactive raw domains | Loaded-document debt; active fields already follow World | G14; DIV-1814 |
| Item Token RuntimeID | Current object carrier when bound; otherwise retained bytes / constructor debt | G14; DIV-1419 |
| Option, panel and view leaves | Existing application projection, retained town leaves or zero constructor debt | G14; DIV-878 |
| Human U8E | Existing loaded accumulator plus equipment deltas; native worn-sum rule debt | G14; DIV-3358 |
| Superseded Block12/U154 | Existing bounded current-motion producer; historical / scheduling debt | G14; DIV-1092 |
| Town U154, U158, U50, U136, U138, U6C, U15C, U178 | Constructor / party debt; town has no complete actor-state carrier | G15; DIV-2500 |
| Town actor Diaries, selection | Missing carrier / empty-selection debt | G15; DIV-2500 |
| Town Player F44, F3D, F50, Outcome, PRaw32 | Constructor debt; no complete town Player carrier | G15; DIV-2500 |
| Terminal original body worn, carried and effect objects | Missing body-contents carrier debt | G13; DIV-2505 |
| Departed actor without map unit or party entry and its contents | Missing complete departed-body carrier debt | G13; DIV-2501 |
| Actor T18 | Existing 2/on-map, 0/off-map rule is debt for other masks | G7; DIV-2503 |
| Worn item T08 | Debt: the pickup stamp is not a universal worn-item zero rule; keep current item Token policy | G7; DIV-1419 |
| Player and Human five null references | Debt: no admitted claim fixes all five reference values or their lifecycle | G7; DIV-2500 for town Player construction; other references remain audit debt |
| Uninitialised Unit bytes | Zero constructor debt | G7; DIV-1719 |
| Unbound actor RuntimeID | Per-SAVE remint debt | G7; DIV-2504 |

DIV-3925 through DIV-3934 are unused. No constructor value is established by
this slice. The following source census narrows what each remaining family
needs; it does not establish an original-game rule.

## Remaining producer boundaries

| Family | Current production boundary | Required work |
| --- | --- | --- |
| G14 effect Token | `currentWorldEffects` and `projectSavedWorldEffects` update modeled effects; the Snapshot binds the other Token words and child graph | A complete World Token and effect graph with alias and removal controls; DIV-2502 |
| G14 actor U50 | `savactoractions.go` and `savactionwire.go` project supported actions; the party state does not implement every imported AI state | Shared party AI import, transition and export model; DIV-1492 |
| G14 town position | Town LOAD clears restored mission placement; a later SAVE has no replacement fine-position carrier | A town placement lifecycle, including zero coordinates; DIV-1414 |
| G14 town map name | `currentCityDocument` uses the last played map | Original town writer evidence, including sub-maps; DIV-1412 |
| G14 pre-tick body target | `importSavedActorActions` drops a non-targetable body; the pre-tick export exception retains the source order | Original LOAD and first AI-call chronology; DIV-1741 |
| G14 projectile raw domains | Active leaves follow World; spelling, leaf order and inactive domains have no current producer | Typed inactive domains and their transitions; DIV-1814 |
| G14 item RuntimeID | Bound current item Tokens are projected; no universal Item/Effect allocation rule is established | Allocation and LOAD membership evidence for the uncovered populations; DIV-1419 |
| G14 option and view leaves | Application projection covers modeled values; absent town state uses retained records or typed defaults | Current town/application carrier for each remaining leaf; DIV-878 |
| G14 Human U8E | Imported load bookkeeping keeps the accumulator and equipment deltas; native construction uses the worn sum | Original native construction evidence; no new sum rule; DIV-3358 |
| G14 historical motion | Current motion has its bounded producer; superseded native Document residue and unsupported scheduling remain | Independent scheduling authority and matching continuation proof; DIV-1092 |
| G15 town actor and Player fields | Town has party, item topology and town carriers, without a complete actor order/mover, actor Diary or selection model | Town lifecycle carriers or individual cited constructors; DIV-2500 |
| G13 original terminal contents | Original dead import keeps a terminal tuple and bounded Unit weapon. Its passive body has no ordinary equipment/pack owner | Separate passive graph import, bounded persistence, writer projection and cold LOAD; DIV-2505 |
| G13 departed actor | `World.remove` releases actor item roots before retaining terminal state. `currentTerminalBodies` reconstructs only map/party actors | Capture the complete body and contents before removal, including an unplaced actor; DIV-2501 |
| G7 normalization and identity | The ordinary actor mask rule and zero constructors do not establish all masks, null references, worn flags or stable unbound runtime IDs | Individual claims or current identity carriers; DIV-2503, DIV-2504, DIV-1719 and the reference debt above |

G13 is not implemented. Merely retaining writer references would still lose
items already disposed by `retireRemovedActorObjects`. Importing them as live
equipment would also break the existing terminal-body admission contract.
The remaining slice requires EN/RU file and cold-World presence tests, alias
and ordering checks, deterministic historical defaults and a loss control.
No fixture-only result closes it.

The integrated main writes Order +0x40/+0x44 and Mover +0x08 from its landed
carriers. This story changes neither reader nor writer for those fields. Main's
town cast-weapon construction and profile SAV digest pins remain main's values.

## Proof and limits

`TestCurrentCreatureSlotsReplaceEveryLoadedWord` changes the World after LOAD
and compares each of the six independent words against an intentionally
conflicting retained word; the five peer words stay zero. It covers zero and
nonzero current values and
requires every byte outside the 24-byte slot window to stay unchanged.
The base implementation fails all six empty-window cases.

`TestCurrentCreatureSlotWordsSurviveSaveAndColdLoad` checks each word through
`ExportCurrentSave`, cold LOAD, a second SAVE and following ticks.
`TestReleaseCurrentCreatureSlotWordsFollowWorldAfterLOAD` is the EN/RU
installed counterpart: original mission input
`2026-09-24/game0002-bigsack.sav`, six independent World mutations, F2 SAVE,
cold LOAD, second F2 SAVE and following ticks. Existing
`TestReleaseCreatureSpellSaveColdLoadCastsNext` remains the installed cast
continuation witness. Fixture proof alone does not establish install proof.

The shared writer is also used by the second game. No second-game rule or
reader is changed. Its SAVE selection and all digest movement must be reported
separately in the lane result. Native original reload and its next action are
NOT RUN under this lane's no-original-program boundary; they are separate
acceptance obligations.

Exact gate receipts, candidate SHA, loss-control results and observed digest
movement belong to the external lane result. A green focused test alone is
not release acceptance.

The storyguard comment-byte baseline falls by 107 because the producer comment
now describes the unconditional projection in two lines instead of three. No
identifier, command-literal or composition count changes.
