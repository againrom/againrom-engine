# 0148 — provenance

Every fact this story reads from the original game, with the claim id it comes from and the
confidence that claim carries. The `research/` submodule is pinned at `d1e38ad` for the whole
story. Nothing here was read from an experiment folder.

This story corrects three defects in what 0147 shipped. It decodes nothing new. Every claim below
is one 0147 already reads; the two rows that carry this story's own new reading are marked.

## What is read from the save

| Claim | Confidence | What is used |
|---|---|---|
| `SAV-ID-015` | High as the identity on this corpus | **New to this story.** The head's `+0x0c` id is a runtime creation-order id and the hero's is **1**. The mechanism is read: the id is the actor's `+0x04` field, assigned at tick-list insert as the lowest free bit of the bitmap at `L06793`. This is the discriminator FR-1 orders the party by. |
| `SAV-TOKEN-034` | High | The 37-byte placeable head, and that `SAV-ID-015`'s creation-order id is the `u32` at head `+12` while the map unit id is the low `u16` of the `u32` at head `+19`. This is why a character carries both ids and why they are different id spaces. |
| `SAV-OWNER-048` | High | The human participant's own objects are the subtree of the stream's first top-level record. It also states the corpus population this story measures on: 23 files, 18 distinct. |
| `SAV-UNITFLD-049` | High for the nine named words | The four statistics and the two pool pairs a restored character carries. Unchanged from 0147; read here only because FR-3 decides where those values stop applying. |
| `SAV-TOPLVL-052` | High | The walk's extent. Named because it is what still bounds this story: the explored plane lies past the terminator and is not restored (L-3). |

## What is read from the map

| Claim | Confidence | What is used |
|---|---|---|
| `ALM-TRIG-046` | High | A `Target_Unit` parameter names one of three id spaces, and the below-10001 band names a placed unit record's own identifier word. FR-2 rebinds a value in that band; it changes which entity the value resolves to and not which band it is in. |
| `ALM-UNIT-048` | High | The type-6 record's identifier word is the id that band names, and it is the same word `SAV-ID-015` reports at save head `+0x13`. This is what makes a restored character and a withdrawn map record the same person rather than two records that happen to share a number. |

## Ours by choice

- **Which character leads the restored party (FR-1, AUTHORED).** `SAV-ID-015` supplies the
  discriminator and grades it High, but nothing published states what order a consumer should put
  the characters in, and the file's own actor-list order is not hero-first. The ordering rule, its
  fallback, and the decision to leave every other character in file order are this tree's, and
  spec.md L-1 states them.
- **Rebinding a withdrawn placement rather than leaving it unresolved (FR-2).** 0147 withdrew the
  map record and accepted, as a disclosed divergence, that a script arm naming it would resolve to
  no entity. That divergence is now measured to be wrong in a way its own text did not predict: an
  unresolved check writes no register, the register keeps its zero, and a proximity comparison
  against zero holds. The repair is this tree's choice of the two available. The original never
  reaches the case at all, because it does not withdraw the record.
- **A `Saved` does not cross a mission boundary (FR-3).** Nothing published says how long a saved
  cell and pool pair remain valid. This tree decides they describe exactly the mission the save was
  taken in.

## Open, and not answered here

- **What the file's actor-list order means.** The order is not hero-first, and it is not
  established here what it is. `SAV-ID-015` gives an id whose ordering is a different question from
  the list's. A research lane is opening this in parallel; this story authors a rule and does not
  publish a claim.
- **The explored plane.** Not decoded in `pkg/formats/sav` at all, so a resumed mission shows only
  what the party reveals after the load. This story does not touch it. A research lane is opening
  the question in parallel.
