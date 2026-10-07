# 0131 — the campaign advances: specification

**Mission 10 finishes and mission 20 starts, with the same hero and everything
he is carrying — the campaign's own key decides that, not ours.** This contract
is self-contained.

## Problem, and the behaviour today

Today a won mission ends at the map list: the banner is dismissed, the map
screen is torn down, the list appears holding a sentence naming what follows,
and the player finds that row and clicks it. Two baseline facts matter:

- The party **already** carries. A mission opened by number starts with the
  members the last won mission left behind — their experience, packs and worn
  sets — and mints a default hero only before the first win. That is true of
  every door this front end has, and this story does not change it.
- **Nothing reads `[Mission<n>] AutoGetMission`.** What follows a mission is
  answered instead by the next `[Mission<n>]` section whose number is a whole
  multiple of ten, ascending. That order is this tree's own and says so.

The gap is exact: the campaign states its own successor, nothing reads it, and
nothing opens a successor at all.

## Functional requirements

### The key

**FR-1** The mission set read from the scenario registry MUST carry, per
`[Mission<n>]` section, the **successor that section declares**. The declaration
is the section's `AutoGetMission` key, read as a single integer:

| The section holds | The mission declares |
|---|---|
| no `AutoGetMission` | no successor |
| `AutoGetMission = -1` | no successor |
| `AutoGetMission = v`, any other `v` | successor `v`, exactly as written |
| `AutoGetMission` held as anything but a single integer value — an array, a string, a subsection | no successor |

The successor a section declares MUST be answerable with no game install
present. A declared successor MUST NOT be required to have a `[Mission<n>]`
section of its own: a mission is addressed by its number, and a campaign may
declare a successor it says nothing else about.

**FR-2** The successor a mission declares MUST NOT be confused with, derived
from, or reconciled against the next main mission this tree names by ascending
order. Both MUST remain separately readable, and a mission set in which they
disagree MUST report each faithfully.

### The advance

**FR-3** Dismissing the victory banner of a won mission that declares a
successor MUST leave the successor's mission **the one being played**, with its
own map on screen and its own world advancing. The map list MUST NOT be shown at
any point in between. The successor's map screen MUST be indistinguishable from
the same mission opened from the map list — the same view of the same window,
opened on the party, at the same starting cadence — whichever of the inputs that
dismiss a banner was used.

**FR-4** The party opening the successor MUST be the party that finished the won
mission — each member's experience, pack and worn set as his entity ended them.

**FR-5** Winning a mission that declares **no** successor MUST behave exactly as
it does today: the front end returns to the map list, showing a sentence that
names the next main mission by this tree's own ascending order, or states that
this is the last mission the campaign declares, or that the install declares no
campaign at all.

**FR-6** The party MUST be carried in every case — a declared successor, an
absent one, a successor that cannot be opened, and a mission set this tree could
not read at all. Carrying MUST NOT depend on any of them.

**FR-7** A declared successor that names no mission — any value at or below zero
— MUST NOT be opened, and MUST be refused before anything is read. The front end
returns to the map list, and the **sentence** it carries there names both the
mission won and the number the campaign declared.

**FR-8** A declared successor that names a mission but whose map cannot be read
or decoded MUST NOT leave the player on a map screen or on no screen. The front
end returns to the map list carrying **the failure's own words** — the same
report a mission that fails to open by any other route already produces — and
not the sentence of FR-7 or FR-5. The two failures are told apart by which of
the two the player is shown.

**FR-9** The input that dismisses the victory banner MUST NOT also act inside
the mission that dismissal opens. One press ends one mission and begins the
next, and does nothing else.

**FR-10** An advance MUST be able to follow an advance: a successor that itself
declares one advances again on its own win, with no limit written anywhere and
no state remembered between the two.

### Saying so headlessly

**FR-11** The headless check mode, asked about a mission, MUST state what
winning that mission does: the successor it declares, or that it declares none;
and where one is declared and can be opened, it MUST actually open it and report
the successor's own address together with the started hero's health, mana and
skill experience **as the advance delivers them**.

## Acceptance criteria

| # | GIVEN | WHEN | THEN | Level |
|---|---|---|---|---|
| AC-1 | a registry whose `[Mission10]` holds `AutoGetMission = 20` and whose other sections hold none | the mission set is read | mission 10 declares successor 20 and every other mission declares none | unit |
| AC-2 | a section holding `AutoGetMission = -1` | the mission set is read | that mission declares no successor, indistinguishably from a section holding no such key | unit |
| AC-3 | a registry in which the declared successor and the next main mission by ascending order differ | the mission set is read | both are reported, each its own value | unit |
| AC-4 | a won mission declaring successor `m`, and a front end holding a mission set that says so | the victory banner is dismissed | the front end is showing the map screen of mission `m`, and was never on the map list | unit |
| AC-5 | the same, where the finished party's hero ended with a given experience, pack and worn set | the victory banner is dismissed | the party standing in mission `m` is that hero, with those three unchanged | unit |
| AC-6 | a won mission declaring no successor | the victory banner is dismissed | the front end is on the map list holding today's sentence, and the party is carried | unit |
| AC-7 | a won mission declaring successor `0` | the victory banner is dismissed | the front end is on the map list, the sentence names mission `0` as declared and unopenable, and the party is carried | unit |
| AC-8 | a won mission declaring a successor whose map is absent from the archives | the victory banner is dismissed | the front end is on the map list, its message states the failure, and the party is carried | unit |
| AC-9 | a map screen showing a victory banner that will advance | the banner is dismissed by each of its inputs in turn | the newly opened mission receives no part of that press: no order, no selection, no attack, no second dismissal | unit |
| AC-10 | a lawful install, both language roots | the check mode is asked about mission 10 | it reports successor 20, opens it, and reports the hero's health, mana and skill experience there | manual |
| AC-11 | a lawful install, both language roots | the check mode is asked about mission 20 | it reports that mission 20 declares no successor | manual |
| AC-12 | a front end holding no mission set at all — none read, or none this tree could parse | a mission is won | the front end is on the map list, the sentence says the install declares no campaign, and the party is carried | unit |
| AC-13 | a window of a size other than the default, and a won mission declaring a successor | the banner is dismissed by each of its inputs in turn | the successor's map screen holds that window's own view, and the view it opens on is the one the map list's own door would have given it | unit |

