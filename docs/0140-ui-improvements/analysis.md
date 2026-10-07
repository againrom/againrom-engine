# 0140 — analysis

## What this story actually is

Twelve commits over three days, all of one kind: **the owner looked at the running game beside his
own lawful install and said what was wrong with ours.** No format was decoded for it and none needed
to be. The work is arrangement — where a box stands, what it states, which press it takes, what the
game opens at — and every number in it is this project's own.

That makes the usual question ("what does the original do?") the wrong first question. It was asked
anyway, once per area, and the answer was consistently *nothing published says*. What the areas do
carry is recorded in `provenance.md`; the short version is that the original's own information
panel cannot be recovered from the executable at all (`UNIT-PANEL-011`), and that its minimap's
click dispatch is decoded but is six arms and five gates we are not reproducing (`AI-MINIMAP-062`).

## What was not known, and how each was settled

**Where the generation screen belongs.** 0119 plan DD-18 put it behind `-chargen` and reasoned that
entering a mission should not regenerate a hero. The owner reversed it: generation is what *starting
a campaign* does. The distinction DD-18 was groping for turned out to be real but differently
placed — a **won mission's successor** must not regenerate, because 0131's carry would be destroyed.
That is now a named boundary rather than a blanket refusal.

**Which resistance array the generation screen was showing.** The Derived block printed
`data.Derived.Resistance`. `UNIT-COMBAT-015` binds columns 19..23 `prot Fire..Astral` to
`Protection` and columns 24..28 `res.Blade..res.Shooting` to `Resistance`; the second family is the
weapon damage-kind five, never re-derived for a human, five zeros on every generated character. The
block was showing five zeros and calling them resistances. Two stated reasons for excluding the
block's other rows were also checked and both were wrong: "six lines is what fits" had never been
measured, and "constant across every spread" is false because Spirit is one of the four the player
spends and the protections are Spirit halved (`HERO-RESIST-012`).

**Whether a monster has a doll picture.** The owner says it does. This build cannot address one: a
figure sheet lives under `equipment/<figure directory>/` and the four figure directories that can be
named are the four human ones (`HERO-APPEAR-051`). Nothing published names a monster's. This is the
one open research question the story leaves, and it is stated as such rather than guessed around.

**Why the minimap grew after picking a hero.** Reported as an initialisation bug. It was not: the
minimap took its width from whatever the unit panel had just composed, and the panel is
fit-to-content and absent entirely with nothing selected. The width was a function of the
*selection*. Two rounds were needed to finish it — pinning the width left the panel's fit-to-content
*height* still bounding the square, and only reserving the square outright removed the coupling.

## What we looked at

The owner's photograph of his own running install, for the bottom band's shape and the panel's row
order; `cmd/paneldump` over missions 10 and 20 against both roots, for how wide a panel actually
composes; Ebitengine's own source, for whether `Monitor()` is valid before `RunGame` and what units
`Monitor.Size()` and `SetWindowSize` speak; and this tree's own test suite, which found a real
defect the first minimap change introduced — twenty-eight failures across nine files, all one cause,
an opaque click surface too large for its window.
