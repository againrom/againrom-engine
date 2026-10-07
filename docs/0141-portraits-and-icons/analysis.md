# 0141 - analysis

## What this tree had already disclosed as open

Three places in the code said, in their own doc blocks, that they were showing something other than
what the original shows, and each named what was missing:

- **The doll box** (0140). It drew the party character's composed figure and, for anything else,
  that unit's own drawn world sprite — because the four figure directories this project could
  name are the four human ones, and nothing published said where a monster's picture lived. It was
  labelled a placeholder that reads as one, with "research owes the real address" written beside it.
- **The face seam** (0079). `FaceSource` had been a one-method interface with no implementation
  since it was written, on the ground that the picture-bearing keys of the registry the speaker
  numbers reach were published with their domains measured and their meaning graded **Unknown**.
- **The spellbook cell.** It carried the first letters of a spell's name, with an admission
  attached: the original plainly draws one small picture per spell, and nothing published named
  where the art lived, what indexed it or how it was keyed.

Two research experiments then landed against all three, and a third landed mid-story.

## What we did not know going in, and what settled it

**Where a monster's picture lives, and whether it is a doll at all.** The prior assumption was that
a monster's picture would be a doll like a person's, addressed out of a figure directory nobody had
named. It is not: whether an actor composes or loads is one test on the class, the two arms are
exclusive, and the flat side is a plain Windows bitmap. That made the shape of the fallback wrong as
well as the picture — and it made a new format leaf necessary, because that tree is not this
engine's own sprite format.

**Which of the composer's two arms a placed PERSON takes.** This is the one thing the story got
wrong and had to reverse, and it is worth writing down because the wrong answer looked right. The
second arm of the engine's composer reads a sex bit out of the face byte's top, and the row's type
id column is 1..24 on every shipped row, which is squarely inside that arm's range. Reading the sex
from bit 7 therefore type-checks against the claim. It drew all 462 campaign human placements as
men, and the owner found it by playing mission 10 and looking at the witch.

The correction is that a human's wire class byte is **not** its type-id column: the constructor
re-reads the gender slot the streamer drops and overwrites the byte with `gender + 0x21` or
`+ 0x23`, which is precisely what makes `[0x20,0x40)` the human range and puts every placed person
on the HERO arm. The face byte's bit belongs to actors whose class byte reaches the wire
unmodified, and this tree places none of those on the composing side.

The lesson that generalises: a claim can be about a code path that the data never reaches. Checking
that a column's domain fits an arm's range test is not the same as checking that the value reaching
that test is the column.

**Which actor an `npc=N` tag names.** Two readings had support and the corpus chose before the
instructions did. A placement's secondary key *is* the `npc.reg` subscript on the NPC arm, so
binding the tag to the placement standing on the map is a real reading — and it is the person
the player can see. It matched 53 of 337 campaign tags, and seven of those missions carry dialogue
while placing no npc at all. Resolved through the registry the same sweep matched 337 of 337. The
claim that later followed the number from the parser's `sscanf` to the array subscript agreed with
the measurement; the measurement had already eliminated the alternative.

**How the dialogue pane crops.** The pane is 88x114 and every picture a speaker resolves to is
160x240, so a crop is unavoidable — and this story invented one twice, first centred and then
top-aligned, at a time when the four `Portrait*` keys were graded Unknown. Both were wrong. The
engine makes one blit with a source rectangle and a destination point, and the rectangle is a fixed
72x96 window at a per-speaker origin. That the *destination* is pane-relative rather than
panel-relative is still ours, on the arithmetic `8 + 72 + 8 = 88`.

## What the owner ruled

Four rulings, all 2026-08-11, all after playing a build:

- a dialogue's portraits should be of whoever is speaking;
- show the doll of the character and of any enemy, and if nobody is selected, show nothing;
- the monster portraits must take the tier into account — an orc is green, red, blue, black.
  This one is a **disclosed divergence**: the original's info panel takes no tier and only its
  dialogue panel does. The address is reproduced; which window uses it is ours;
- the portrait's background is missing — make it black, and it sits a little too far left. Both
  turned out to be one defect with one decoded answer, and the "5-10px to the right" he estimated by
  eye came out of the registry as +4 to +13, mean 7.9.
