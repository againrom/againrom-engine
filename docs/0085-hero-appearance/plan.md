# Plan — the party member is drawn as what he wears

## Design decisions

**DD-1 — the law lives in `pkg/data`, beside the class records it answers about.** The appearance
law is a function of names and integers; it opens nothing and decodes nothing. It goes where the
unit class records already are, because its output is a subscript into them and its whole purpose is
to pick one — a second package for four pure functions would put the question one import away from
its answer. The drawing tier is excluded on its own standing rule (P-2): it may not import `pkg/data`
and re-derives nothing from a registry.

**DD-2 — a body name is a defined string type, not a bare string.** `data.HeroBody`. The set is
closed and pinned by the corpus, and every function in this story takes one or answers one; a bare
string would let a class name, a sheet address or a registry key be passed where a body name is
meant, and all three exist in the same packages. The bundle's own map is keyed by the plain string
for DD-9's reason.

**DD-3 — the mapping is TOTAL and answers a pair.** `HeroBodyClass(b) (int32, bool)`. The key comes
back for every input, because the original's fallback is a real behaviour and not an error path: the
literal is stored before the compare chain runs, so an unmatched name leaves the character drawn as
the bare-handed body. The bool is what separates *matched* from *fell back on the same value the
first arm would have given*, which no single return can. This is the only place in the tree that
knows the fallback's value.

**DD-4 — the directory function takes what the reading rests on, not a slot array.** `HeroBodyDir`
takes a mage flag, a material index and whether a material is known at all. The corpus grades
*which slot is armour* below the arithmetic around it, so the law here consumes the two facts and
not the slot: a tree that later grows an equipment channel supplies them from wherever it finds
them, and nothing in this story has to be revisited to be right.

**DD-5 — the sheet address is composed by the formatters the class records already use.** The body's
base is `units/` + directory + `/` + name + `/sprites`, handed to the same two extension formatters
a class record's own address goes through. So the ".256" and the sibling's inserted "b" exist once,
and a body address and a class address cannot come to disagree about either.

**DD-6 — the party member carries its body, and its class is derived where the party is built.**
`mapload.PartyMember` gains one field, `Body`. It reaches no entity, no byte form and no digest — it
is a loader input, exactly as the hero's statistics are — and the loop that mints entities is not
touched. `MissionParty` is the single site that turns the body into the class key, so there is one
expression in the tree that produces a party member's class (P-1) and the retired constant has no
successor.

**DD-7 — the authored body is a FUNCTION and not a package variable.** `PartySpread`'s own reason:
a variable is writable from anywhere, and "every mission starts with the same hero" would then be
true by mutation rather than by design. It sits beside the two authored values already there, so the
three authored facts about this hero are in one place and each says so.

**DD-8 — a loaded body is a STRUCT COPY of the class record with the frames replaced.** Not a new
type and not a wrapper: the render tier's loaded class is already exactly "canvas, descriptor, name,
corpse link, frames", which is what a hero body is, with one field from elsewhere. Copying gives
FR-4's independence for free — the copy's frames cannot reach the record and the record's cannot
reach the copy — and every selection, placement, cull and texture path downstream receives what it
already received. The tier slices are cleared rather than carried: a tier is a colour table for the
record's **own** sheet, and carrying them would recolour a hero through a table for another picture.

**DD-9 — the bundle gains a second map, keyed by the plain body name.** `terrain.UnitSet.Bodies`.
It sits on the bundle rather than on the front end because it is art, and because the driver that
resolves an entity's picture already holds the bundle and nothing else that could carry it. The key
is a plain string because that tier may not import `pkg/data`.

**DD-10 — the front end resolves the party's body once, beside the bundle, and a failure is nil.**
One call after the unit load, for the one body this front end's party wears, in the directory the
law gives for a fighter holding no armour. Every failure — no class record, no entry, an undecodable
or palette-less sheet — leaves no entry, which is FR-6's whole implementation: the lookup misses and
the member draws the class record's own art. The font's precedent; a cosmetic asset does not gate a
mission opening.

