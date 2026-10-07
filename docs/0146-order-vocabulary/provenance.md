# 0146 — provenance

## Claims relied on

Every claim below already carries an implemented arm in `pkg/sim`. This story cites them because
the behaviour the player now reaches is theirs, not because any of them is being read again for a
value: no number, offset or byte enters this story from research.

| Claim | Confidence | What it settles here |
|---|---|---|
| `AI-ORDER-010` | High | `grpAI+0x20` is the group order byte and has eight live values; `0` hands control to the members' own states. This is why a player Patrol may leave a group at order none and why the engagement pass then decides nothing for it. |
| `AI-GROUPCMD-020` | High | The vocabulary itself: the eleven `Par0` literals, which of them shipped content uses, and which of them write the order byte. `1 Guard`, `2 Swarm`, `3 Stand Ground`, `4 Move`, `5 Swarm 2` and `14 Patrol` are the six with an arm in this build; `10 Attack`, `11 Defend`, `15 Follow` and `17 Roam` are the published ones it does not implement. |
| `AI-GRPGUARD-074` | High | Group order 1 read end to end, including that the walk home targets the member's own POST. FR-1's whole visible consequence, and DD-3's whole reason. |
| `AI-STAND-076` | High | Order 3 is Stand Ground: no radius clip, no walk, and a scorer that vetoes every candidate past reach. FR-2. |
| `AI-STANCE-098` | High | All four stance setters anchor a post, unconditionally for the Stand Ground forms. FR-1 and FR-2's post write, and why it stands beside the order write rather than inside the group release. |
| `AI-POST-095` | High | A guard's post is written by the guard SETTER and not by the per-actor initialiser. Same. |
| `AI-SWARM-022` | High | Order 2 is *walk to the commanded cell and fight what you see*, with the walk decided per member inside the arm rather than by the command. FR-9's cut rests on this: the command itself gives nobody a destination. |
| `AI-MOVE-023` | High | Orders 4 and 5 and the distribution they share. FR-3's March is order 5 through that same distribution. |
| `AI-PATROL-013`, `AI-PATROL-017`, `AI-PATROL-018` | see below | The patrol state, that the `.alm`'s own type-7 script reaches it, and that what the command builds is a two-node ring between the creature's own cell and the commanded one. FR-4. |
| `AI-AUTHOR-015` | High | Which slot's groups load standing their ground and which guard. Read here only to know what a player's units are under before he gives an order. |

`AI-PATROL-013` carries a **partial refutation** and is cited in its amended form: the original
second clause ("no shipped file reaches it") is refuted by `EXP-0083` and reads, now, as "no
shipped file reaches `R0163`". The ring, the state byte and the arm — the parts this story
depends on — are untouched by that amendment, and `AI-PATROL-018` is the row for the ring itself.

## Ours by choice

- **That the player has this vocabulary at all** (spec A-1). Research decodes the group command as
  the map script's authoring surface. Nothing decoded says the original's control panel issues
  these literals, and this story asserts nothing about what it does. What the player may ask for,
  and by what gesture, is authored.
- **The four keys** (spec A-2), by the rule stated there. Keys are authored throughout this
  front-end — `G`, `E`, `M`, `F3`, `F4` all are — and each is written down beside its binding.
- **Arming rather than a cursor mode** (spec A-3), reusing the attack key's own shape.
- **That a player Patrol builds a command group** (spec A-4), with its consequence disclosed.
- **That a player order clears the patrol state** (spec A-5). This is the one place this story
  changes what a world does with no claim behind it. It is authored, it is on the player's path
  only, and the alternative is an order that cannot be given.
- **Which four of the six are exposed and which one is cut** (FR-9).

## Open, and deliberately not opened from here

- **What the original's own control panel binds.** Whether the shipped UI has buttons for these
  orders, and which, is not decoded. This story does not need it: nothing here claims to reproduce
  a placement, and when it is decoded, binding a button is one statement beside each key.
- **Whether the law's own player order clears the actor state.** Not decoded, and not asked for
  from here — the authored rule (A-5) is disclosed and is confined to the player path, so a later
  decode replaces one statement rather than unpicking a behaviour.
- **The four published sub-commands with no arm** (`10`, `11`, `15`, `17`). They stay unimplemented;
  this story deliberately does not give a key to a sub-command whose behaviour this build does not
  have.

## Data read from the lawful install

None for the contract. The census argv in `verification.md` runs `cmd/missionrun` against
`AGAINROM_ASSETS`, which reads the install; no value from it enters the repo.

## Removed

Nothing. No claim was consulted and discarded, and no research item was opened.
