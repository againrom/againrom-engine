# Verification

Focused synthetic tests pass for lossless movement/history, invalid/no-op moves,
cancelled drags, atomic malformed/dirty opens, checkpoint retention, safe explicit
new-file writes, native input dispatch, retained view context and modal layout.
Real temporary symlink and Windows junction checks pass without skipping.

Focused installed drives pass on both roots:

| Install / source | Unit index / ID | Cell move | Bytes compared |
|---|---|---|---|
| EN loose Beast.ALM | 0 / 31 | 169,56 to 172,58 | 342544 |
| RU loose FORESTER.ALM | 0 / 136 | 234,23 to 237,25 | 317784 |
| EN and RU scenario/10.alm | 2 / 21 | 36,51 to 39,53 | 67554 each |

Every byte outside the selected coordinate pair is unchanged. The completed
drag is one history entry; Undo restores the original canvas exactly, and fresh
reopen restores the moved canvas exactly at the same camera. Archive-map unit
references and raw coordinate details update to the new cell. Source inputs
remain byte-identical. The Save As path uses temporary output outside installs.

Observable result: `cmd/mapedit`'s production editor can now move a unit and
produce a lossless new ALM through its visible controls. Untracked composed
frames live under `review/story1118/{en,ru}/`; they are not repository assets.
They witness CPU composition, not a native-window or original-runtime run.

## Final candidate

Tested code: `72dfb69cad1b424f47a598cf11e0f6af95ce02ab`, reconciled with
published master `6c13f93781eac4c6630ee806bbd2dea7a139199b` after stories 1116
and 1117. Research remains `1172d41a90ef928aa7345f76d0d3dcf0fc987bc9`.
The final proof commit changes this verification document only.

| Check | Result |
|---|---|
| Focused model, editor, safety and architecture tests; gofmt | PASS |
| `go test -trimpath -count=1 ./...` | PASS |
| One paired `check-release-tests.sh` invocation | EN 149/149; RU 149/149; zero missing subjects |
| `check-no-game-assets.sh` | Clean tree scan |
| `check-seat-tree.sh` and working-date guard | PASS; clean branch and exact research checkout |
| `check-div-claims.sh --ids` | Parsed all 292 live rows; 74 amended-claim matches |
| Allocation sweeps before/after ledger edit and master reconciliation | 32 answers; zero missing; floor 826 unchanged |
| Windows path safety and `go vet -unsafeptr ./pkg/game` | PASS |
| Preserved-install check after write-capable drives | 181 files; both roots unchanged |

The added `DIV-618` match is `ALM-UNIT-040`: its amendment corrects the roles
of fields +0x40 and +0x42, not its unchanged 70-byte stride or coordinate
DWORDs. The pinned claim reader was checked again; this implementation uses
`ALM-UNIT-048`'s corrected unit/group identities. No new divergence IDs were used.

The lane-required mission 10/20 trace probes each report zero `UNSUPPORTED`
nodes. Their script populations remain 16/27/12 and 14/15/11
(checks/instants/triggers), matching `pipeline/milestone-baseline.txt`.
`-ticks 1` is clamped by the existing runner to 64 ticks. No full milestone
campaign census was rerun for this editor-only change.

The initial final-Go attempt at `9f6e3465` rejected an external Windows import
in the game tier and a non-ASCII device-name string. Both were corrected without
weakening either architecture guard. That candidate's paired release did pass,
but the final paired release was rerun on corrected code. An additional broad
`go vet ./pkg/game` reports nine existing unkeyed `BookSpell` literals in four
unchanged save-related test files; those unrelated tests were not edited.

Logs in the seat's untracked `.cache/`:
`story1118-final-go-72dfb69c.log`,
`story1118-final-release-72dfb69c.log`,
`story1118-final-guards-72dfb69c.log`, and
`story1118-m{10,20}-72dfb69c.log`.

Composed EN/RU before, moved/Modified, long-path prompt, refusal and fresh-reopen
frames were visually inspected. The modal overlap found in the first capture
was fixed; path/caret and refusal text now occupy separate bounded rows.
The installed font verifies the complete `Ctrl+Z / Ctrl+Y / Ctrl+S` footer fits
300 pixels. The frontend-design skill kept the existing six-colour rail,
installed font and label/value hierarchy; it introduced no new art or redesign.
These frames use UI code at `248417f4`, unchanged by subsequent host-safety and
test-only corrections. No native-window interaction is claimed.

No merge, current-build rebuild or independent review was performed by this
lane. The exact pushed candidate is returned to the seat for its sole review
and serialized landing.