**DD-11 — the per-entity art is a lookup on the map driver, and a constructor argument.**
`mapWorld.art`, resolved when the map opens and never written again, looked up by key and never
ranged. `tiers` and `chars` are the same shape for the same reason, and `chars` is a constructor
argument for a reason that applies here unchanged: the tick-0 push is the only push a mission opened
and left stopped ever gets, so a lookup installed afterwards would leave that mission's first frame
drawing the old picture.

**DD-12 — the override replaces the WHOLE class at the one lookup site.** One statement after the
class lookup in the entity push, before anything reads the result. So the name, the live selection,
the swing and the fall all follow from it by construction rather than by three edits that agree
today — and the fall follows it into the class record's corpse link, which is FR-8's second
divergence arriving as a consequence rather than as a decision.

## Risks

**R-1 — the composite's descriptor and the composed sheet must agree.** The frame selector is bounds
guarded against the sheet's own count, so a body sheet shorter than the descriptor expects does not
crash: it silently selects a wrong frame or refuses. The evidence for the pairing is the corpus's,
not this tree's, so what is tested here is that the composite carries the record's descriptor
unchanged and the sheet's frames unchanged — the agreement itself is a fact about an install.

**R-2 — every caller of the map driver's constructor moves.** Three production sites and one test
helper. The nil argument must mean "no override for anybody", so the plain constructor keeps
behaving exactly as it did (P-4).

**R-3 — the retired constant has readers.** Tests assert a placed member's class against it. They
must be re-aimed at the derived value rather than at a new literal, or P-1 is satisfied in the
source and broken in the evidence.

**R-4 — the derived key changes what a mission's world hashes to.** It is a value change and not a
form change, and the distinction has to be demonstrated rather than asserted: the version literal
and the encoded layout are what must be shown unchanged (AC-11).

## Success criteria

**SC-1** — every published body name maps to its published key, and an unmatched name answers the
fallback with the match flag clear.

**SC-2** — the suffix and both substitutions compose as published, and a living fighter's name is
untouched by either substitution.

**SC-3** — all sixteen material blocks and both special arms answer their published directory.

**SC-4** — a composed sheet address and its sibling are the published strings.

**SC-5** — the party a mission starts with carries the law's answer for the authored body, and it is
not the unmatched fallback.

**SC-6** — a loaded body carries the class record's canvas, centre, descriptor, name and corpse link
and the composed sheet's frames, and carries no tier slice.

**SC-7** — an absent composed sheet produces no body, and the member still draws.

**SC-8** — a mission's entity picture draws the party member from the body's frames and every other
entity from its own class record.

**SC-9** — the byte form's version literal is unchanged and a world assembled from identical
entities encodes to identical bytes and the same digest.

**SC-10** — the authored body and the authored trained skill are checked as one pair.

**SC-11** — the full gate is clean: build, vet, gofmt, tests, and the three project scripts.

## Traceability

| Requirement | Decisions | Criteria |
|---|---|---|
| FR-1 | DD-6, DD-7 | SC-5 |
| FR-2 | DD-2, DD-3 | SC-1, SC-2 |
| FR-3 | DD-4 | SC-3 |
| FR-4 | DD-5, DD-8, DD-12 | SC-4, SC-6, SC-8 |
| FR-5 | DD-7 | SC-5, SC-10 |
| FR-6 | DD-10 | SC-7 |
| FR-7 | DD-6 | SC-9 |
| FR-8 | DD-4, DD-6, DD-12 | SC-2, SC-3 |
| P-1 | DD-6 | SC-5 |
| P-2 | DD-1, DD-9 | SC-8 |
| P-3 | DD-7 | SC-10 |
| P-4 | DD-11, DD-12 | SC-8 |
