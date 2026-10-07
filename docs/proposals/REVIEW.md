# Adversarial review — integration fidelity harness proposal

Reviewed at implementation master `744b2fc`, research master and pin `d1d350d`, proposal branch
`origin/proposal/integration-fidelity-harness` (`6ae2196`), merge-base `727722c`.

Every count below names the command that prints it. Commands are run from
`<seat>\implementation` unless another directory is named.

## Premise verification

The four premises supplied with this review were checked before any review work.

| Premise | Command | Result |
|---|---|---|
| Three scratchpad copies are byte-identical to the branch | `git show origin/proposal/integration-fidelity-harness:docs/proposals/<f> \| git hash-object --stdin` against `git hash-object <copy>` | Verified. `ea8cd5ae…` (brief), `3c18609c…` (harness), `b6d7d215…` (completion model) |
| Implementation master `744b2fc`; research master and pin `d1d350d` | `git rev-parse master`; `git ls-tree master research`; `git -C ../research rev-parse master` | Verified |
| Merge-base `727722c`; branch adds only the three files | `git merge-base master origin/proposal/…`; `git diff --name-status 727722c origin/proposal/…` | Verified. Three `A` entries, no other change |
| `scenarios/` holds 13 JSON files plus `README.md` | `ls scenarios/` | Verified |

Master is 12 commits ahead of the merge-base (`git rev-list --count 727722c..master`). Two of those
commits are load-bearing for this review and are described in the Duplication section.

## Verdict

**ADOPT WITH CUTS.**

The cuts are large. They remove the harness as a standing mechanism and leave three concrete pieces
of work: a one-time coverage gap table, two new headless scenarios (J3 and J6), and a two-class
extension to the existing adversarial-review finding vocabulary. A reader who prefers the word
`REJECT` for that outcome is not contradicted by this review; the actions are identical either way.
`ADOPT WITH CUTS` is used because the proposal contains material worth keeping and `REJECT` would
discard it.

### Strongest argument for adoption

The scenario suite grew by accretion, one or two scenarios per story, and no document asks which
**player journeys have no scenario at all**. `scenarios/README.md` describes what each of the 13
files does; nothing states what is not covered. The J1–J6 table is a coverage gap analysis over
that suite, and it identifies two real gaps that survive every attack in this review:

- **J3** (combat → loot → equip → later combat). No scenario asserts that an equipment change alters
  a later production combat outcome. `1005-doll-and-shop.json` equips, unequips and ground-drops,
  and walks into the next mission, but its assertions end at `assert_inventory` / `assert_shop`
  container contents. The mission-stage scenarios that could observe a combat outcome
  (`0155-*`, `0156-*`, `0159-*`) never touch equipment. The two halves are in different stages,
  which the README states are disjoint and never mixed in one file.
- **J6** (multi-mission continuity, repeated). `0163-mission-to-town.json` crosses one boundary pair
  once. No scenario repeats it, so no witness exists for state that decays across two or more
  crossings rather than one.

This is a genuine contribution and it does not require a harness. It requires two scenario files in
a directory that already exists, run by a gate that already runs at every landing.

### Strongest argument against adoption

The proposal's two normative rules and its execution machinery already exist on master, and its
central new artifact is the thing the review brief names as grounds for rejection.

- The **production-value rule** (proposal lines 94–104) is `AGENTS.md` "Coverage and witness rules"
  rule 4, added to master after the branch base.
- The **boundary mutation rule** (lines 106–112) is `AGENTS.md` rule 2, and the casebook case
  *Mutation kill does not validate the fixture* is strictly stronger than it.
- The **evidence levels M and I** (lines 44–54) are `scenarios/README.md` lines 40–52, where the
  `synthetic` / `install` distinction is already stated with a sharper epistemic boundary.
- The **fidelity lane** (lines 137–141) is `pipeline/check-scenarios.sh`, and the proposal's cadence
  wording would weaken it from unconditional to conditional.
- The **harness manifest** (lines 114–129) is a second index over `scenarios/`. The review brief,
  line 38: *"A renamed campaign sweep or a second index over scenarios should be rejected."*

After removing these, what remains is not a harness.

## Attack 1 — Duplication

The proposal was authored against `727722c`. Master has since acquired the two commits that make
most of the proposal's normative content redundant. This is not an authoring defect; it is the
state the proposal must now be judged against, per the instruction to review against master as it
stands.

`git diff --name-status 727722c master` returns 17 entries. Two matter here:

- `A docs/HARNESS-CASEBOOK.md` (77 lines, `wc -l docs/HARNESS-CASEBOOK.md`)
- `M AGENTS.md` (268 lines to 180, `git show 727722c:AGENTS.md | wc -l` against `wc -l AGENTS.md`)

### 1.1 The production-value rule is `AGENTS.md` rule 4

Proposal, lines 96 and 100:

> A witness is invalid if its observation is derived independently from the same inputs rather than
> taken from the production value that the player-facing path produced.
>
> If a doll composes equipment into a figure, assert the composition result carried out of that
> production composition, not a second call to a helper that ought to produce the same figure.

Master `AGENTS.md` line 97, present since `d8078a9`:

> **Observe the production result.** A witness carries out the value produced by the production
> path; it does not recompute a second value that ought to equal it.

`docs/HARNESS-CASEBOOK.md` lines 59–67 carry the incident the rule was written for, and it is the
same doll composition the proposal uses as its example: a headless `figure` witness computed
`equipmentSlots(mw.currentFigureEquipment())` while the doll composed from `member.Worn`.

The proposal's version adds nothing. It is longer, it is non-normative, and it lives in a document
an agent does not load at startup, while the existing rule is in the repository's always-loaded
instruction file.

### 1.2 The boundary mutation rule is `AGENTS.md` rule 2 plus a weaker form of casebook rule 3

Proposal, lines 108 and 112:

> change the production line or constant whose regression the witness claims to catch and verify
> that the journey fails for the intended reason
>
> A mutation that changes a stand-in path the real journey never reaches proves nothing.

Master `AGENTS.md` lines 92–96:

> 2. **Expected values are independent.** A test must not derive its expectation from the constant,
>    formula, helper, or state path it is testing. Mutate the production line a maintainer would
>    actually change.
> 3. **The fixture must install production state.** Killing an assertion proves the assertion
>    matters; it does not prove the fixture reproduced the live state read by the reverted line.

Rule 3 is a defect class the proposal does not contain. The casebook case behind it (lines 49–57)
records tests that passed against both the broken and the corrected live-state interaction while the
mutation reddened them, so the mutation kill was satisfied and the fixture was still wrong. The
proposal's mutation rule, adopted as written, would license exactly that outcome: it asks only that
the journey fail for the intended reason, not that the fixture install the production state the
reverted line reads.

Adopting the proposal's weaker wording alongside the stronger existing rule creates two rules about
the same subject in two documents, one of which is silent on the failure mode the other exists for.

### 1.3 Evidence levels M and I are `scenarios/README.md` lines 40–52

The README states the distinction and its limit:

> They prove different things. `synthetic` proves mechanism: the rule fired, the state changed, the
> order is right. It cannot prove this build reads the original's bytes correctly, because the bytes
> were authored here from the same belief the decode holds. `install` proves fidelity of ingestion:
> the real archive opens, the real map loads, the counts come out. Neither states what the original
> game does; only a research claim does.

The proposal's M and I paragraphs (lines 44–54) say the same thing in fewer words and omit the
reason `synthetic` cannot prove a decode — that the fixture bytes were authored from the belief
under test. That reason is the load-bearing half.

The only new level is **J**. J is not an asset-source classification, which is what M and I are; it
is a scope classification. Putting a scope label in the same enumeration as two source labels is a
category mix: a J scenario is necessarily also an I scenario, because the front-end stage runs on
`install` only (`scenarios/README.md` line 79). The manifest would therefore record `evidence: J`
for a scenario whose asset source is `install`, losing the fact the M/I axis exists to record.

### 1.4 The fidelity lane is `pipeline/check-scenarios.sh`

The script runs every scenario in `scenarios/` against a lawful install, prints the count it
selected before running any, prints the checkout SHA, and exits 2 when it has no asset root. Its
header states the failure it was written for: two scenarios that had refused to load for two days
while the repository's own gate chain stayed green.

