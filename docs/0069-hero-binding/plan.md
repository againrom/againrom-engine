# 0069 — plan

## The shape of the change

The binding is a two-line change guarded by one new rule and one new test. Everything the contract
asks for already exists except the answer to *which entity is the hero*, and that answer is
available earlier than the tree currently believes.

## DD-1 — the hero's entity id is a property of the MAP, not of the built world

The world builder emits exactly one entity per placed record, with id equal to the record's index,
skipping none; the start then **appends** the party. Party member *i* therefore receives the id
`placementCount + i`, and that value is computable from the decoded map with nothing built.

This is what dissolves the apparent ordering problem (FR-6). The compile runs before the world
exists, but it does not need the world — it needs a count the map already carries.

Rejected: **compile inside the start.** It would put the compile where the entity count is directly
in hand, and it is wrong on three counts. The start's scripted form deliberately takes an
already-compiled program so the caller keeps the decisions the compile's report forces — which
announcements a mission can raise, and what to do about a script that will not decode (FR-7). Moving
the compile there would take both from the one place that holds them and would compile a second time
for every caller that already had. It also edits a file another lane is live in.

Rejected: **build the world, read the hero's id, then compile.** Two derivations of one world, which
the start's own contract forbids, and it buys nothing DD-1 does not give for free (FR-6).

Rejected: **re-bind the compiled program after the world is built.** A compiled script is immutable
by construction — every accessor hands back a copy — precisely so that a world holding one can rely
on what validation established. Mutating a validated program to patch in a reference is the thing
that design refuses.

## DD-2 — the rule is published by the assembling tier, and is the only spelling of it

`pkg/mapload` gains one exported function, in a **new file**, answering "what entity id does the
start give party member *i*". The mission path calls it to fill the compile's hero field.

The start's own loop is **not** rewritten to call it. That would be the tidier factoring and it is
declined deliberately: the start is contended by a live lane, and the agreement between the two is
better secured by a test that fails when they diverge than by a shared expression that a merge can
silently take apart. FR-5 asks for one rule; a pinned equality is a rule, and it is the one that
keeps failing if the start's id assignment ever changes.

## DD-3 — presence is party size, not entity existence

The compile's hero-present flag is set from *whether the mission was started with a party*, not from
whether the computed id names an entity. The two agree today, and they answer different questions: a
party of none must bind nobody (FR-3), and entity zero is a real entity, so no id value is free to
mean "none".

## DD-4 — ordinal 1 only, unchanged

The resolver already binds ordinal 1 and refuses the rest. Nothing about the band changes; this
story only makes the field it reads non-empty. FR-4 is therefore satisfied by existing behaviour and
is verified rather than built.

## DD-5 — the byte form does not move

The compiled check's unit id and its presence flag are already encoded, at fixed widths. Only the
values change, and only on the mission path. No version bump, and no digest pinned in this tree is a
literal that moves — every digest comparison in the tree is world-against-world (FR-9, P-2).

## DD-6 — the map-inspection tool is left alone

It holds no party, so it has no hero to bind and its report stays honest (FR-10). Making it accept a
synthetic party would change a developer tool's meaning from "what a map says" to "what a map would
say under an invented world".

## DD-7 — the comment that stopped anyone looking is corrected with the rule

The binder's own doc asserts that nothing in the shipped corpus fails to resolve. That is a
compression of a research claim about the **original's** builder, which has a live player list, into
a sentence that reads as a statement about this build — where it is false, by 196 references. It is
corrected in the same task that publishes the id rule, because a wrong comment beside a correct fix
is how the next reader concludes the fix was unnecessary.

It rides on T1 rather than T2 for a mechanical reason: T1 already opens that package, and the file
holding the comment is not one another lane is live in.

## Files

| File | Change |
|---|---|
| `pkg/mapload/party.go` | **new** — the id rule (DD-1, DD-2) |
| `pkg/mapload/script.go` | comment only — the corpus-resolution claim (DD-7) |
| `pkg/mapload/party_test.go` | **new** — the rule, and its equality with the start (AC-6) |
| `pkg/game/mission.go` | the compile's refs gain the hero and the presence flag (DD-3) |
| `pkg/game/mission_test.go` | the mission-path criteria (AC-1..AC-5, AC-7, AC-8) |

`pkg/mapload/start.go` is **not** edited. `pkg/sim` is not touched.

## Risks

**R-1 — a wrong id binds a wrong entity, silently.** A reference that resolves to the wrong entity is
worse than one that resolves to nothing: it measures something, so nothing reports it. AC-2 and AC-6
are aimed here — AC-2 asserts the bound entity by its *cell*, which no off-by-one survives.

**R-2 — the digest of every started mission moves.** That is intended and is the story, but it means
any test pinning a started world against a stored value would fail. None does; the tree's mission
digests are all compared world-against-world. AC-8 fences the half that must not move.

**R-3 — the false win looks like a pass.** A test asserting only "the mission can be won" passes both
before and after this story. AC-3's first half — *not* won while the hero is away — is the half that
discriminates, and it must be written first.

## Success criteria

**SC-1** Mission-10-shaped map: zero unresolved hero-band references with a party, and the same map
with no party still reports them all (FR-1, FR-3).

**SC-2** A synthetic map whose win requires the hero at a point does not win with the hero elsewhere
and does win with the hero there (FR-1, FR-2, AC-3).

**SC-3** Compile-bound id equals start-assigned id across placement counts and party sizes (FR-5).

**SC-4** No world built without a party changes digest; no byte form version changes (FR-8, FR-9).
