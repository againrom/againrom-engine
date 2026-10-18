# Script spell parameters

## Intent

Choose a spell by number and installed display name in an action popup or the
E prompt. Both routes use the host choices. A current value without a named
row remains numeric and writes nothing when retained.

## Authority

Knowledge k328: `TRIG-SPELLDOMAIN-141`, `TRIG-EFFECTDOMAIN-142`,
`TRIG-SPELLCORPUS-143`, amended `TRIG-EFFECTTIME-034`, `TRIG-CAST-033`,
`TRIG-CELLTAIL-035` and `TRIG-PARAM-030`.

Supporting claims read: `DAT-LOC-001`, `DAT-GRAM-003`, `TEXT-091`,
`MAGIC-SPELL-001`, `MAGIC-MAPLAYER-040`, `MAGIC-ATTACH-016`,
`MAGIC-EFFECT-015`, `TRIG-CELLEFFECT-045`, `MAGIC-SHAPE-008`,
`MAGIC-EFFMODE-009`, `SAV-DOC-053`, `SAV-CELLREC-017`, `SAV-CELLLOAD-111`,
`ALM-TRIG-045`, `ITEM-CODE-029`, `TRIG-ITEMBIND-121`, `TRIG-ITEMKEY-124`.

The spell definition domain and identity copies are High; arbitrary authored
execution and the complete non-spell producer population remain Unknown.
The amended action-30 comparison uses the full dword, not a truncated
parameter. The cell tail has separate entry-cast and relocation readers;
the retracted sole-reader and shipped no-cast conclusions are not retained.

## Behaviour

| Action | Packed scalar argument | Named values | Numeric values retained |
| --- | --- | --- | --- |
| 21, Cast at cell | p4 | 1..28 | 0, 29..255 and larger values |
| 24, Cast at unit | p2 | 1..28 | 0, 29..255 and larger values |
| 25, Cell entry spell | p0 | 1..25, 27..28 | 0, relocation operation 26, 29 and larger values |
| 29, Cell effect time | p2 | 3, 7, 8, 12, 17, 19 | Every other value |
| 30, Unit effect time | p0 | 1..28 | 0, 29 and larger values |

The scalar index excludes preceding group, player, unit, item and structure
references. It does not assume an authored slot number. Only int/constant
spell slots acquire the view kind; the ALM type word is unchanged. Conditions
and other parameters keep their existing choices.

Action 29 uses the six established spell-layer identities conservatively.
The claim does not assert that other byte identities are impossible. Action
30 names spell identities; it does not promise a lasting attachment for every
spell. Its id-0 Potion attachments are not named as a spell. Values above 255
cannot match either effect-time comparison. Unknown values are not truncated
or renamed as aliases.

`recordCatalog("spell")` is the only spell catalog. The recipe, popup and E
routes read its rows, with action-specific filtering for scripts. Installed
names come from spell.txt line id-1, bounded by the running root's Spells
table. The recipe list now shows installed spell.txt names on both roots:
EN changes 19 to Stone Wall and 25 to Raise Spirit; RU changes all 28 former
English definition names to Russian names. Numeric identities and order are
unchanged. The recipe's zero row also changes to the localized EN/RU
`editor.param.nospell` word (RU: "Нет заклинания"). A missing
installed name is not offered. Synthetic roots without archives use their
definition names. No second spell list is maintained.

Picking changes the chosen value word only, on Apply. Undo restores the
source bytes. Retaining a current value writes nothing. The SAV format and
installed data are unchanged. Required merges inherit main's version files;
this lane makes no version bump. The remaining interface
difference and bounded area choices are in `DIV-3625`; its former unconditional
numeric-spell clause is narrowed. No reserved divergence ID is consumed.

Action 30 compares the zero-extended attached effect id byte with the full
authored p0 in both simulation and scriptcoverage. It writes the low word of
p1 to every match. An authored p0=276 cannot retime spell 20; p0=20 can.
Action 29 is unchanged. This follows the amended comparison in
`TRIG-EFFECTTIME-034` and `TRIG-EFFECTDOMAIN-142`. Remaining is hashed and saved.

## Proof

Synthetic tests: `TestScriptSpellChoicesRowsAndExclusions`,
`TestScriptSpellParameterFollowsPackedScalars`,
`TestScriptSpellUnknownNumberPreserved`, and
`TestScriptSpellPickWritesOnlyValueAndUndo`. They compare the host and popup
rows, exclude action-specific values, retain raw values including -1, compare
the complete byte slice against one changed dword, and test Undo and E.

Installed witnesses for both EN and RU:
`TestReleaseScriptSpellCurrentPicksPreserveEveryMap` checks every action node,
including unattached nodes, compares its current row to installed spell.txt,
picks it through the popup and saves every map byte-equal.
`TestReleaseScriptSpellPickerFrames` captures an installed campaign Cast
popup and open dropdown without a game window.

The expected per-root counts and value multisets are independently fixed
from `TRIG-SPELLCORPUS-143`: actions 21/24/25/29/30 have 35/20/2/23/2 nodes,
82 total. The witness includes all 38 EN and 34 RU catalog maps, including
RU's map without scripts.

Receipts, mutation controls, frames and the final gate results are under
`review/story1468-spell-picker/` in the seat's notes directory. The loss
controls overlay the base builder, shift each named value by one, or offer
excluded values; each must fail the row test for every action.

`TestInstantThirtyMatchesFullParameter` checks unchanged Remaining, world hash
and serialized state for p0=276, then the spell-20 Remaining write and hash
change for p0=20. The former byte narrowing fails this named test before the
fix and when restored by a temporary loss-control overlay. The overlay is
removed after measurement. Existing all-match, traced dispatch and SAVE/LOAD
tests use p0=20. `TestUnitEffectAgeMatchesFullParameter` checks the oracle's
positive and negative outcomes independently.

The existing `TestReleaseCampaignScriptExecutionClosure`, instrumented only
in notes, compares corrected and old-narrowing execution on both installed
roots. Both shipped action-30 nodes are mission 90, raw action indices 33 and
34 (zero-based), with p0=20. Every before/after hash pair below is identical
between the two implementations on EN and RU. The matching path creates its
effect through a production script cast. The 82-node editor witness separately
fixes the complete shipped action-30 population at two nodes per root.

| Raw action | Path | Before hash | After hash |
| --- | --- | --- | --- |
| 33 | Initial | 835fbe1f3bd48f18 | db51b8215c187488 |
| 33 | Matching effect | 2e7156121dc8ccfe | 521bdddf03fd3657 |
| 34 | Initial | 4c565ae24ac7b16b | faeb930217ab9d1b |
| 34 | Matching effect | e91449f6097b8e49 | be436a88352e09a7 |

Shipped-map execution is unchanged by the width correction. It therefore has
no separate gameplay changelog entry. Full comparison receipts, the temporary
overlay diff, its removal and gate results are in the correction notes.

## Open debt

Completed original casts, target admission, arbitrary byte ids 29..255,
larger authored values and the complete non-spell effect-producer population
remain Unknown under the cited claims. These UI witnesses do not execute
the original game or settle those questions. GPU presentation is outside the
lane's no-window boundary; its release witness must be reported as not run.
