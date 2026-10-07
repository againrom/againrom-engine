# Integration Fidelity Harness — proposal

Status: **PROPOSAL, NON-NORMATIVE**. This branch exists for review. Nothing in this document changes `AGENTS.md`, the pipeline-v2 ruling, story gates, architecture, or the meaning of existing evidence unless the owner explicitly adopts it.

## Problem

againrom already has strong local evidence: deterministic simulation, synthetic tests, real-install inspection tools, versioned headless scenarios, campaign sweeps, story closures, a divergence ledger, and adversarial review. The remaining risk is increasingly **composition risk**: several locally correct systems can form a player-visible path that is wrong.

The harness proposed here does not add another review layer. It gives the existing process a small set of long-lived, cross-story witnesses whose unit is a **player journey**, not a story.

The harness must answer questions such as:

- Can a new game reach a real campaign mission through production UI?
- Can mission state cross into town and back into the next mission without losing roster, inventory, purse, documents, skills, or campaign state?
- Can combat, loot, equipment, triggers, dialogue, mission verdict, persistence and campaign routing coexist in one run?
- Does save/load preserve the state that later systems consume rather than merely round-trip a file?
- When a later story changes one subsystem, which previously working journeys stop working?

## Non-goals

This proposal does **not**:

- replace story-level tests or closures;
- replace research claims as authority for ROM1 behaviour;
- compare againrom against ROM1 by treating againrom output as reference data;
- require every story to add a new journey;
- add pass-by-pass review prose;
- introduce a second divergence ledger;
- require screenshots or golden rendered frames for ordinary simulation assertions;
- require a game install under `go test ./...`.

## Build on the existing headless surface

The repository already has the correct substrate: `againrom --headless` scenarios reach either the production `ui.App` or a campaign mission's `sim.World`; install-backed scenarios prove production ingestion and synthetic scenarios can run without game data.

Do not create a parallel runner unless the current scenario vocabulary cannot express a required observation. Extend the existing vocabulary only when the new command or assertion observes a production value that a player journey genuinely needs.

The default harness is therefore a **curated suite of existing headless scenarios plus a small number of new cross-system scenarios**.

## Evidence levels

Every harness scenario declares one of these purposes in its adjacent manifest entry.

### M — mechanism

Synthetic assets. Proves an engine rule and its wiring under controlled state. Suitable for `go test` and deterministic regression.

It does not prove that shipped ROM1 data is decoded correctly.

### I — ingestion

Lawful install. Proves that production loaders, shipped content and the path under test compose against real assets.

It does not by itself prove ROM1 behaviour. A claim remains the authority for that.

### J — journey

Lawful install, production front-end path where possible. Proves that a player can traverse a meaningful cross-system sequence and that state survives the boundaries the sequence crosses.

A J witness should observe values produced by the production path. It must not recompute the same answer through a second copy of the rule merely to assert equality.

## Journey portfolio

Keep the portfolio deliberately small. The first useful target is six journeys, not one journey per story.

| ID | Journey | Boundaries exercised | Required end observation |
|---|---|---|---|
| J1 | New game to first playable mission | menu → picker → chargen → campaign → map | live controllable party in the selected mission |
| J2 | Mission success to town to next mission | sim verdict → notice → town → offer/selection → world map → map load | expected party/campaign state present in the next mission |
| J3 | Combat to loot to equip to later combat | AI/order → combat → death → corpse/ground → inventory → equipment → combat | equipped state changes a later production combat outcome or state |
| J4 | Save/load across a non-trivial state | gameplay → inventory/campaign mutation → save → abort/new session → load | production state after load equals the captured persisted state on named fields |
| J5 | Dialogue/trigger driven progression | map trigger → dialogue/notice → script state → mission/campaign consequence | the consequence becomes reachable through normal player input |
| J6 | Multi-mission continuity | mission N → town/world map → mission N+1, repeated | roster, historical fields, purse/documents and campaign route remain coherent |

Existing scenarios should satisfy a row when they already prove it. Do not duplicate them to make the table look complete.

## State checkpoints

A journey may checkpoint only state that matters after a boundary. Prefer stable semantic identifiers and exact integers.

Candidate checkpoint fields:

