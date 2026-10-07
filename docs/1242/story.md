# Dialogue mouse and inn registration

## Result

Town and mission dialogue OK advances once on an owned inside release. Every
nonzero live inn hearing queues a mission identity, including repeats. Leaving
the inn registers those identities in heard order.

## Authority

Pinned knowledge is `6c6da036a53085d4e58ec028c81b629b4cf24d94`.
`DIALOGUE-044/045` establish captured modal dispatch and button down/up.
`TAVERN-023` establishes pending multiplicity and ordered registration calls.
`TAVERN-024` establishes first-match paired removal, selected identity, announce
latch and the marker identity guard. Its constructors and later campaign
consequences remain Unknown. `TOWN-123/040` identify that same persisted marker
collection and its valid non-`nothing` Picture population rule.

## Behaviour

App owns one transient dialogue pointer gesture. Town page revision and mission
notice picture identify its page. A press inside arms OK. Moving outside then
back preserves the press; release outside cancels. Orphan release, body and
right input advance nothing. Key-driven paging, replacement, close, focus loss
and screen changes cancel ownership and absorb the pending release. Enter and
Escape keep their paging routes. Success and failure keep their separate
press-triggered controls. Native frame/window placement stays unchanged.

The inn queue holds identities for one visit. Zero sentinels and kept speakers
queue nothing. Registration consumes the first remaining matching live pair
through its stable label, then selects and announces the identity even when no
pair remains. Registering the current main does not reload its record or alter
map-entry policy. Direct `Town.Take` still rejects a consumed stable index.

Registration reaches the existing picture-bearing marker seam. SAV writes the
same campaign offers, selection, announce and identity collection; no second
journal or reward is added. Existing marker payloads remain intact and newly
constructed markers use the existing picture path and zero trailing fields.
The kept-speaker owner rule remains separate (`DIV-1529/1530`).

## Proof

Evidence is under `review/story1242-dialogue-inn/` outside this repository.
`reproduction-dialogue.txt` and `reproduction-inn.txt` retain baseline failures
through `HeadlessPointer` and App inn input.

`TestDialoguePointer*` compares every pointer edge and blocked underlying
action for both surfaces, four rectangle edges, cancellation, out-and-back,
page/lifecycle changes, scaled windows and outcome panels.
`TestInnRegistersHeardIdentitiesInOrder` covers native/current models with
`[31]`, `[31,31]`, `[31,41,31]`, zero and paired `[31/A,41/B,31/C]`.
`TestInnRegistrationRunsAfterLastPairAndKeepsCurrentMain` independently checks
absent-pair effects and current-main record retention.
`TestInnQueueCallsEveryIdentityInOrder` records the production registration
loop's arguments independently of selection and paired removal.

`TestReleaseInnRegistrationSAVAndNextAction` uses installed inn dialogue and
App pointer input with synthetic repeated paired offers at main 30 and 50,
current SAV through the ordinary SAVE dialog, cold App LOAD and the next city
action. Ordinary SAV arrays and selection are read independently. Identity,
stable-index, selected-identity and marker loss controls each produce a
different restored result. Gold equality is a bound for this vector.
`TestReleaseMissionDialoguePointerCancellation` uses
mission 10's installed event dialogue. EN/RU receipts name their populations.

Focused EN/RU witnesses, ordinary tests in every touched package, storyguard,
architecture and divergence guards, and `go build ./...` pass. The release
merge gates remain the seat's responsibility.

## Open debt

The marker producer is reused, not a complete campaign-registration oracle.
`DIV-903` retains the constructor payload gap; `DIV-904` retains the policy for
preexisting duplicate imported marker records and sorted new-marker construction.
TAVERN-024 does not establish native interleaving, later rewards or complete
mission-loader effects. The
engine witnesses do not execute the original runtime.

`DIV-1584` is closed by the promoted queue and registration contracts.
`DIV-1583` still concerns missing/hidden first parts and is unchanged.
Reserved `DIV-1613..1614` remain unused. The seat owns the sole review, final
merge chain, EN/RU release/M2, scenario and build promotion.