The seat runs it at every landing (`<seat>\AGENTS.md`, "This seat's own gate, at
every landing"). The proposal, line 141:

> This lane is appropriate before a release candidate and after changes to cross-cutting state,
> persistence, campaign routing, scenario infrastructure, map loading, or hashed simulation
> semantics. It need not become a mandatory cost on every tiny documentation or leaf-format change.

That is a cadence reduction against current practice. The failure `check-scenarios.sh` was written
for was caused by a change that fits the proposal's exemption exactly: commit `ac7592a` replaced em
dashes in drawn string literals, which is a leaf presentation change touching no cross-cutting
state, no persistence, no campaign routing, no scenario infrastructure, no map loading and no hashed
simulation. Under the proposal's cadence the lane would not have run, and the two dead scenarios
would have stayed dead.

This is the single most damaging line in the proposal. It is a non-goal violation as well: the
proposal claims not to change existing gates, and this sentence changes one.

### 1.5 The manifest is a second index over `scenarios/`

The manifest's eight columns are `id`, `scenario`, `evidence`, `boundaries`, `claims`,
`divergences`, `observations`, `owner_intent`.

`scenarios/README.md` is 420 lines (`wc -l scenarios/README.md`) and already carries, in prose, the
content of `scenario`, `evidence`, `boundaries`, `claims`, `divergences` and `observations` for the
scenarios that need them. Its entry for `1013-world-map-one-click.json` (lines 272–278) cites
`TOWN-121` and `DIV-135` by id and states what the file observes and why. Its entry for
`0163-mission-to-town.json` (lines 186–189) states the boundaries crossed and the fields asserted on
both sides.

The manifest would restate this for 13 files plus new ones, in a second location, with no mechanism
keeping the two consistent. `docs/DIVERGENCES.md` lines 30–33 record what happened the last time a
count was written in two places:

> No per-table row count is written here. The first version of this header carried three, and the
> next landing made all three wrong within the day.

The `divergences` and `claims` columns are the worst case. A divergence row's id survives a status
change and moves between files: `DIV-135` is now a row in `docs/DIVERGENCES-CLOSED.md` and is named
in `docs/DIVERGENCES.md` only as prose recording the move (`grep -n 'DIV-135' docs/DIVERGENCES.md`
returns two lines, both narrative). `scenarios/README.md` tracked that move and reads
"`TOWN-121`, `DIV-135` closed". A manifest would need the same tracking in a second file, with no
gate keeping the two in step. Nothing in the proposal makes the manifest re-read at a research pin
bump, which is the exact failure `pipeline/check-div-claims.sh` exists for one ledger over.

### 1.6 What the proposal adds after duplicates are removed

| Proposal section | Status against master |
|---|---|
| Problem, Non-goals | Framing. No cost, no addition |
| Build on the existing headless surface | Already policy; `scenarios/README.md` and `AGENTS.md` |
| Evidence levels M, I | Duplicate of `scenarios/README.md` 40–52, weaker |
| Evidence level J | New label, but category-mixed with M/I (1.3) |
| Journey portfolio J1–J6 | **Genuine contribution as a gap table.** J3 and J6 are real gaps |
| State checkpoints | Duplicates the existing `assert_*` vocabulary; the "do not snapshot the whole world" advice is new and correct |
| Production-value rule | Duplicate of `AGENTS.md` rule 4 |
| Boundary mutation rule | Duplicate of `AGENTS.md` rule 2, and weaker than casebook rule 3 |
| Harness manifest | Second index over `scenarios/`; named as reject-grounds by the brief |
| Execution lanes | Duplicate of `check-scenarios.sh`, and weakens its cadence |
| Failure classification | Partial duplicate of P/W/D; **EXPECTATION and INFRA are new and useful** |
| Adoption criterion, suggested pilot | Criterion 1 is already satisfied at HEAD (see Attack 8) |

Five of twelve sections duplicate existing normative text. One weakens an existing gate. One is
named by the brief as grounds for rejection. Two are genuine contributions.

## Attack 2 — Correlated evidence

The proposal states the production-value rule for the **scenario author**. The defect it describes
is live in the **shared snapshot** every front-end scenario reads through, so the rule as worded
cannot reach it. Adding six journeys multiplies the number of assertions flowing through a partly
self-derived snapshot without improving the snapshot.

### 2.1 `Defense` and `Absorption` are re-derived off the map screen

`pkg/game/headless.go` lines 623-624, inside `FrontEnd.HeadlessSnapshot`:

```go
derived, _, _ := mapload.PartySpawnWithTable(member, f.Table)
m.Defense, m.Absorption = derived.Combat.Defence, derived.Combat.Absorption
```

Lines 626-631 overwrite both from the live entity, but only when the screen is `map` and a live
entity id exists. On the **town** screen, and on any screen with no live world, the witness's
defence and absorption are the witness's own call to `mapload.PartySpawnWithTable`.

`grep -rn 'PartySpawnWithTable' --include=*.go pkg/ cmd/ | grep -v _test.go` returns eight lines:
three comments, the definition at `pkg/mapload/start.go:475`, three production call sites
(`pkg/game/world.go:1238`, `pkg/game/chargen.go:312`, `pkg/mapload/start.go:751`), and the witness at
`pkg/game/headless.go:624`. The witness makes a second, parallel call to the same helper rather than
reading what production computed.

This is the proposal's own example, line 100 — "not a second call to a helper that ought to produce
the same figure" — and it is live in the code the proposal proposes to build on.

`scenarios/0152-save666.json` shows the consequence. Step 18 asserts `npc:22` has
`"defense": 22, "absorption": 0` while `"screen": "town"`. Step 35 asserts the same two numbers on
the map screen. The first is a second derivation and the second is a production observation. The
scenario file cannot tell them apart, and neither can a reader of the file.

This is a finding about the repository, not about the proposal. It is reported here because it
determines what the proposal buys: six journeys asserting `defense` across a town boundary would all
be asserting the witness against itself.

### 2.2 `Documents` is a second scan

`pkg/game/headless.go` lines 666-675 count carried items equal to `data.QuestDocumentCode`.
`pkg/game/world.go:3284` runs an independent loop over the same code.
`grep -rn 'QuestDocumentCode' --include=*.go pkg/ cmd/ | grep -v _test.go` returns six lines: a
comment and the constant declaration in `pkg/data/itemcode.go`, two in `pkg/game/world.go`, one in
`pkg/mapload/start.go`, and one in `pkg/game/headless.go`.

`assert_state`'s `documents` field, used by `0163-mission-to-town.json` on both sides of the town
crossing, therefore compares the witness's own count against a literal, not production's count
against a literal.

### 2.3 Where the code already gets this right

`headlessSkills` (`pkg/game/headless.go` lines 781-795) carries a comment stating the rule and the
reason:

> This is deliberately the live panel's resolver, not a second headless-only class mapping. A
> scenario therefore cannot pass while the GUI calls a mage's slot 2 Axe.

`scenarios/README.md` line 212 states it for the pointer step: "The pixel is asked of the production
hit test each time, never computed in the scenario."

The rule is understood and applied where someone applied it. The remaining gap is at named fields in
one function. That is a code defect, not a process gap, and restating the rule in a third document
does not close it.

### 2.4 J4's row and J4's rule disagree

Portfolio row J4, line 71, requires: "production state after load equals the captured persisted
state on named fields." The production-value rule, line 102, requires: "inspect the restored live
field after load, not the serialized bytes alone."

"Captured persisted state" reads as the serialized side. `scenarios/0152-save666.json` steps 38-43
already do the correct thing: `save`, `load @first`, then `same_character_as` against a capture taken
from live state before the save. The row's wording, taken literally, is weaker than the scenario that
already exists.

### 2.5 `same_character_as` is stronger than the proposal's checkpoint advice

`assertHeadlessMember` (`pkg/game/headless.go` lines 1127-1140) implements `same_character_as` as
`reflect.DeepEqual` over the whole `HeadlessMemberSnapshot` with `Entity` zeroed. The struct
(`pkg/game/headless.go` line 238) has 14 fields, including `Appearance`, `Weapon`, `Equipment`,
`Worn`, `SkillXP` and `Skills`, none of which the scenario file names.

The proposal's State checkpoints section, line 92, says: "A checkpoint should name the minimum state
whose corruption would make the journey semantically wrong." Applied to `same_character_as`, that
advice replaces a total-equality comparison with a named-field list. That is a coverage loss: total
equality catches fields nobody thought to name, which is the class of defect a cross-system journey
exists for. The advice is correct for a whole-world golden and wrong for a whole-member one, and the
proposal does not distinguish them.

## Attack 3 — Maintenance cost

Churn is measurable here rather than estimated, because the scenario suite's history is in git.

`git log --oneline -- scenarios/ | wc -l` returns 19 commits. Per-file revision counts, from
`for f in scenarios/*; do echo "$(git log --oneline -- "$f" | wc -l)  $f"; done | sort -rn`:

| Revisions | File |
|---|---|
| 11 | `scenarios/README.md` |
| 5 | `scenarios/1005-doll-and-shop.json` |
| 3 | `0163-mission-to-town.json`, `0154-synthetic-spells.json`, `0152-save666.json` |
| 1 | the remaining 9 files |

`README.md` is revised more often than any scenario. It is already the maintained index, and the
proposal adds a second one with the same update trigger and no mechanism tying them together.

### 3.1 One correct production change cost 121 scenario lines

`git show 938920c -- scenarios/1005-doll-and-shop.json | grep -c '^[+-][^+-]'` returns **121**
changed lines in a 789-line scenario (`wc -l scenarios/1005-doll-and-shop.json`). The commit message
states the cause: a production correction to the town roster cull removed the actor both halves of
the scenario had navigated to, so both halves were re-pointed at a different member and the shop
picker sequence was rewritten.

Production was correct after the change. The scenario changed because the journey's route through
production changed. Churn scales with journey depth: `1005-doll-and-shop.json` has 90 steps, the most
of any scenario, and it is the one that churned.

The proposal bounds journey **count** at six. It does not bound journey **depth**, which is the
dimension this measurement identifies.

### 3.2 Not every scenario revision is churn

`git show 769e25e --stat -- scenarios/` returns 12 insertions and 2 deletions across three files.
Reading the diff, the change **added** `member_count` assertions rather than relaxing existing ones.
That is strengthening, and it is not counted against the harness here.

Of the 19 commits touching `scenarios/`, this review classified the four multi-revision files by
reading each commit subject: 2 revisions were repairs to a witness broken by a correct production
change (`63e7968`, `938920c`), 1 was a strengthening (`769e25e`), and the rest were feature work
adding new steps. Churn is real but it is not the majority of the history.

### 3.3 What would churn after twenty more stories

Named by manifest column, from proposal lines 118-127:

- **`divergences`.** Ids move between `docs/DIVERGENCES.md` and `docs/DIVERGENCES-CLOSED.md` without
  changing. A manifest cell naming a closed id goes stale silently. `pipeline/check-div-claims.sh`
  exists for that failure one ledger over, and it reads `docs/DIVERGENCES.md`, not a manifest.
- **`claims`.** A claim can be amended in place between publication and a pin bump while staying
  active with its id resolving, so a citation stays valid while its content moves. The seat file
  records this happening to `TOWN-186` and `TOWN-187` on 2026-08-19 in a story contract. A manifest
  column of claim ids has the same exposure and the proposal gives it no re-read instruction.
- **`observations`.** Prose restating what the scenario file's own `assert_*` steps state, with
  nothing detecting divergence between the two.
- **`boundaries`.** Derived from `docs/DOMAINS.md`, which each story's `contract.md` already names.
- **`evidence`.** M and I are derivable from the scenario's own `assets` field. J is not derivable and
  is category-mixed with the other two (Attack 1.3).

Five of eight columns either duplicate a maintained source or go stale silently. `id` and `scenario`
are stable and non-duplicative, and those two together are a filename list. `owner_intent` is
optional.

## Attack 4 — False authority

The proposal's prose is careful and its vocabulary is not. Three vectors.

### 4.1 "Fidelity lane" names a cross-root consistency run

Proposal line 139: "Run install-backed curated journeys against both supported asset roots where the
scenario is language-independent."

Passing on both roots establishes that the EN and RU installs agree with each other under this build.
It establishes nothing about ROM1. `pipeline/check-scenarios.sh`'s header records why the two-root
run is usually unnecessary: "ROM.EXE being the same bytes on both (EXP-0123)."

Attaching the project's word for agreement-with-ROM1 to a run that measures cross-root data
consistency is the false-authority vector. Line 154 asserts the opposite in prose. A section heading
is read more often than a paragraph four sections later, and a future agent reporting "fidelity lane
green" would have stated something the lane did not measure.

### 4.2 `FIDELITY` collides with `FIDELITY-DEBT`

`docs/DIVERGENCES.md` defines `FIDELITY-DEBT` as "correct behaviour known, implemented otherwise for
now". It is the debt type.

`PROJECT-COMPLETION-MODEL.md` line 39 defines `FIDELITY` as "required behaviour is reconciled with
current research except accepted deviations/unknowns". It is the best state on axis 2.

The two tokens differ by a suffix and mean opposite things, and the model's own example report prints
them one line apart (lines 127-128):

```text
Systems: 5 FIDELITY / 3 FUNCTIONAL / 1 PARTIAL / 0 ABSENT
Fidelity debt: 12 OPEN FIDELITY-DEBT / 7 ACCEPTED deviations
```

This is a naming collision in a format designed to be read at a glance.

### 4.3 The manifest's `claims` column reads as a verification record

Proposal line 124: "`claims` | research claims relevant to fidelity assertions."

A manifest row pairing a scenario with a claim id reads as "this scenario verifies this claim." It
does not. A scenario verifies that this build behaves as this build's spec says.
`docs/DIVERGENCES.md` states the authority order in its own header: "Authority for ROM1 truth:
promoted claim > provisional research > inference > this code. This code is never evidence of ROM1
behaviour." The manifest schema has no equivalent line. Line 129 comes close but is about the
divergence ledger's primacy, not about what a green scenario means.

## Attack 5 — Process multiplication

The proposal claims, line 9: "The harness proposed here does not add another review layer." Lines
21-30 list non-goals including "add pass-by-pass review prose". Four contradictions.

### 5.1 Two classification vocabularies for overlapping subjects, with no mapping

`AGENTS.md` lines 108-114 define review finding classes **P / W / D**. The proposal, lines 145-152,
defines harness failure classes **PRODUCT / WITNESS / EXPECTATION / INFRA**.

PRODUCT and P have nearly the same definition: "player-visible production behaviour or hashed
simulation state is wrong" against "player-visible behaviour or hashed simulation state is wrong."
WITNESS and W are likewise near-identical. **D has no harness counterpart. EXPECTATION and INFRA have
no review counterpart.** Neither document maps them.

An agent holding both must decide which vocabulary applies and what a cross-vocabulary finding is
called. That is a second classification scheme over an overlapping subject, whatever the non-goals
say. EXPECTATION and INFRA are genuinely useful additions; the correct form is two more classes on
the existing list, not a second list.

### 5.2 The mutation record contradicts the closure rule

Proposal line 110: "It should be recorded tersely in the proposal/story closure if this harness is
adopted."

`AGENTS.md` line 82: "Closure is an as-built record, not a pass-by-pass journal; review history
belongs in git and `pipeline/LOG.md`." The casebook case *Closure is as-built, not a review diary*
records the 1,730-line closure that produced that rule.

A mutation performed once during a witness's introduction and then reverted is review history, not
as-built state. It belongs in the commit message. "Tersely" does not resolve the conflict, because
the rule is about kind rather than length.

### 5.3 The lane cadence is a per-landing judgment with no instrument

Proposal line 141 makes the fidelity lane conditional on whether a change touches "cross-cutting
state, persistence, campaign routing, scenario infrastructure, map loading, or hashed simulation
semantics." Nothing measures whether a change touches those, so the seat decides per landing.

`pipeline/check-scenarios.sh` currently requires no such decision. Replacing an unconditional run
with a judgment is the more expensive of the two, and Attack 1.4 shows the judgment would have
answered "no" for the change that produced the failure the script was written for.

### 5.4 The adoption criterion is a five-clause review gate

Lines 158-166 require a pilot to demonstrate five things. Clause 4 ("The manifest does not duplicate
specifications or divergence prose") and clause 5 ("A harness failure can be triaged to
PRODUCT/WITNESS/EXPECTATION/INFRA without opening an adversarial review chain") are judgments a
person must make and record. That is a review of the pilot, on top of the pilot story's own
adversarial review. The proposal names neither the performer nor the artifact.

## Attack 6 — Coverage illusion

The six journeys observe state: party membership, equipment codes, purse, documents, campaign route,
mission verdict, simulation hash. Every "Required end observation" in the portfolio table (proposal
lines 68-73) is a state predicate. The proposal's non-goals confirm this deliberately, line 29:
"require screenshots or golden rendered frames for ordinary simulation assertions."

The dominant class of shipped defect in this project is drawn output.

### 6.1 The measurement

`docs/hotfix/LEDGER.md` holds 77 rows, counted by matching lines whose first cell begins with a
commit hash:

```sh
python -c "import re;print(len([l for l in open('docs/hotfix/LEDGER.md',encoding='utf-8') if re.match(r'^\|\s*.[0-9a-f]{7}',l)]))"
```

The last 30 rows were classified by hand from the row text. The classification is this reviewer's,
not the ledger's, and a different reader would move one or two rows:

| Class | Rows | Examples |
|---|---|---|
| Drawn output wrong | 13 | `df0f479`, `9e298eb`, `47b220a`, `6d4a3c8`, `de72458`, `ca63ab1`, `45b1962`, `25b046d`, `0262712`, `03bf5a4`, `b9c61ee`, `42fad77`, `828064c` |
| Input routing wrong | 3 | `66ee8bf`, `39ddd8b`, `42fad77` (Book control arm) |
| Simulation or carried state wrong | 6 | `1d5e3a1`, `769e25e`, `e466101`, `795df35`, `bbabb04`, `a15e670` |
| Witness or infrastructure repair | 8 | `63e7968`, `cd75745`, `22b189b`, `60e95a1`, `b1b2b69`, `cfe9510`, `5a17ce4`, `bf98769` |

A keyword scan over all 77 rows for drawing vocabulary returns 36; that scan over-counts, because it
matches rows such as `60e95a1` (an import-graph registration) whose text happens to name a package.
The hand classification of the last 30 is the figure this review relies on.

The six journeys would catch some of the 6 simulation-and-state rows and **none** of the 13
drawn-output rows.

### 6.2 Three defects all six journeys would miss

Each is a defect this project actually shipped, not an invented shape.

**Defect 1 — `47b220a` / `6d4a3c8`: the shop's four command buttons drew numbers with no caption.**
The owner found it in a screenshot on 2026-08-19. `scenarios/1005-doll-and-shop.json` drives the shop
room with 35 `pointer` steps and 5 `assert_shop` assertions and passed throughout, because
`assert_shop` checks `member`, `gold`, `doll`, `worn`, `carries` and `table` — container contents and
a roster index. J3 as specified ends at "equipped state changes a later production combat outcome or
state." A caption is neither.

The same shape recurs at `1017-button-frames`: the seat file records that the story shipped a button
panel of ornate empty plaques, and the adversarial review found blank buttons where the game had
shown `Train / 518` and `EXIT / 100`. It was found by looking at composed output, not by any state
witness.

**Defect 2 — `df0f479` / `de72458`: the school, tavern and character generator's upper-right plaque
did not close flush at its shipped 176-pixel width, and the generator's right-edge seam did not
survive the class column.** `de72458` landed on master on 2026-08-20, one day before this review.
J1's whole path runs through character generation — `scenarios/0163-chargen-mission10.json` drives
`create_character` and asserts state and members. It passed across both defects. A seam is a pixel
column; no journey observes one.

**Defect 3 — `828064c`: the skill school opened with a skill already selected and the Train button
quoting its price before the player had chosen anything.** `townScreen.schoolCell`'s Go zero value is
a real skill. This is presentation **state**, not a pixel, so a witness could in principle see it —
but no journey does. J2 crosses into town and its required end observation is "expected
party/campaign state present in the next mission." The school's opening selection is neither party
state nor campaign state.

A fourth and fifth are available on the same terms: `ca63ab1` (a bolt pointed the wrong way and drew
an undeformed line) and `03bf5a4` (a cast departed from the caster's navel). Both are player-visible,
both were owner-reported, and neither reaches any state a journey checkpoints. Several of these rows
state `pkg/sim` untouched, so the simulation hash does not move either.

### 6.3 Does the proposal describe this limit honestly?

Partly. Line 29 declares golden rendered frames a non-goal, which is a limit stated. But the
proposal's problem statement, line 7, claims "the remaining risk is increasingly composition risk:
several locally correct systems can form a player-visible path that is wrong," and then builds
witnesses that cannot observe the composition this project actually gets wrong. The word
"composition" in `docs/HARNESS-CASEBOOK.md` and in the recent hotfix rows means **image**
composition — a doll figure, a plaque, a page. The proposal uses it to mean system composition.

The honest statement of the limit is not "no golden frames." It is: the harness addresses
state-carrying composition and does not address drawn composition, which is where 13 of the last 30
hotfixes were. The proposal does not say this, and a reader of the portfolio table would not infer
it.

### 6.4 What already covers the drawn class

`pipeline/check-release-tests.sh` runs the install-gated tests, which include pixel-level release
witnesses. `bf98769` added `TestReleaseChargenDetailedSeamColumnsDrawShippedStrips`, which compares
every pixel of `x:[464,480)` on the composed detailed page against the shipped strip. That is the
instrument for the dominant defect class, it exists, and the proposal does not mention it.

Strengthening that gate — more release witnesses over composed output, on both roots — addresses 13
of the last 30 hotfix rows. The journey portfolio addresses at most 6.

## Attack 7 — Failure triage

Four historical defects, classified as the proposal's scheme would report them. Two classify cleanly,
two do not, and one exposes a structural problem with the scheme.

### 7.1 `63e7968` — the em-dash save label. Ambiguous: WITNESS or INFRA

Two scenarios named their save `666 — mission 20 [original]` with an em dash. Commit `ac7592a`
correctly replaced em dashes in drawn string literals with hyphens, because the fonts are the game's
own bitmap atlases indexed by byte. The label formatter stopped producing one and both scenarios
matched no row on the load screen. Production was correct throughout.

- **WITNESS** fits: "the journey observes the wrong value, cannot reach the production state, or
  encoded a stale assumption." The scenario encoded a stale assumption.
- **INFRA** fits: "asset root, runner, environment or harness machinery failed before the behaviour
  was exercised." The load failed before the journey's behaviour was exercised at all.

The classification does no work here, because the remedy is the same one-line edit under either
label. That is not a defect in the scheme so much as evidence that its discriminating power is lower
than the four labels suggest.

### 7.2 `22b189b` / `cfe9510` — `MercenaryType = 1` collision. Clean WITNESS, and the harness adds nothing

A release witness set `MercenaryType = 1` on 2026-08-14 as an arbitrary nonzero marker. Story 1006
gave 1 a meaning on 2026-08-17 and the witness went red on both roots for two days while every gate
printed green. Production was correct.

Clean **WITNESS**. But the harness contributes nothing: the defect was found by
`pipeline/check-release-tests.sh`, which was written for exactly this failure, and the failing test is
not a journey.

### 7.3 `1d5e3a1` — the NPC Humans template loadout. Clean PRODUCT, and the harness plausibly catches it

An `NPC`-named Humans template lost its live loadout, and the row states the defect crosses map
placement, joined and hired party identity, original-save restoration, hashed `sim.Entity` state and
simulation form 54/55. This is the archetypal cross-system defect the proposal exists for.

Clean **PRODUCT**, and J2 or J6 asserting worn equipment by stable code across a crossing would
plausibly go red. This is the proposal's best case.

It is also already covered in shape: `scenarios/0152-save666.json` step 29-30, 36-37 and 42-43 use
`same_character_as`, which `reflect.DeepEqual`s the whole member snapshot including `Equipment` and
`Worn`. An existing scenario is the witness class that catches this. The gap the harness would close
is coverage of more crossings, not a new witness kind.

### 7.4 `769e25e` — item-spell training. PRODUCT and EXPECTATION are indistinguishable from harness output

The hotfix changed training attribution: item-borne casts stopped paying a half-mana award at
release, training began following accepted effects, and delayed kills paid the victim's surviving
attribution. XP values that any journey checkpoints would move. The same commit advanced the research
pin to `02a1403` for `EXP-0191`.

A journey asserting XP after combat goes red. The classification:

- **PRODUCT** if the new behaviour is wrong;
- **EXPECTATION** if "the expected behaviour conflicts with current research" — which is exactly what
  happened, because a new claim arrived and the old expectation was superseded.

The proposal, line 145: "A harness failure is classified before work begins." Distinguishing PRODUCT
from EXPECTATION here requires reading `EXP-0191`'s published claim against the pin, comparing it to
the assertion, and deciding which is stale. That is the work. **The scheme's timing rule is
unsatisfiable in the case where the scheme matters most**, and this is the ordinary case: any pin bump
that lands beside a behaviour change produces it.

### 7.5 The structural problem: the scheme triages output, not defects

`828064c` (school opens with a skill selected), `df0f479` (plaque seam), `47b220a` (captionless
buttons) and `de72458` (chargen seam) are all `PRODUCT` by the scheme's own definition — player-visible
production behaviour is wrong. None of them would ever be classified, because no journey fails.

PRODUCT/WITNESS/EXPECTATION/INFRA is a triage of harness **output**. A report using it says nothing
about defects the harness cannot see, and the proposal's Failure classification section does not say
so.

## Attack 8 — Completion model

The model was applied to the repository at master `744b2fc`, pin `d1d350d`, using repository evidence
only. Places where the model forces a guess are marked **GUESS** and counted.

### 8.1 Axis 1 — campaign reachability: no instrument exists

The model's four sub-metrics and what the repository can answer:

| Sub-metric | Instrument | Answer |
|---|---|---|
| missions reachable | none | **GUESS 1** |
| missions completable | none | **GUESS 2** |
| required town/world-map transitions working | `scenarios/0163-mission-to-town.json`, `1013-world-map-one-click.json` | 2 transitions witnessed, of an unnamed total |
| campaign terminal state reachable | none | **GUESS 3** |

The two campaign instruments do not answer the model's question, by the model's own definition. Model
line 19: a node passes when it is "reachable through production game flow… and does not require a
developer-only bypass."

- `pipeline/check-milestone.sh` loads each map for one tick and counts unrunnable script nodes. It
  does not route.
- `implementation/scripts/campaign-sweep.sh` drives each mission unattended. Its own header states:
  "IT IS AN INSTRUMENT AND NOT A VERDICT… `undecided` is not one either: a mission nobody plays is a
  mission nobody finishes."
- Both drive `cmd/missionrun`, whose doc comment says it "starts the mission exactly as the front-end
  does" but takes `-mission N` directly. Mission construction is production; the routing to it is
  bypassed.

The model's axis 1 is defined so that both existing campaign instruments are the developer-only
bypass it excludes. That is a correct and useful definition, and it means axis 1 currently reports
nothing.

What can be stated: 28 campaign missions load and drive on both roots, with 59 unrunnable script
check nodes per root and 0 unrunnable instants. Command:

```sh
python -c "print(sum(int(l.split('cannot run')[1].split('x')[0]) for l in open('pipeline/milestone-baseline.txt') if l.startswith('en') and 'cannot run' in l))"
```

returns 59; the mission list from the same file has 28 `mNNN` entries plus one `drive` row.

### 8.2 Axis 2 — system completeness: the four labels do not fit the nine domains

`docs/DOMAINS.md` names nine domains. The model's labels are defined in player-path terms:
`ABSENT` — "player path does not exist".

Domains 1 (Assets) and 2 (Sim Core) have no player path by construction. `docs/DOMAINS.md` says
Assets "knows no game rule" and Sim Core "never reads a file". Classifying either as `ABSENT` is
literally correct and semantically wrong. **GUESS 4 and GUESS 5.**

Domains 3, 4 and 5 are described in `docs/DOMAINS.md` as "cohesion regions inside the determinism
wall" whose boundary is "subject ownership, not a package border". They have no separate player path
either; a player experiences Combat & Magic through Client. **GUESS 6.**

Attempting the remaining four with repository evidence:

| Domain | Attempt | Basis |
|---|---|---|
| 6 Campaign & Scripts | `PARTIAL` | 59 unrunnable check nodes per root; `docs/DIVERGENCES.md` carries open trigger rows |
| 7 Town & Economy | `FUNCTIONAL` | shop, tavern, school and gates are all driven by shipped scenarios; `DIV-165`, `DIV-166`, `DIV-167` open |
| 8 Client | `FUNCTIONAL` | 13 of the last 30 hotfixes were drawn-output defects, so it is not `FIDELITY` |
| 9 Persistence | `FUNCTIONAL` | `0152-save666.json` round-trips; no `FIDELITY-DEBT` row names save format |

Each of these four required a judgment the model does not define: how many open rows move a domain
from `FIDELITY` to `FUNCTIONAL`, and how many move it to `PARTIAL`. The model says `FUNCTIONAL` means
"a known required behaviour is missing" is false and `PARTIAL` means it is true, but "required" is
undefined. **GUESS 7.**

### 8.3 Axis 3 — fidelity reconciliation: the best-instrumented axis, and still five guesses

`bash pipeline/check-div-claims.sh` prints:

```text
check-div-claims: selected 121 live row(s) of 121 (0 closed), citing 175 distinct claim id(s)
34 row(s) cite a claim carrying a retraction row.
```

Cross-tabulating type against status over the same 121 rows:

| Type | OPEN | ACCEPTED |
|---|---|---|
| UNKNOWN | 57 | 7 |
| FIDELITY-DEBT | 28 | 0 |
| DEVIATION | 11 | 11 |
| CONFLICT | 0 | 4 |
| HOTFIX | 3 | 0 |

`docs/DIVERGENCES-CLOSED.md` holds 22 rows.

Mapping these onto the model's five reporting lines:

- **"open known fidelity debt"** → 28. Clean.
- **"accepted deliberate deviations"** → 11 `DEVIATION/ACCEPTED`, or 15 including the 4
  `CONFLICT/ACCEPTED`. A CONFLICT is an owner directive contradicting a promoted claim, which is a
  deliberate deviation by meaning and a different type by the ledger. **GUESS 8.**
- 11 `DEVIATION/OPEN` rows have no line in the model at all. They are deliberate differences that are
  not yet accepted. **GUESS 9**, and it is 9 percent of the ledger.
- 3 `HOTFIX/OPEN` rows have no line in the model. **GUESS 10.**
- **"authored behaviour where research is silent"** → 64 `UNKNOWN`, of which 7 are `ACCEPTED` and 57
  `OPEN`. The model gives no place for that split, and the model's own rule 3 ("Never count an
  UNKNOWN research row as known ROM1 fidelity") suggests the two should not be merged. **GUESS 11.**
- **"closed divergences"** → 22. Clean.
- **"research claims that are provisional or retracted and still affect live expectations"** → the
  script reports 34 rows citing a claim carrying a retraction row and states in its own output that a
  claim is usually retracted in part and that it cannot judge whether the retracted clause is the one
  the row leans on. 34 is an upper bound, not the model's quantity. **GUESS 12.**

Five distinct guesses on the axis where the repository has the most machinery. The cause is that the
model's five reporting lines and the ledger's five types are different partitions of the same rows.

### 8.4 Axis 4 — journey integrity: not applicable

The proposal is not adopted, so there is no curated portfolio. The nearest existing figure is
`pipeline/check-scenarios.sh`'s selection and pass count, which is a count of scenario files — the
thing the model's line 61 says not to report.

### 8.5 Release level

Classifying under R0-R5:

- **R0** is satisfied.
- **R1** requires "at least one production player journey crosses menu/gameplay and reaches a real
  mission outcome." `scenarios/0163-chargen-mission10.json` crosses menu to mission through
  production. `scenarios/0163-mission-to-town.json` drives a mission to its verdict and crosses into
  town. R1 is satisfied on repository evidence.
- **R2** requires "every required campaign mission is naturally reachable and completable without
  developer-only bypasses." Unmeasurable — see 8.1. **GUESS 13.**

The honest classification is **R1, with R2 unmeasured**. The model cannot place the project higher
without an instrument that does not exist, which is the model working correctly.

### 8.6 Result of the adoption test

The model's own adoption test (line 148): "If two independent agents cannot classify the same current
state within one adjacent category without inventing facts, refine the definitions before adoption."

This reviewer reached R1 with 13 forced guesses. Axis 1 is unmeasurable, axis 2's labels do not fit
6 of 9 domains, axis 3's partition disagrees with the ledger's on 3 of 5 lines, and axis 4 is
inapplicable. A second agent would very likely reach the same **release level** — R1 versus R2 is
one adjacent category, so the test passes on that clause — while producing a different axis-3 report,
because guesses 8 through 12 are free choices.

The model passes its own adoption test on release level and fails it on the axes. The axes are where
the model claims its value, since its opening argument is that a single number is the wrong measure.

## Pilot design task

All 13 scenarios pass on the EN root at master `744b2fc`:

```text
bash pipeline/check-scenarios.sh "<seat>/gameversions/en"
check-scenarios: selected 13 scenario(s)
check-scenarios: ok (13 of 13)
```

exit code 0, captured with `out=$(...); code=$?`. The three smallest existing scenarios that could
serve J1, J2 and J4 are below. Each already exists, so the proposal's adoption criterion 1 is
satisfied today with no work at all.

### J1 — `scenarios/0163-chargen-mission10.json` (9 steps)

**Production boundaries crossed.** Main menu brooch button, picker row selection, the chargen gate
(`FrontEnd.newGameChargen`, `pkg/game/frontend.go:924`), the generation screen's own focus/Enter/typed
dispatch, party construction, mission construction, map screen. Real boundaries, and it presses no
control the scenario invented — `scenarios/README.md` lines 138-140 state that.

**Currently strong.** The skill spread. The file names the skill by label (`"skill": "Air"`) and
asserts `Air: 10` with the other five schools at 0. The level 10 is production's, not the file's.
`headlessSkills` resolves the names through `ui.CharacterSkillName`, the live panel's own resolver,
so the assertion cannot pass while the GUI names a mage's slot differently.

**Currently missing.** *The scenario does not assert which mission it entered.*
`HeadlessStateAssertion` (`pkg/game/headless.go` line 208) has exactly five fields: `Screen`,
`Purse`, `Documents`, `MemberCount`, `Members`. `HeadlessState` (line 219) has no mission field
either. The `mission` field belongs to the mission stage's scenario header (line 83), and the two
stages are disjoint.

J1's required end observation, proposal line 68, is "live controllable party in the **selected**
mission." **That observation is not expressible in the current front-end scenario vocabulary.** This
is the single most valuable finding of the pilot task, and it applies to four existing scenarios, not
one: `0163-chargen-mission10.json`, `0163-mission-to-town.json`, `1013-world-map-one-click.json` and
`1020-abort-witness.json` all land in a mission and none of them names it.

**Production mutation.** `pkg/game/frontend.go:928`, inside `newGameChargen`:

```go
n := f.Maps[row].Mission
```

change to `n := 20`. Expected: `scenarios/0163-chargen-mission10.json` still passes, because
`{"screen": "map"}` and an Air-10 spread are true of mission 20 as well. **Not run** — this review is
read-only. The prediction does not depend on running it: the assertion vocabulary contains no field
that distinguishes the two missions.

**Killed by a cheaper test?** Partly, and not decisively. `grep -rn 'newGameChargen' --include=*_test.go
pkg/` returns 15 lines in three files. `pkg/game/chargen_test.go` witnesses which rows the gate claims
and that `Begin` composes without error; `pkg/game/sessionend_test.go` and
`pkg/game/worldmapsession_test.go` witness session clearing. None asserts the mission number the
returned opener opens. Whether the mutation reddens one of them by accident, through a fixture whose
map list holds only mission 10, is unresolved by reading; the seat can settle it by running the
mutation above.

### J2 — `scenarios/0163-mission-to-town.json` (28 steps)

**Production boundaries crossed.** Original-save load, live mission, four notice advances to the
verdict, the town, the tavern's dialogue chain, the gates, world-map departure, the next mission's
construction and map screen. Three crossings in one file.

**Currently strong.** Purse and documents across the first crossing: step 2 asserts
`{"screen": "map", "purse": 600, "documents": 1}` and step 13 asserts
`{"screen": "town", "purse": 1600, "documents": 1, "member_count": 2}`. The purse changes by a
production amount the file did not compute. Steps 14 and 27 use `same_character_as`, which
`reflect.DeepEqual`s the whole member snapshot.

**Currently weak.** *The third crossing checks almost nothing.* Step 26 is
`{"command": "assert_state", "state": {"screen": "map", "member_count": 2}}` — no purse, no
documents, and no mission number. The purse and documents are asserted twice on the mission-to-town
side and zero times on the town-to-mission side, which is the crossing J2's own row is about.

**Production mutation.** Zero the purse when the next mission's world is constructed. A precise line
is not named here because the transfer was not traced to a single statement in this review; the seat
should locate it in `MissionOpenerWith`'s path and set the destination purse to 0. Expected: steps 2
and 13 pass unchanged (they precede the mutation), step 26 has no purse clause, and the scenario
passes green with the player's gold destroyed.

**Killed by a cheaper test?** Unknown for the front-end transfer path.
`grep -rln 'Purse' --include=*_test.go pkg/ | wc -l` returns 22 test files, most in `pkg/sim`, which
covers serialization rather than the town-to-mission handover. This is the one J-shaped gap in the
existing suite that a two-field edit closes.

**Cheapest fix, no new mechanism.** Add `"purse": 1600, "documents": 1` to step 26 of the existing
file. One file, two fields.

### J4 — `scenarios/0152-save666.json` (38 steps)

**Production boundaries crossed.** Original-save load, notices, town, tavern dialogue, gates, next
mission, member selection, production `save` through the game menu, production `load` of the just
written slot, and re-assertion.

**Currently strong.** Steps 38-43 do exactly what the proposal's production-value rule asks: `save`,
`load @first`, then `same_character_as` against `restore_complete` and `town`, which are captures of
live state. `assertHeadlessMember` compares the whole `HeadlessMemberSnapshot` — 14 fields, including
`Appearance`, `Weapon`, `Equipment`, `Worn` and `SkillXP` — with only `Entity` zeroed. Nothing in the
file names those fields, so the comparison catches fields nobody thought of.

**Currently missing.** The purse and documents are never asserted anywhere in this file. Step 16 is
`{"screen": "town", "member_count": 2}` and step 41 is `{"screen": "map", "member_count": 2}`. Neither
carries a `purse` or `documents` clause. Also, `same_character_as` is per-member, so no state-level
field survives the round trip under any assertion.

Additionally, per Attack 2.1, the `defense: 22, absorption: 0` asserted at step 18 on the **town**
screen is a second derivation through the witness's own `mapload.PartySpawnWithTable` call, while the
same numbers at step 35 on the **map** screen are the live entity's. The file reads identically in
both places.

**Production mutation, and why the obvious one is worthless.** The obvious mutation — drop the purse
from the save writer — **is already killed decisively by cheaper tests.** `pkg/sim/binary_test.go`
carries explicit `strippedOfTheCarryAndPurse` fixtures and compares digests against them at lines
2638, 2837-2838, 2918-2919, 3030-3031 and 3125. A serialization-omission mutation reddens `go test
./pkg/sim` with no install at all.

The mutation worth running is one level out and is the casebook's own defect class: make a **reader
after load re-derive a persisted fact** instead of reading it. `PartyMember.WeaponMaterialized` is
the field, and `docs/HARNESS-CASEBOOK.md` lines 25-37 record the three sites that once did this.

**Killed by a cheaper test?** Yes, now. `grep -rln 'WeaponMaterialized' --include=*_test.go pkg/`
returns 8 files, including `pkg/game/weaponlatch_scan_test.go`, whose name states that it enumerates
the population. Story 1005 closed this class with a population sweep, which is `AGENTS.md` rule 1
working as designed.

### Result of the pilot task

**Existing tests already kill every meaningful mutation on J4, and the J4 row's own wording is weaker
than the scenario that implements it. J4 should not enter a curated portfolio.** That is the answer
the brief asked for explicitly.

J1 and J2 are worth the work, and the work is not a harness:

| Journey | Work | Files |
|---|---|---|
| J1 | add a `mission` field to `HeadlessState` and `HeadlessStateAssertion`, populate it in `HeadlessSnapshot`, and assert it in four existing scenarios | `pkg/game/headless.go`, `pkg/game/scenario.go`, `scenarios/README.md`, 4 scenario files |
| J2 | add `purse` and `documents` to step 26 of `0163-mission-to-town.json` | 1 scenario file |
| J4 | nothing | none |

The two journeys the portfolio identifies as genuine gaps, **J3** and **J6**, are the ones the
proposal's suggested pilot omits. The pilot as proposed (line 170: "Start with J1, J2 and J4 because
the repository already has strong front-end scenario primitives") selects the three rows that already
exist, which is why criterion 1 cannot fail. A pilot that cannot fail its own first criterion
measures nothing.

**The pilot should be J1 (with the mission field), J3 and J6.** J1 because it exposes a vocabulary
gap that strengthens four existing files; J3 and J6 because they are the only rows with no existing
coverage.

## Cost estimate 1 — one-time, to a useful three-journey pilot

Stated in files, commands and review actions, per the brief's instruction.

**Proposal's pilot as written (J1, J2, J4).**

| Item | Count |
|---|---|
| New scenario files | 0 |
| Existing scenario files edited | 2 (`0163-mission-to-town.json` step 26; optionally `0152-save666.json`) |
| Go files edited | 2 (`pkg/game/headless.go`, `pkg/game/scenario.go`) for the mission field |
| Documentation files edited | 1 (`scenarios/README.md`) |
| New Go tests | 1-2, for the new assertion field's parse and failure paths |
| Manifest file | 1, if not cut |
| Mutation runs and reverts | 3 |
| Gate runs | `go build`/`vet`/`gofmt`/`test` on both repos, plus `check-scenarios.sh` and `check-release-tests.sh` on both roots |

This is a small story by this project's standards: 5 files, one new scenario command field, no new
package, no `pkg/sim` change. It is not a harness; it is a scenario-vocabulary story.

**Recommended pilot (J1, J3, J6).**

| Item | Count |
|---|---|
| New scenario files | 2 (J3, J6) |
| Existing scenario files edited | 4 (the mission assertion) |
| Go files edited | 2, plus whatever J3 needs to cross the stage boundary |
| Mutation runs and reverts | 3 |

J3 carries a real risk the proposal does not mention: it requires observing a combat outcome from the
**front-end** stage. `scenarios/README.md` lines 9-17 state the two stages are disjoint and a file
names exactly one. The front-end stage has `wait_ticks`, `wait_until` and `select_member` but no
`assert_unit`, and the mission stage has no screens. J3 therefore needs either a new front-end
assertion reaching live entity combat state, or an accepted split across two files that assert
different halves. **The proposal's own line 36** covers this — "Extend the existing vocabulary only
when the new command or assertion observes a production value that a player journey genuinely needs"
— but the portfolio table gives no hint that J3 is the row with a structural obstacle, and the
suggested pilot avoids it.

## Cost estimate 2 — recurring, per ten ordinary stories

Measured from the scenario suite's own history rather than estimated.

`ls -d docs/[0-9]* | sed 's|docs/||' | cut -d- -f1 | awk '$1+0>=152' | wc -l` returns **38** stories
since the scenario directory was created. `git log --oneline -- scenarios/ | wc -l` returns **19**
commits touching it. That is 0.5 scenario commits per story, or **5 scenario commits per ten
stories** at the current suite size of 13 files.

Classifying those 19 by commit subject:

| Kind | Commits |
|---|---|
| A witness broken by a correct production change, repaired | 2 (`63e7968`, `938920c`) |
| Assertions strengthened alongside a behaviour change | 1 (`769e25e`) |
| New scenarios or new steps as feature work | 16 |

**False-failure investigations run at 2 per 38 stories, or roughly 0.5 per ten stories**, at the
current suite size. One of the two cost 121 changed lines in one 789-line file
(`git show 938920c -- scenarios/1005-doll-and-shop.json | grep -c '^[+-][^+-]'`).

Projecting the recommended pilot: the suite grows from 13 files to 15, a 15 percent increase, so
scenario commits rise to roughly 5.8 per ten stories and false-failure investigations to roughly 0.6.
J6 by construction is a deep multi-crossing scenario, which is the depth class that produced the
121-line churn event, so the variance rises more than the mean.

Projecting the proposal **as written**, with the manifest: add one manifest row per journey, and a
maintenance obligation on five columns that either duplicate a maintained source or go stale silently
(Attack 3.3). Nothing gates the manifest, so it decays like `docs/DIVERGENCES.md`'s header row counts
did — the same day, per that file's own note. Estimated additional recurring cost: **1 manifest edit
per landing that touches any curated scenario, plus one unbounded audit** the first time a reader
notices a stale cell. The audit is the expensive item and its timing is unpredictable, which is the
argument for not creating the file.

The proposal's lane cadence adds one judgment per landing (Attack 5.3), replacing an unconditional
script run that currently costs one command and, measured here, about the time of 13 `go run`
invocations.

## Proposed edits to `docs/proposals/INTEGRATION-FIDELITY-HARNESS.md`

Eleven edits. Six are deletions.

### H1 — Delete the production-value rule (lines 94-104)

Current:

> ## Production-value rule
>
> A witness is invalid if its observation is derived independently from the same inputs rather than
> taken from the production value that the player-facing path produced.
>
> Examples: […]
>
> This is a harness design constraint, not a request for another checker.

Replacement:

> ## Witness rules
>
> This harness adds no witness rules. `AGENTS.md` "Coverage and witness rules" 1-5 already bind every
> witness in this repository, and `docs/HARNESS-CASEBOOK.md` carries the incident behind each. Rule 4
> is the production-value rule and rule 2 is the independent-expectation rule.
>
> One live violation of rule 4 is named here because it determines what this harness can buy:
> `FrontEnd.HeadlessSnapshot` re-derives `Defense` and `Absorption` through its own call to
> `mapload.PartySpawnWithTable` (`pkg/game/headless.go:624`) whenever no live entity is available,
> and `Documents` through its own scan (lines 666-675). Every front-end journey reads through that
> snapshot. Fixing the snapshot is worth more than adding journeys over it.

### H2 — Delete the boundary mutation rule (lines 106-112)

Current:

> ## Boundary mutation rule
>
> For a new J witness, perform one deliberate local mutation during its introduction when practical:
> change the production line or constant whose regression the witness claims to catch and verify that
> the journey fails for the intended reason. Revert the mutation before landing.
>
> The mutation is evidence about the witness, not a permanent test mechanism. It should be recorded
> tersely in the proposal/story closure if this harness is adopted; no mutation diary is required.
>
> A mutation that changes a stand-in path the real journey never reaches proves nothing.

Replacement: delete the section. `AGENTS.md` rules 2 and 3 cover it, and rule 3 covers a failure mode
this wording does not: a mutation kill does not prove the fixture installed the production state the
reverted line reads. Line 110's instruction to record the mutation in `closure.md` also contradicts
`AGENTS.md` line 82 and the casebook case *Closure is as-built, not a review diary*. A mutation
performed and reverted belongs in the commit message.

### H3 — Delete the harness manifest (lines 114-129)

Current:

> ## Harness manifest
>
> If adopted, add a small machine-readable or Markdown manifest beside `scenarios/` with one row per
> curated journey: […]

Replacement:

> ## Index
>
> `scenarios/README.md` is the index. It is 420 lines, it is revised more often than any scenario
> (11 revisions against 5 for the most-revised scenario, `git log --oneline -- <path> | wc -l`), and
> it already carries each scenario's boundaries, cited claims, cited divergences and what it observes,
> in prose. A journey's row in the portfolio table below names its scenario; nothing further is added.

Grounds: the review brief, line 38, names a second index over scenarios as a rejection criterion, and
five of the eight proposed columns either duplicate a maintained source or go stale silently.

### H4 — Rewrite Evidence levels (lines 40-54)

Current, lines 42-54:

> Every harness scenario declares one of these purposes in its adjacent manifest entry.
>
> ### M — mechanism
>
> Synthetic assets. Proves an engine rule and its wiring under controlled state. […]
>
> ### I — ingestion
>
> Lawful install. Proves that production loaders, shipped content and the path under test compose
> against real assets. […]

Replacement:

> A scenario's asset source is already declared in the scenario file's own `assets` field, and
> `scenarios/README.md` lines 40-52 state what each source proves and what it cannot. This proposal
> adds no third source label.
>
> **Journey** is a scope, not a source. A journey scenario runs on `install` by definition, because
> the front-end stage runs on `install` only (`scenarios/README.md` line 79). It is marked by
> appearing in the portfolio table below and by nothing else.

Grounds: M and I duplicate the README, which states the reason `synthetic` cannot prove a decode — the
half the proposal drops. J is a scope label and does not belong in an enumeration of source labels.

### H5 — Rewrite Execution lanes (lines 131-141)

Current, line 141:

> This lane is appropriate before a release candidate and after changes to cross-cutting state,
> persistence, campaign routing, scenario infrastructure, map loading, or hashed simulation semantics.
> It need not become a mandatory cost on every tiny documentation or leaf-format change.

Replacement:

> `pipeline/check-scenarios.sh` is this lane. It already runs every scenario against a lawful install
> at every landing, prints the count it selected before running any, and exits 2 with no asset root.
> **This proposal does not change its cadence.** The failure its header records was caused by a
> presentation-only change to drawn string literals, which is precisely the class a cadence exemption
> would have excused.

Also rename the section heading "Fidelity lane" to "Install lane", per H9.

### H6 — Fold the failure classes into `AGENTS.md`'s existing scheme (lines 143-154)

Current:

> A harness failure is classified before work begins:
>
> - **PRODUCT** […] - **WITNESS** […] - **EXPECTATION** […] - **INFRA** […]

Replacement:

> `AGENTS.md` already defines review finding classes **P**, **W** and **D**. A harness failure uses
> the same three, with two additions this proposal recommends adding to that list rather than to a
> second one:
>
> - **E** — the expectation conflicts with current research, owner intent, or a recorded divergence.
>   Reconcile against authority before changing code.
> - **I** — asset root, runner or environment failed before the behaviour was exercised.
>
> **E cannot always be told from P before work begins.** Any pin bump landing beside a behaviour
> change produces a failure that is one or the other, and separating them requires reading the cited
> claim against the pin. That reading is the work, so the classification is an early output of
> investigation, not a precondition for it.
>
> The classification triages harness **output**. It says nothing about defects no journey observes.

Grounds: PRODUCT and P, WITNESS and W have near-identical definitions; D has no counterpart in the
proposal's scheme and EXPECTATION and INFRA have none in `AGENTS.md`'s. Two vocabularies over an
overlapping subject, with no mapping, is process multiplication whatever the non-goals say.

### H7 — Fix two portfolio rows (lines 68 and 71)

**J1**, current required end observation:

> live controllable party in the selected mission

Add a note to the section under the table:

> J1's observation is not expressible in the current front-end vocabulary. `HeadlessStateAssertion`
> (`pkg/game/headless.go:208`) has five fields — `Screen`, `Purse`, `Documents`, `MemberCount`,
> `Members` — and none names a mission; `HeadlessState` carries no mission either. Four existing
> scenarios land in a mission and none asserts which one. Adding that field is the cheapest
> improvement identified by this review.

**J4**, current required end observation:

> production state after load equals the captured persisted state on named fields

Replacement:

> live production state after load equals the state captured from live production state before the
> save

Grounds: "captured persisted state" reads as the serialized side, which the proposal's own line 102
forbids and which `scenarios/0152-save666.json` already avoids.

### H8 — Replace adoption criterion 1 (line 160)

Current:

> 1. At least three portfolio rows can be satisfied mostly by existing scenario infrastructure.

Replacement:

> 1. At least one portfolio row with **no existing coverage** — J3 or J6 — can be expressed in the
>    scenario vocabulary, extending it by at most one new assertion.

Grounds: criterion 1 as written is satisfied at master `744b2fc` before any pilot work.
`0163-chargen-mission10.json`, `0163-mission-to-town.json` and `0152-save666.json` already satisfy J1,
J2 and J4. A criterion satisfied before the experiment begins cannot discriminate.

### H9 — Rename the lane and add an authority sentence

Current, line 137: `### Fidelity lane`. Line 139: "against both supported asset roots".

Replacement heading: `### Install lane`. Add after line 139:

> A pass on both roots establishes that the two installs' data agrees with itself under this build. It
> establishes nothing about ROM1; `ROM.EXE` is the same bytes on both roots (`EXP-0123`). Research
> remains the sole authority on ROM1 behaviour, and no harness result is evidence of it.

Optionally retitle the document to drop "Fidelity". A section heading is read more often than a
paragraph four sections later, and "fidelity lane green" states something the lane did not measure.

### H10 — State the coverage limit honestly (new section after the portfolio table)

Add:

> ## What this harness does not observe
>
> Every required end observation in the portfolio is a state predicate. The harness does not observe
> drawn output.
>
> Of the last 30 rows in `docs/hotfix/LEDGER.md`, classified by hand, 13 are defects in drawn output,
> 3 in input routing, 6 in simulation or carried state, and 8 are witness or infrastructure repairs.
> The journeys address at most the 6. The instrument for the 13 is
> `pipeline/check-release-tests.sh` and the pixel-level release witnesses it runs, such as
> `TestReleaseChargenDetailedSeamColumnsDrawShippedStrips` (`bf98769`).

Grounds: the problem statement (line 7) claims composition risk is the remaining risk, and the word
"composition" in this repository's recent history means image composition. Declaring golden frames a
non-goal is not the same as stating which defect class the harness leaves uncovered.

### H11 — Change the suggested pilot (lines 168-171)

Current:

> Start with J1, J2 and J4 because the repository already has strong front-end scenario primitives for
> new game, mission-to-town, inventory/shop and save/load paths.

Replacement:

> Start with **J1, J3 and J6**. J1 because its end observation is not expressible today and adding the
> field strengthens four existing scenarios at once. J3 and J6 because they are the only two rows with
> no existing coverage.
>
> J2 and J4 are already satisfied by `0163-mission-to-town.json` and `0152-save666.json`. Each has one
> gap closed by editing the existing file: J2's step 26 asserts neither purse nor documents on the
> town-to-mission crossing, and `0152-save666.json` never asserts the purse at all.
>
> J3 carries a structural obstacle the table does not show: it needs to observe a combat outcome from
> the front-end stage, and the two stages are disjoint (`scenarios/README.md` lines 9-17). Deciding how
> to cross that is the pilot's real content.

## Proposed edits to `docs/proposals/PROJECT-COMPLETION-MODEL.md`

Six edits.

### C1 — Rename the axis-2 top state (line 39)

Current:

> - `FIDELITY` — required behaviour is reconciled with current research except accepted
>   deviations/unknowns.

Replacement:

> - `RECONCILED` — required behaviour is reconciled with current research except accepted deviations
>   and unknowns.

Grounds: `docs/DIVERGENCES.md` defines `FIDELITY-DEBT` as the debt type. `FIDELITY` and
`FIDELITY-DEBT` differ by a suffix and mean opposite things, and the model's own example report prints
them one line apart (lines 127-128). Update line 127 and line 114 to match.

### C2 — Restrict axis 2 to player-facing domains (line 34)

Current:

> Use the project's domain model rather than story directories. For each domain, classify:

Replacement:

> Use the **player-facing** domains of `docs/DOMAINS.md`: 6 Campaign & Scripts, 7 Town & Economy,
> 8 Client, 9 Persistence. Domains 1 (Assets) and 2 (Sim Core) have no player path by construction —
> `docs/DOMAINS.md` says Assets "knows no game rule" and Sim Core "never reads a file" — and domains
> 3, 4 and 5 are cohesion regions inside Sim Core that a player reaches only through Client. Applying
> a label whose `ABSENT` case reads "player path does not exist" to a domain that has none by design
> produces a literally correct and semantically wrong answer.

Also define "required" in the `PARTIAL` case, or the boundary between `PARTIAL` and `FUNCTIONAL` is a
free choice.

### C3 — Make axis 3 a partition of the ledger's own types (lines 47-53)

Current:

> Report the divergence ledger by meaning, not just row count:
>
> - open known fidelity debt;
> - accepted deliberate deviations;
> - authored behaviour where research is silent;
> - closed divergences;
> - research claims that are provisional or retracted and still affect live expectations.

Replacement: the same five lines plus an explicit mapping, because the current five do not cover the
ledger. Measured at master `744b2fc` with `bash pipeline/check-div-claims.sh` and a type-by-status
cross-tabulation of the 121 live rows:

| Ledger cell | Rows | Model line |
|---|---|---|
| `FIDELITY-DEBT` / OPEN | 28 | open known fidelity debt |
| `DEVIATION` / ACCEPTED | 11 | accepted deliberate deviations |
| `CONFLICT` / ACCEPTED | 4 | **unmapped** — an owner directive contradicting a promoted claim |
| `DEVIATION` / OPEN | 11 | **unmapped** — deliberate, not yet accepted |
| `HOTFIX` / OPEN | 3 | **unmapped** |
| `UNKNOWN` / OPEN | 57 | authored behaviour where research is silent |
| `UNKNOWN` / ACCEPTED | 7 | **unmapped** — the model does not split UNKNOWN by status |
| `DIVERGENCES-CLOSED.md` | 22 | closed divergences |

Three of the model's five lines leave 25 of 121 live rows unclassified. State a destination for each,
or the axis-3 report is a free choice made differently by each reader.

For the fifth line, state that `pipeline/check-div-claims.sh` reports an **upper bound**: it printed
"34 row(s) cite a claim carrying a retraction row" and says in its own output that a claim is usually
retracted in part and that it cannot judge whether the retracted clause is the one a row leans on.

### C4 — State that axis 1 has no instrument (after line 27)

Add:

> **No current instrument answers this axis.** `pipeline/check-milestone.sh` loads each campaign map
> for one tick and counts unrunnable script nodes; `implementation/scripts/campaign-sweep.sh` drives
> each mission unattended and its own header states it is "an instrument and not a verdict". Both
> drive `cmd/missionrun`, which takes `-mission N` directly, so both bypass the campaign routing this
> axis is about. Until a production-routing walk exists, this axis reports "unmeasured", not a number.

Grounds: the axis's own definition (line 19) excludes a developer-only bypass, and both existing
campaign instruments are one. This is the model working correctly, and it should say so rather than
leaving a reader to infer that 28 loading maps is 28 reachable missions.

### C5 — Make axis 4 independent of the harness proposal (line 61)

Current:

> Use the curated integration-fidelity journeys if that proposal is adopted.

Replacement:

> Report the pass count of `pipeline/check-scenarios.sh` on each supported root, naming the checkout
> SHA and asset root it prints, and name every failure. If a curated journey portfolio exists, report
> `passed / applicable` journeys separately.

Grounds: `check-scenarios.sh` already produces a number today — "ok (13 of 13)" on the EN root at
`744b2fc` — and axis 4 should not be blocked on an unadopted proposal.

### C6 — Give R2 its instrument (line 77)

Current:

> Every required campaign mission is naturally reachable and completable without developer-only
> bypasses.

Add:

> R2 cannot be claimed until an instrument walks the campaign through production routing. Naming that
> instrument is a prerequisite for the level, not a detail of reporting it.

## Reversal condition

This verdict is `ADOPT WITH CUTS`, meaning the harness as a standing mechanism is not justified and
three pieces of work are. Two observations after the pilot would reverse it toward `ADOPT`:

1. **A J3 or J6 journey catches a defect that a production-path population sweep would not have
   found.** `AGENTS.md` rule 1 requires a story to enumerate producers and readers; the argument
   against this harness is that a completed population sweep already covers what a journey would
   catch, and that the journey's value is regression over time, which `check-scenarios.sh` already
   provides. If the pilot's J3 or J6 finds a **P**-class defect in a subsystem whose own story
   performed a documented sweep, that argument fails and the portfolio is buying coverage the sweep
   rule cannot.

2. **The manifest survives ten stories with no stale cell.** The case against it is that five of its
   eight columns decay silently. If a manifest is built anyway and an audit after ten landings finds
   every `claims` and `divergences` cell current, without a gate enforcing it, the decay prediction is
   refuted and the index is cheap enough to keep.

One observation would push the verdict to `REJECT`:

3. **The pilot's three journeys produce two or more WITNESS or INFRA failures before they produce one
   PRODUCT failure.** The measured baseline is 2 witness repairs across 38 stories at 13 scenarios.
   A curated portfolio that inverts that ratio is costing lane cycles to maintain itself, which is the
   cost side of the review question with none of the benefit.

Each of the three is measurable from the pilot story's own closure and the following ten landings. No
new instrument is required to evaluate any of them.

## Disagreements between the brief and the covering instructions

None material. The brief (line 89) forbids developer-day estimates unless evidence justifies them,
and none are given here. The covering message asked for the strongest arguments and the pilot mapping,
which the brief also requires. Both were followed.

## What could not be checked

- The J1 and J2 mutations were **not run**, per the read-only constraint. Both are named with file,
  line and edit in the pilot section, and neither conclusion depends on running them: J1's rests on
  `HeadlessStateAssertion` having no mission field, and J2's on step 26 having no purse clause. Both
  are readable facts.
- The J2 purse mutation's exact production line was not located. The transfer was not traced to a
  single statement in `MissionOpenerWith`'s path within this review's scope.
- `pipeline/check-release-tests.sh` was not run. `pipeline/check-scenarios.sh` was run once on the EN
  root only; the RU root was not exercised.
- The hand classification of the last 30 hotfix rows is this reviewer's and is stated as such. A
  keyword scan over all 77 rows returns 36 and over-counts; the review relies on the hand
  classification.
- No research claim was opened. The proposal makes no ROM1 fidelity assertion that required one, and
  the only claim id cited here (`EXP-0123`) was taken from `pipeline/check-scenarios.sh`'s own header
  rather than re-read through `go run ./tools/claim`.

## Seat verification, 2026-08-21

Written at the seat, after the review above and separately from it. It records what was re-run here
rather than taken on report, and what that changed. Commands were run from `againrom` or
`againrom/implementation`, at master `744b2fc`, pin `d1d350d`.

### Re-verified without change

- `AGENTS.md` "Coverage and witness rules" 2, 3 and 4 exist on master and reached it in `d8078a9`,
  which is an ancestor of master (`git merge-base --is-ancestor d8078a9 master`). Rule 4's text is
  the proposal's production-value rule. This is the review's strongest argument against adoption and
  it holds.
- `pkg/game/headless.go:624` calls `mapload.PartySpawnWithTable` and assigns `Defense` and
  `Absorption` from its result. Lines 626-631 overwrite both only when the screen is `map` and a live
  entity id exists. `Documents` at lines 666-675 is a second scan.
- `HeadlessStateAssertion` has five fields and `HeadlessState` has no mission field.
- The divergence ledger's type-by-status cross-tabulation: `UNKNOWN` 64, `FIDELITY-DEBT` 28,
  `DEVIATION` 22, `CONFLICT` 4, `HOTFIX` 3, total 121. Reached independently here and identical to
  the review's table.

### The J1 mutation was run, and it separates into two mutations

The review named this mutation and could not run it. It was run here, applied and reverted with the
same edit pair, with `pkg/game/frontend.go` verified byte-identical to master afterward
(`git diff --exit-code -- pkg/game/frontend.go`).

**Mutation 1**, the review's own wording: replace `n := f.Maps[row].Mission` with `n := 20`. The full
suite fails. `TestNewGameChargenClaimsExactlyTheMissionRows` (`pkg/game/chargen_test.go:1163`)
reddens, because a constant `20` also passes the `n <= 0` guard for rows that carry no mission, so
the gate claims loose-map rows. That test witnesses **which rows the gate claims**, not which mission
it opens.

**Mutation 2**, isolating mission identity: keep the guard, then assign `n = 20` after it. Every row
is claimed exactly as before, and every new game opens mission 20 regardless of the row activated.

| Instrument | Result under mutation 2 |
|---|---|
| `AGAINROM_ASSETS=<en root> go test -trimpath -count=1 ./...` | exit 0, 0 failures, with the 34 install-gated tests running |
| `bash pipeline/check-scenarios.sh <en root> 0163` | exit 0, 2 of 2 ok |

A player activates `Mission 10: 10.alm`, mission 20 opens, and nothing in the repository reports it.
The review's conclusion holds and is now measured rather than predicted: the mission-identity
assertion is absent from the front-end vocabulary, and no cheaper test covers it. The J1 work named
in the pilot table is the highest-value item this review produced.

### A ledger filing defect

`docs/DIVERGENCES.md`'s second section is headed "Authored where research is silent" and its own
introduction states `Type UNKNOWN`, plus: "A row here is not evidence that ROM1 behaves differently;
it is evidence that nobody has measured what ROM1 does."

Thirteen of that section's 77 rows are typed otherwise: 6 `DEVIATION` (`DIV-149`, `DIV-150`,
`DIV-160`, `DIV-168`, `DIV-169`, `DIV-170`) and 7 `FIDELITY-DEBT` (`DIV-142`, `DIV-144`, `DIV-147`,
`DIV-153`, `DIV-161`, `DIV-165`, `DIV-167`). Counted by splitting each `| DIV-` row on the pipe
character and reading cell 6 per section. `DIV-149` and `DIV-150` state in their own Reason cells
that they were re-typed from `UNKNOWN` at story `1016`'s pin bump, so those rows are correct and the
section they sit in contradicts them. A reader taking the section header at face value reads thirteen
known divergences as unmeasured silence.

No gate sees this. `check-div-claims.sh` reads retractions and `check-claim-citations.sh` resolves
ids. Neither compares a row's type against the section it is filed under.

### Two seat gates reported a relative asset root as a product failure

One of the completion-model agents invoked `pipeline/check-release-tests.sh` with a relative asset
root, reported 34 failing tests, then re-ran with an absolute path and got 34 passes. The cause is in
both install-backed seat gates: the `-d "$root"` guard runs in the caller's directory, the script
then runs `cd "$impl"`, and the relative root no longer resolves. Reproduced here on the cheaper
gate, which fails legibly where the other fails as 34 red tests:

```text
bash pipeline/check-scenarios.sh "gameversions/en" 0163
check-scenarios: FAIL scenarios/0163-chargen-mission10.json
                     againrom: open gameversions\en\main.res: The system cannot find the path specified.
exit 1
```

Both scripts now resolve the root to an absolute path immediately after the `-d` test, and each
carries the failure in its own header. After the fix, `check-scenarios.sh` accepts the relative root
and prints the absolute path it resolved to, 2 of 2 ok. `check-release-tests.sh` accepts it and
reports 34 of 34 install-gated tests run and passed, 0 skipped. Both still exit 2 with no root, and
with a root that is not a directory.

This is a class the project already names: an instrument whose failure mode looks like a product
failure. It cost one agent a re-run and produced a report of 34 failing tests against a green tree.

### The completion model's own adoption test was run with three agents

`PROJECT-COMPLETION-MODEL.md` line 148 requires that two independent agents classify the same state
within one adjacent category without inventing facts. Three did, on byte-identical briefs, without
access to each other's work: this reviewer, and two agents whose reports are `COMPLETION-MAP-A.md`
and `COMPLETION-MAP-B.md` beside this file.

| Agent | Release level | Axis 2 | Axis 3 live rows | Forced guesses |
|---|---|---|---|---|
| This review | R1, R2 unmeasured | 9 FUNCTIONAL, 0 FIDELITY | 121 | 13 |
| Map A | R2 candidate, R2 not established | 9 FUNCTIONAL, 0 FIDELITY | 121 | 9 |
| Map B | R1, R2 not confirmed | 9 FUNCTIONAL, 0 FIDELITY | 121 | 10 |

The release levels are within one adjacent category, and the axis-2 and axis-3 totals are identical,
including 28 `OPEN FIDELITY-DEBT` and the 34 rows citing a claim that carries a retraction. All three
gave the same reason for not claiming R2: no instrument walks the campaign through production
routing.

The three disagree on where the model is underdefined rather than on the numbers. All three flagged
the `PARTIAL`/`FUNCTIONAL` boundary as undefined and named different domains as the boundary case.
This review named Client and Campaign & Scripts; Map A named Client's `DIV-099` and Campaign &
Scripts; Map B named AI & Orders and Campaign & Scripts. Map A also reported that the model has no
defined answer for domains with no player path of their own, which is this review's edit C2.

The model passes its own adoption test on release level, at three agents rather than two. It fails it
on the axes, which is where its own argument locates its value.

### Reported and corrected outside this repository

Map B reports that `PIPELINE-STATUS.md`'s open item for mission 20's win names two unsupported script
ops. That is a seat document rather than this repository's. It was checked and corrected at the seat:
`bash pipeline/check-milestone.sh` prints no `cannot run` line for `m20` on either root.