- campaign mission / route state;
- party membership and stable member IDs;
- HP, mana, XP and skills where the journey changes them;
- worn equipment and carried items by stable code;
- purse and documents;
- mission verdict and relevant trigger/script state;
- persisted history fields whose purpose is to survive later re-derivation;
- deterministic simulation hash where the hashed state is itself the contract.

Do not snapshot the entire world by default. Whole-state goldens create noisy failures and make intentional changes expensive. A checkpoint should name the minimum state whose corruption would make the journey semantically wrong.

## Production-value rule

A witness is invalid if its observation is derived independently from the same inputs rather than taken from the production value that the player-facing path produced.

Examples:

- If a doll composes equipment into a figure, assert the composition result carried out of that production composition, not a second call to a helper that ought to produce the same figure.
- If campaign routing chooses a mission, observe the mission the production controller actually opened, not a separately computed expected route using the same graph.
- If save/load restores a field, inspect the restored live field after load, not the serialized bytes alone.

This is a harness design constraint, not a request for another checker.

## Boundary mutation rule

For a new J witness, perform one deliberate local mutation during its introduction when practical: change the production line or constant whose regression the witness claims to catch and verify that the journey fails for the intended reason. Revert the mutation before landing.

The mutation is evidence about the witness, not a permanent test mechanism. It should be recorded tersely in the proposal/story closure if this harness is adopted; no mutation diary is required.

A mutation that changes a stand-in path the real journey never reaches proves nothing.

## Harness manifest

If adopted, add a small machine-readable or Markdown manifest beside `scenarios/` with one row per curated journey:

| Field | Meaning |
|---|---|
| `id` | stable J1… identifier |
| `scenario` | existing scenario path |
| `evidence` | M, I or J |
| `boundaries` | named subsystem boundaries crossed |
| `claims` | research claims relevant to fidelity assertions; may be empty for authored behaviour |
| `divergences` | live divergence IDs intentionally exercised |
| `observations` | semantic fields the journey actually checks |
| `owner_intent` | optional note where the expected behaviour is authored rather than researched |

The manifest is an index, not a second specification. Scenario files and production code remain the executable evidence; research remains authority for ROM1 truth; `docs/DIVERGENCES.md` remains the only divergence ledger.

## Execution lanes

### Fast lane

Run synthetic mechanism scenarios and ordinary Go tests. No game install required.

### Fidelity lane

Run install-backed curated journeys against both supported asset roots where the scenario is language-independent. A language-specific scenario states why.

This lane is appropriate before a release candidate and after changes to cross-cutting state, persistence, campaign routing, scenario infrastructure, map loading, or hashed simulation semantics. It need not become a mandatory cost on every tiny documentation or leaf-format change.

## Failure classification

A harness failure is classified before work begins:

- **PRODUCT** — player-visible production behaviour or hashed simulation state is wrong;
- **WITNESS** — production is right but the journey observes the wrong value, cannot reach the production state, or encoded a stale assumption;
- **EXPECTATION** — the expected behaviour conflicts with current research, owner intent, or a recorded divergence;
- **INFRA** — asset root, runner, environment or harness machinery failed before the behaviour was exercised.

Only PRODUCT is automatically a product defect. EXPECTATION requires authority reconciliation before changing code. WITNESS fixes the witness. INFRA fixes or reports infrastructure.

This deliberately prevents a green/red harness from becoming an accidental new authority on ROM1 behaviour.

## Adoption criterion

Adopt this harness only if a pilot demonstrates all of the following:

1. At least three portfolio rows can be satisfied mostly by existing scenario infrastructure.
2. At least one pilot journey catches a deliberate mutation in a production path that story-local tests do not obviously cover.
3. The full curated fidelity lane remains small enough that agents will actually run it.
4. The manifest does not duplicate specifications or divergence prose.
5. A harness failure can be triaged to PRODUCT/WITNESS/EXPECTATION/INFRA without opening an adversarial review chain.

If the pilot fails these criteria, discard the proposal rather than institutionalizing it.

## Suggested pilot

Start with J1, J2 and J4 because the repository already has strong front-end scenario primitives for new game, mission-to-town, inventory/shop and save/load paths.

The pilot should be implemented as a separate story only after this proposal is accepted. Until then, this branch is documentation for agent review, not a process change.
