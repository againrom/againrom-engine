# SAV output for remaining save tools

## Result

The candidate removes production callers of the AGS envelope encoder. Ordinary
SAVE already writes SAV on the reconciled engine main. The headless convenience
SAVE now selects SAV, `scenariofixture` exports its controlled current World to
SAV, and the owner fidelity witness writes SAV and loads it through the normal
local-save row token. Explicit conversion accepts only AGS input to SAV output.
`savemigrate` and `saverepair` retain report-only legacy readers.

This is writer removal. The AGS reader, test fixture encoder, legacy store API,
and migration corpus remain until the one-time conversion is complete. Final
reader retirement is outside this candidate.

## Authority

Owner direction requires one `ExportCurrentSave` producer over current state,
with no AGS fallback and no new save refusal. Install files and owner saves are
read-only inputs. Original behaviour is not inferred from this implementation.

## Touched surfaces

- `pkg/game/headless.go`, `savedialog.go`, `saveconvert.go` and
  `owner_fidelity_witness.go`.
- `cmd/saveconvert`, `cmd/scenariofixture`, `cmd/savemigrate`, `cmd/saverepair`.
- `internal/archtest/agswriter.go` rejects production encoder references.
- Mission capture keeps the current same-code weapon display name through
  the existing named-weapon supplement. Item definitions still own combat
  values. The fixture passes the imported autohealing policy into its new
  mission through the normal incoming-party input.
- Legacy conversion fixtures remain test-only. Existing city-sales and
  spellbook SAV/next-action witnesses use the current main versions.
- The original corpus runs original SAV -> LOAD -> current SAV -> fresh LOAD.
  Its comparison observes the current roster separately from the producer's
  captured Snapshot. No new refusal or loss allowance is added.

## Proof

Reconciled base: `d319c75829b5cfa2d4cd39902d5f9a008ad43f34`.
The paused branch at `6caa6a19caa46ab37a0caa8e2690e0cb9b1ad012` remains intact.
The knowledge pin stays `701f478f597f85641da9e327a3fb0a2c407845c6`.

Before the later main reconciliation, EN and RU each audited 111 original files: 70 mission, 40 city and one
unreadable input. All 110 readable files completed the direct SAV cycle with
exact compared state, zero refusals and zero mismatches. Discovered-population
floors are 111 original files and 114 legacy AGS files. The corpus was not repeated after reconciliation; exact means the existing
comparison fields, not all source bytes or complete future behavior. The AGS
corpus remains a separate migration instrument with its existing disclosures
and ceilings.

The production AGS guard, report-only tools and EN owner HUD/SAV
row/double-click/cursor witness passed. Spell transport retained 161 route and
current-damage samples. The headless current-gold SAV regression and the
weapon-name regressions passed.

EN and RU each passed both shipped mission-endpoint scenarios and the
fixture's complete party comparison at the typed inventory boundary. The
source entry template has 102 null item-handle fields. Each binds to one of
15 independently observed source World objects only after exact item value,
order and stack-count checks. Cold snapshot handles match its World without
null adoption. One ID bijection across the whole party preserves aliases,
distinct objects and nulls; every other Party field compares exactly.
Controls reject merged objects, split aliases and null materialization.
The former ManaReservePercent exclusion is removed. The fixture constructs
a new mission and does not claim complete source World equivalence.

The imported-city conversion checks each source object and state leaf, with
three explicit boundaries. Reminted object identities compare through table
indices. Placement cells compare with the pre-conversion current party's
Saved.Cell, which town LOAD clears; all other token bytes stay exact. The
existing town selection debt remains exactly /Objects/Selection [1] to empty,
as recorded by writerCensusDebt. This route does not claim complete source
city fidelity. The current continuity leaf has its own full-party and
next-mission proof. Independent party and gold equality and mission30 World
hashes passed at every tick from 0 through 32.

## Review boundary

The EN/RU changed-roster city controls and production AGS guard passed.
The measured storyguard counts are 5592 test identifiers, 271 test filenames,
7882115 comment bytes and zero forbidden comment forms. The tracked asset
scan is clean. Resumed owner receipts are `city-final-6` and `fixture-final-8`;
paired corpus evidence is in `focused-en-1` and `corrections-3`.
The latest main ending SAVE restrictions, city potion/identity changes and
fresh-game autohealing changes are preserved. EN/RU fixture, alias, weapon,
new-game policy/session, headless SAVE and ending-no-SAVE checks passed on the
reconciled code. `candidate-focused-10` holds the paired runtime results; its
guard leg did not complete. `candidate-correction-11` is the passing guard
receipt: fixture/alias retests, converter compilation/path checks, production
AGS guard, dependency DAG, gated population, storyguard and asset scan.
It completed in 21.843 seconds with exit 0 and no remaining processes.
The full corpus and final chain are not repeated here.
The sole review and serialized final gates belong to the seat. These focused
results are not release or landing evidence.

The old DIV-1395 four-refusal observation is absent from the measured current
corpus. The paused DIV-1396/1397 statements are not copied into the current
ledger. No new divergence row or source-selection equivalence is asserted.
Reserved IDs are not recycled. AGS one-time conversion and reader retirement
remain outside this phase.
