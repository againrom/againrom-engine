# A ROM1 script node that cannot be bound is not built

## Intent

Owner report, mission 100: the first servant follows the hero for the rest of
the game. The owner did not return the amulet to the elder and kept playing on
the map after the win.

Named result: on the ROM1 dialect, an action or a check whose hero-band
reference the binding roster cannot resolve is not built, takes no subscript or
register, and a trigger slot or pair naming it resolves to subscript or
register 0. In mission 100 each role of 10002..10005 the roster leaves
unresolved runs subscript 0 at the win, and the servant leaves the hero.

Base: engine main `33c929f7`. Knowledge pin moved from k218 to k225.

## Authority

| Leaf | Authority |
|---|---|
| an unresolved hero ordinal leaves its node unbuilt | `TRIG-HEROFAIL-078` (High for the 0 result, Medium for the unbuilt consequence) |
| three passes; an id naming no built node resolves to 0 | `TRIG-BIND-010` (High) |
| a slot naming an unbuilt node runs subscript 0; in 100.alm that is action 2 | `TRIG-M100-096` (High for the rule, Medium for which action holds subscript 0) |
| the servant's Follow starts from T1 and T2 only | `TRIG-M100-095`, `AI-437` |
| a later group command replaces or suspends a Follow | `AI-438` |
| the servant is destroyed at the mission end | `PARTY-M100-033` |

## As built

- `mapload.bound.unbuilt` is the one omission rule for both dialects. ROM2
  omits a node any of whose references fails, unchanged. ROM1 omits a node
  whose hero-band value (10001..11000) the roster leaves unresolved, and only
  when `ScriptRefs.Roster` says the hero band was resolved against the binding
  roster. An omitted node is listed in `ScriptReport.OmittedActions` or
  `OmittedChecks`.
- ROM1 trigger slots and pairs naming an omitted node resolve through the
  existing miss-to-zero lookup: subscript 0, register 0. Such a slot takes
  instant 0's announcement when instant 0 is a message, in the slot's position
  among the trigger's slots; an empty slot and an id the map does not hold
  raise nothing. A ROM2 slot naming an
  omitted action stays empty and a ROM2 trigger naming an omitted check is
  omitted, unchanged.
- `campaignScriptPartyRefs` sets `Roster` when it resolves a non-empty party on
  a map of player capacity one. A compile with no party, the almtool dump and
  every fixture keep building every node.
- A below-band unit or structure miss stays built (DIV-2792): LOAD recompiles a
  mission whose withdrawn records are gone, and such a miss is not a
  binding-time miss.
- LOAD: a SAV carries its compiled program, so a restored mission runs the
  program its mission load built. The role completion (`restoreCurrentScriptBindings`)
  and the terminal completion (`restoreTerminalScriptBindings`) compile with the
  same party roster, so their program has the loaded program's shape. A program
  saved before this rule built every node; `currentScriptRolesProgram` then
  compiles the roles without the roster, in that shape, and the saved program
  is kept (DIV-1658). The mission 30/130 legacy repair in `resume.go` compiles
  without the roster, as the old program was built.

## Census

Instrument: `TestReleaseScriptUnbuiltNodeCensus` over the 28 scenario.res maps
of each root, rosters: the primary alone in the four archetypes, and a full
roster resolving every hero-band value the script names. It fails when the
compile's omitted nodes differ from the nodes whose reference does not resolve,
and pins the two lists below. EN and RU give the same rows.

- Full roster: no unbuilt node on any map.
- Unit-id, structure-id and group misses at mission load: 0 on every map.
- Primary alone: every unbuilt node names 10002..10006. Maps whose triggers name
  one: 30, 60, 81, 90, 100, 120, 130, 131, 151 (111's unbuilt checks are named
  by no trigger).

| Map | Unbuilt, named by | Subscript 0 / register 0 | Effect for a roster missing the role |
|---|---|---|---|
| 100 | actions 37..40 (10002..10005), T16 | action 2, group 21 Move to (13,14) | the servant is sent to (13,14) at the win |
| 30 | action 14 (10002), T6 | action 2, message 4 | T6 raises message 16, then message 4; message 4 meets message 16's open dialogue and is dropped (`DLG-LIFE-005`), so the screen is unchanged |
| 60 | actions 56, 57, 59, 60 (10002, 10005), T6, T28; checks 62..65, T16..T18 | action 2, unit 70 off map / constant 10 | unit 70 taken off the map at the win; T16..T18 unchanged (10 == 1 false, as 0 == 1) |
| 81 | actions 21..23, 26..28; checks 22..24, 28..30, 35..37, 39..41 | action 2, ogre diplomacy / constant 4 | diplomacy at the T8 win; the check-named triggers stay false |
| 90 | actions 44..46, T4, T19; checks 36, 37, T13 | action 2, message 1 / constant 10 | message 1 at the win: T4 raises it for each unbuilt slot and T19 once, the first shows and the later ones meet its open dialogue; T13 stays false |
| 120 | check 5 (10005), T4 | constant 0 | none: register 0 holds 0, the value the unbuilt check held |
| 130 | action 39 (10002), T8, T9; check 45, T2, T3 | action 8, Force Mission Complete / check 28 | T2 fires at the start as before and sets state 2; T9 then wins the mission (the original does the same, owner run, DIV-2794) |
| 131 | actions 76..78, 81..83, T28, T29 | action 2, group 4 Move to (14,15) | group 4 moves when T28/T29 fire |
| 151 | actions 42..46, checks 6..11 (10006), T5..T9 | action 2, group 1 Stand Ground / constant TRUE | T5..T9 fire at the start: five increments of variable 60, the Lord thawed and moved, five Bless casts |

