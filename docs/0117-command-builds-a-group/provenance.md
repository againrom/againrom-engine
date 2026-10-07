# Provenance — 0117

Research pin: `87a256d`. Claims are read through `research/tools/claim`; no ledger was opened by
hand.

## What the claims carry

| Claim | Confidence | What this story took from it |
|---|---|---|
| `AI-CMD-033` | **High** on the allocation chain and on each order routine's own final store; **Medium** that the family split is complete | The whole shape of FR-4. A player order resolves the commanded actors, sweeps the commanding player's group list, allocates a **new** group, adds each actor to it and appends it; the AI block's constructor writes **0** into the group's order byte, and the order routine's own last store decides what the group is left at. **Move leaves 4.** |
| `AI-ORDER-010` | **High** on the arm table; **Medium** on the writer enumeration | That order 4 has an arm of its own, distinct from guard's and from stand ground's, and that a group under 1 or 3 never evaluates a per-actor state. This is what makes "leave the new group at 4" a behaviour and not a number. |
| `AI-GUARD-012` | **High** on the post, the leash and both walk-back stores; amended twice, and one clause refuted | Why the walk home is right and must not be weakened: guard is motion without an order, and an actor away from its post with nothing to fight walks back. Read with its amendments — the first clause is superseded (patrol is also motion without a **player** order) and the no-wander clause is refuted outright. Neither amendment touches the post or the walk-back, which is what this story leans on. |
| `AI-GROUP-009` | — | That a group is formed by equality of the map's own group id at load, which is what makes the placed word a *map datum* and so what DD-1 protects. |
| `AI-AUTHOR-015` | — | That "every group on a shipped map is set to order 1 or 3 at load" is about **authored** groups only. `AI-CMD-033` says so in its own words, and it is the sentence that licenses a group at order 4 existing at all. |

## What is ours by choice, and why the claims do not decide it

- **The second id (DD-1).** `AI-CMD-033` says the actors are added to the new group; it does not say
  what becomes of the map's group word on the actor, because nothing in the experiment needed to
  read it. This tree has exactly one field for both meanings and three mission-script arms keyed on
  it, so the two meanings had to be separated here. Which of them keeps the old field is ours.
- **The id allocator (FR-5, DD-2, DD-3, DD-4).** The law allocates heap memory and sweeps the
  commanding player's list first, and a heap allocator reusing a just-freed block is not a rule we
  can restate. What is reconstructed is the *effect* — one live command group per commanded set, no
  unbounded accumulation — by a rule this tree can state and test.
- **The floor being above the script's ids.** No claim asks for it. It exists because in this tree a
  script's group parameter and an actor's group word are one id space, so an unfenced allocator
  could hand a script node a group it never named.
- **One record per distinct owner.** The law hangs the new group off the **commanding player's**
  list. This tree keys a group record by the actor's own owner, and moving an actor's owner would
  move its diplomacy, so a mixed-owner selection becomes one id under several owners rather than one
  record. On a selection of one owner — which is every selection the front end can make today — the
  two are the same thing.

## What is open

- **What the law does with the actor's placed group word.** Unknown; not needed, because DD-1 keeps
  both readings available.
- **The sweep's own predicate.** `AI-CMD-033` names the routine that decides which of the
  commanding player's groups are deleted before a new one is allocated but not what it tests. FR-5's
  reuse rule stands in for it and is stated in this tree's own terms.
- **The other order families.** `AI-CMD-033` names nineteen order slots and four order-byte
  families; ten arms leave the group at 0 and act through a per-actor state instead. This story
  builds one arm. Its Medium half — that the family split is complete — is not leaned on: only the
  move routine's own store is used, and that half is graded High.

## What was removed

Nothing was retracted under this story. Two premises the lane was handed were falsified before any
code was written, and both are recorded in `analysis.md`: the group order is **stored** state frozen
at construction, not a per-tick derivation from the owner.
