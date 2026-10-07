# Provenance — saves written by older builds load again

## Backing

This story cites no research claim. Every fact it depends on is this build's own recorded history,
not decoded ROM1 behaviour:

- The byte-form version ladder from 50 through 57 (which version added which field, at which byte
  width) is `pkg/sim/binary.go`'s own header comment, written and amended by the stories that made
  each bump (`0165`, `0166`, `0168`, `0169`, `1001`, story `1029`'s corpse-loot work at 54,
  another story at 55, `1025` at 56, `1029` at 57). `spec.md` cites the file and line, not a claim
  id, because the fact is this tree's own commit history rather than a decoded fact about the
  original executable.
- `pkg/sim/castbinary.go`'s pre-53 area-effect record shape (cell key, spell id, remaining
  lifetime, 6 bytes, no tag) is read from this tree's own git history (`git show
  <pre-1001-commit>:pkg/sim/castbinary.go`), the same way — it is what THIS build's encoder wrote
  before story `1001`, not a claim about ROM1's own save format.
- `pkg/sim/weight.go`'s treatment of `Load` is read from that file's own `recomputeLoad`: the value
  is a sum over the world's item-weight table, so it is zero for every actor whenever that table is
  empty. An earlier draft of this document and of `DIV-257` called zero a not-computed sentinel that
  the next inventory change would resolve. That was wrong, and the correction is why the load path
  re-declares the table from the started mission.

`docs/DIVERGENCES.md`'s `DIV-026` (not amended by this story) is the one row that names what the
**original** game's own save format restores; every row this story adds (`DIV-254` through
`DIV-258` and `DIV-274` through `DIV-276`) cites `—` for ROM1 behaviour and says why: the fields
this story's `UpgradeSaveForm` substitutes are this build's own byte-form additions with no
counterpart in the original save at all, so there is no ROM1 fact to be silent about.

## Why no claim applies

The five behaviours are about reading **this build's own prior output** across **this build's own
version history**, not about reproducing an original-game format. `AMBER`'s threshold and B1 (never
admit an unverified fact) both bind facts about ROM1; nothing this story asserts is a fact about
ROM1. The owner's three directives recorded in `contract.md` — read 50 and above, refuse below 50
by name, disclose every substitution, and match `cmd/saverepair`'s report/`-w` shape — are testimony
about what this project's own tooling should do, which `CLAUDE.md`'s B3 distinguishes from owner
testimony about ROM1: an owner ruling on our own build is a fact, not a question to verify against
the game.
