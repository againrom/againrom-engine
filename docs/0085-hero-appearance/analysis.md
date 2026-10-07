# Analysis — the party member is drawn as an unarmed man

## What was not known

The party member a mission starts with was placed carrying the class key `1`, and the comment beside
that constant said the value was arbitrary and asserted nothing. It was not arbitrary in effect: `1`
is a real class of the shipped roster, the roster's own name for it is *Unarmed Fighter*, and its
sheet is the bare-handed human body. So the one figure the player controls arrived drawn as a man
holding nothing, and the picture was a consequence of a constant rather than of anything about him.

What was not known was what the original uses instead. Two readings were open and they are not
variations of one another: either a character carries a class the way a placed unit does — in which
case the fix is to carry a better number — or the class is produced from something else, in which
case a better number is the same defect with a nicer value.

## What the corpus says

The second reading is the right one, and the two paths really are different.

For an actor a map places, the drawn class **is** the streamed type id and nothing derives it
(`UNIT-APPEAR-030`). For a player's character the client throws the shipped id away, banks two flag
bits and forces the drawn class to the literal `1` — `Unarmed Fighter`, the very picture the owner
is looking at — and then **derives** the real answer from a twelve-slot array of visible equipment
(`HERO-APPEAR-041`, `HERO-APPEAR-042`). One slot supplies a body **name**; a second, when occupied,
appends `_` to it; a mage's name is substituted, and so is a dying character's. The name then runs a
seventeen-arm string comparison into the drawn class key.

Two consequences decide the shape of this story.

**The forced `1` is the law's own fallback.** The literal is stored before the chain runs and the
chain only overwrites on a match, so a body name matching no arm leaves the character drawn as an
unarmed man. That is exactly the state this tree is in — not by coincidence but because it never ran
the chain at all.

**A hero's pixels do not come through the class record's `File`** (`HERO-APPEAR-043`). The sheet is
composed: `units/` + a directory + the body name + `sprites.256`. Only the geometry — canvas,
centre, phase scalars, timelines — comes from the class record the name resolved to. Drawing a party
member through the class record's own art would therefore be the map-placed actor's route applied to
the one actor it is documented not to serve.

The directory is a three-way and this tree lands on a **derived** arm of it: a mage always takes
`heroes`, a fighter with **no object in the armour slot** takes `heroes_l`, and anything else takes
the armour material's own path out of a sixteen-entry table. This tree's party member is a fighter
holding no armour, so its directory is the second arm and is not authored.

The agreement that settles the composition is `HERO-APPEAR-046`: every shipped hero body sheet holds
exactly the frame count the class record its name resolves to predicts, on fourteen of sixteen, with
both residuals equal to an independently derived constant. So the class record's animation
descriptor and the composed sheet are made for each other, and a composite of the two is not a
guess.

## The term this tree does not have

Slot 0's five-bit field indexes an **ordered list of names built at run time**. The corpus pins the
*set* — the sixteen shipped body directories — and states the order as not established
(`HERO-APPEAR-042`, and EXP-0113's own bounded Unknown). This tree has no equipment slots at all: a
party member holds a `data.Weapon` resolved from an authored literal, with no five-bit field
anywhere. So the step *from what he carries to which body he wears* is missing on both sides, and no
amount of care with the rest of the chain supplies it.

That is the one value this story authors, and it is authored as a **body name** rather than as a
class key. The difference is not cosmetic: a body name is one element of a set the corpus pins, and
everything after it — the class key, the sheet, the geometry, the corpse link — follows from the
game's own law. A class key would be a second arbitrary constant, which is what the retired one
already was.

## What was read

`research/claims/hero.md` (`HERO-APPEAR-040`…`046`), `research/claims/unit.md` (`UNIT-APPEAR-030`,
`UNIT-EQUIP-005`), `research/formats/hero/format.md`, `research/formats/unit/format.md`, and
`research/experiments/EXP-0039-unit-frame/evidence/unit-classes.csv` for the shipped roster's own id
and name columns — read to check the handed premise that id 1 carries that name, which it does.

In this tree: the retired constant in `pkg/game/frontend.go` and the whole path it reached the
screen through — `pkg/mapload/start.go`'s party loop, `sim.Entity.Class`, `pkg/game/world.go`'s
`entityDraws`, `pkg/game/units.go`'s loader and `pkg/render/terrain/units.go`'s bundle.

## An observation the plan had to answer

The per-entity art override this story needs already exists in shape, twice: `mapWorld.tiers` and
`mapWorld.chars` are both per-placement facts resolved once when the map opens and read by the push.
A third of the same kind is not a new mechanism, and writing it as anything else — a field on the
entity, a second bundle keyed differently — would have been.