Subscript and register 0 follow file order; `TRIG-M100-096` grades that Medium
beyond 100.alm.

### Real rosters

Instrument: `TestReleaseScriptUnbuiltNodeRealRosters`, over `gameversions/saves`
and a copy of the owner's `engine/saves`. A file carrying this build's native
leaves (244) or under a generated-kit directory (6) is skipped; 0 refused. A
mission SAV of a changed map, or a town SAV offering one, binds that map against
the party this build restores from it. EN and RU give the same rows.

| Map | SAVs | Roles resolved | Unbuilt and named |
|---|---|---|---|
| 30 | 39 files: 2026-08-15 (1), 2026-08-24 (1), 2026-08-27/EXP-0261-owner-runs (5), 2026-08-30/EXP-0278-human-runtime-en (2), 2026-08-30/story-1073-original-acceptance (1), 2027-09-07 (29) | 10002 | none |
| 30 | EXP-0261 game0000, 0004, 0005, 0006, 9001, 9004, 9006, 9999 (parties of 1 or 4) | none | action 14: T6 runs message 4, dropped behind message 16 |
| 120 | owner savenu (town) | 10002..10005 | none |
| 130 | 2026-09-27/oldsaves7 game0005, game0006 (mission); owner savenakonecto (town) | 10002..10005 | none |
| 131 | 2026-09-24 game0002-bigsack, game0017-victory; owner game0001x, game0017, savenakonecto | 10002..10005 | none |
| 151 | owner game0003, saidenrelief, saidenrelief2 | 10003..10006 | none |
| 60, 81, 90, 100 | none in the corpus | - | - |

No original-produced roster reaches an outcome or off-map arm through
subscript 0. The owner saves listed carry no native leaf; whether each was
written by the original or by an older build is not established.

## Proof

- `pkg/mapload/scriptunbuilt_test.go`: the binder omits an unresolved role
  action and check under the roster, resolves the slot to subscript 0 and the
  pair to register 0, keeps a below-band miss built, builds every node with a
  resolved roster and without the roster.
- `TestReleaseMission100ServantLeavesAtTheWin`, EN and RU: with the primary
  alone the mission builds four fewer instants, the program after a SAVE and
  cold LOAD equals the one the mission load built, the LOAD role compile keeps
  its shape, the servant follows (state `0x11`) before the win, and at the win
  group 21's order is Move with the servant's target (13,14) and no escort.
  With npc 22..25 carried every role resolves and the servant still follows.
- `TestReleaseScriptUnbuiltNodeCensus` and `TestReleaseScriptUnbuiltNodeRealRosters`,
  EN and RU.
- `pkg/mapload` `TestASlotNamingAnUnbuiltNodeRaisesSubscriptZerosMessage`, on
  90.alm's T4 and T19 in miniature: each unbuilt slot raises message 1, an id
  the map does not hold raises nothing; a resolved roster raises nothing, and
  nothing is raised when subscript 0 is not a message.
- `TestReleaseMission30HealerWinRaisesSubscriptZerosMessage`, EN and RU: the
  hero placed at the healer; without 10002 T6 raises message 16 then message 4,
  with 10002 message 16 alone; both show message 16 and then the victory.
- `TestReleaseMission130WinsWithoutItsTenThousandTwoCompanion`, EN and RU: the
  primary with npc 23..25 and without npc 22 is won with no player action after
  32 ticks and the victory returns to the town; with npc 22 the same
  4000-tick drive does not end the mission.
- `TestLoadReresolvesHeroOrdinalsFromTheLoadedParty`: a program saved before
  the rule gets its roles completed in its own shape.

## Open debt

- DIV-1658: programs saved before the rule are not migrated; no warning window
  or log line.
- DIV-1657: no actor-registry scan; after an in-mission join a native LOAD keeps
  the mission-start program where the original rebinds.
- DIV-2792: a below-band unit or structure miss stays built.
- DIV-2793: mission 30's subscript 0 is file order, Medium.
- DIV-2794: 60, 81, 90, 120, 130, 131 and 151 change only for rosters no
  original save in hand shows; mission 130 without 10002 wins at the start, as
  the owner's run of the original does. Whether an original actor-registry scan
  binds a map-placed NPC (100.alm unit 245, 30.alm 56, 81.alm 52, 151.alm 205
  and 589) and so builds a node the engine leaves unbuilt is Unknown.
- Unknown: whether the original's group lookup ever fails at binding
  (`TRIG-M100-096` Medium); the corpus holds no group miss.