**Error cases:** AC-7, AC-8 and AC-12. The first two differ in kind — a
declaration refused before anything is read, and a load that fails when tried —
and are told apart by which message the player is shown. AC-12 is neither: an
install declaring no campaign is not broken, and only the carry must survive.

## Derived properties

**P-1** *(invariant)* For any won mission, the party the front end holds
afterwards is the party that finished it, whatever the mission set says and
whether or not a successor could be opened.

**P-2** *(negative invariant)* For any won mission whose declared successor
cannot be opened — non-positive, or a map that will not read — no map screen is
left showing, and the front end holds no seam onto the mission that ended.

**P-3** *(negative invariant)* For any dismissal that opens a successor, no
input of that frame reaches the opened mission.

**P-4** *(completeness)* Every `[Mission<n>]` section of a parsed registry
yields an answer to "what successor does it declare" — a declaration or its
absence — and no section is skipped for the shape or type of its key.

**P-5** *(idempotence)* Dismissing an already-dismissed banner advances nothing:
the second dismissal opens no mission and carries no party.

**P-6** *(invariant)* For any mission this front end opens — from the map list,
by number, from the generation screen, or by an advance — the map screen it
leaves showing is in the same state: sized to the window it is in, opened on the
party, at the cadence the map load itself runs at.

## I/O examples

The sentence a win produces, by case. The first `%d` is the mission won, the
second the mission named:

```
mission %d won — your party carries over; mission %d follows and opens now
mission %d won — your party carries over; mission %d follows: choose its row
mission %d won — your party carries over; the campaign leads to a town here and the town is not built, so mission %d stands in for its gate: choose its row
mission %d won — your party carries over; the campaign declares mission %d follows and this tree can open no such mission: choose a row
mission %d won — your party carries over; it is the last mission the campaign declares
mission %d won — your party carries over; this install declares no campaign, so nothing names what follows
```

The first and the fourth are new; the rest are today's. The first is shown on
no screen — the mission it names opens instead — and exists for FR-11.

The check mode's advance line, on the two shapes FR-11 names:

```
againrom: winning mission 10 opens mission 20 at <address>, <w>x<h>, <n> entities; hero health <hp>/<max>, mana <mana>/<max>, skill xp [<six integers>]
againrom: winning mission 20 opens nothing — mission 20 declares no successor
```

## Constraints

**C-1 — which fact decides the advance.** The alternatives, and what an observer
could see:

| | Choice | Observable trade-off |
|---|---|---|
| A | The campaign's own `AutoGetMission` | A modded campaign advances the way its author wrote it. On a stock campaign, indistinguishable from B. |
| B | The next main mission by ascending number | Simpler; silently wrong wherever an author declared something else, and wrong in a way no stock install can reveal. |
| C | Both, reconciled | Needs a rule for the disagreement, and any such rule is ours. Neither fact survives intact. |

**A**, with B retained where A is silent: a mission declaring no successor must
lead somewhere, and the map list is where this tree can send it.

**C-2 — the advance is a load, not a journey.** The original walks the party to
the successor's own position on a global map first. This tree has no global map,
so the successor's map opens directly; nothing of that walk is stood in for.

**C-3 — nothing is persisted.** The carried party lives for the life of the
process; a campaign that survives the program closing is a different contract.

## Out of scope

- **The town.** What follows mission 20 is the campaign's home screen; this tree
  has none and this story invents none.
- **`globalmap.reg`** and everything positional it carries.
- **Mercenaries**, and the keys sitting beside `AutoGetMission` in the same
  sections.
- **`LastMission`.** It ships once and is stored by the same routine; nothing
  here reads it, and the campaign's end stays FR-5's "last mission the campaign
  declares".
- **Mission 20's own win condition**, and whether any shipped mission can be won
  by a player today.
- **Health, equipment and item content.** What a carried hero *is* does not
  change; this story decides only which mission he walks into next.
- **Choosing a row.** Every map-list row stays choosable and still carries the
  party. The advance adds a door; it closes none.

## Verification mapping

| AC | How |
|---|---|
| AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7, AC-8, AC-9, AC-12, AC-13 | automated, `go test` |
| AC-10, AC-11 | manual, the check mode against each lawful root |

## Gate check

FR-1 → AC-1, AC-2, P-4. FR-2 → AC-3. FR-3 → AC-4, AC-10, AC-13, P-6. FR-4 → AC-5.
FR-5 → AC-6, AC-11. FR-6 → AC-6, AC-7, AC-8, AC-12, P-1. FR-7 → AC-7, P-2.
FR-8 → AC-8, P-2. FR-9 → AC-9, P-3. FR-10 → AC-4 with P-5: nothing is
remembered between one advance and the next. FR-11 → AC-10, AC-11.
