# The second game's chat cheats and debug letters through one command parser

## Intent

Owner direction: implement every cheat and debug command the original
accepts. The second game has its own command text, and the same launch flag
raises the second game's cheat state. One builder per kind: the chat command
parser becomes one parser over a per-game command table, the first game's
commands built unchanged from data and the second game's commands added as
data.

Base: `8b3eab92` (game 0.107.0). Knowledge pin moved from k217 to k218.
Reconciled main: `34ae6dac` (game 0.108.0).

## Authority

Second-game evidence only. No first-game claim backs a second-game behaviour.

| Leaf | Authority |
|---|---|
| one parser, 24 texts, prefix match in fixed order | R2-ENGINE-295 (High) |
| cheats only in campaign mode; host and latency commands in other modes | R2-ENGINE-296 (High / Medium) |
| unlock line, cheat state, reply 5, locked lines dropped | R2-ENGINE-297 (High) |
| replies 5, 6, 7: main.txt 221-226 around the Player name, 5 seconds | R2-ENGINE-298 (High) |
| `#create` | R2-ENGINE-299 (High / Medium) |
| `#modify` | R2-ENGINE-300 (High / Unknown) |
| `#summon`, `#pickup all` | R2-ENGINE-301 (High / Medium) |
| `#killall`, `#kill all`, `#kill cheaters`, `#kill name` | R2-ENGINE-302 (High / Medium) |
| `#show map`, `#hide map`, `#victory`, `#event` | R2-ENGINE-303 (High / Unknown), R2-ENGINE-047 |
| Alt+B..Y | R2-ENGINE-304 (High / Medium) |
| no launch switch in the original | R2-ENGINE-305 (High) |
| the dedicated server | R2-ENGINE-306 (High / Unknown) |
| cheat state not saved; LOAD starts locked | R2-SESSION-135 (High / Medium) |
| money saved; +god, inventory, spellbook in the unit serializer; reveal not saved | R2-SESSION-136 (High / Medium / Unknown) |

The unlock line is the prefix R2-ENGINE-297 states; the code holds that prefix
and nothing more of the installed text.

## As built

### One parser

`pkg/game/chatcommands.go` is the one parser. A line without `#` goes to the
adapter's ordinary chat. A command line goes to the adapter's admission, then
to the first table row whose text it starts with. A row carries its text, its
effect, its gate (`chatAdmitted`, `chatUnlocked` with a refusal reply,
`chatHost`), its success reply and the fields that differ between the games:
an extra console line, case-folded gold, count before the hero token, a reply
naming the target Player, and the `#modify` arms with their replies, spell
count and book test. Each effect is one operation both games call:
`CheatAddGold`, `CheatAddItem`, `CheatGod`, `CheatSpell`, `CheatSummon`,
`CheatKillPlayer`, `CheatPickupAll`, the map reveal, the victory panel, the
event and the unlock.

`pkg/game/chatgames.go` holds the two adapters: the table, the admission rule,
the debug-letter table, the help lines, chat echo, reply time, the launch line
and whether the launch flag also acts at LOAD. The campaign service picks the
adapter (`chat()`) and answers whether a mission runs in the adapter's
campaign (`chatCampaign`). The edition's former `Cheats` flag is removed: both
games now have an adapter.

The first game's table is its 13 texts in the former switch order, each with
its former gate, refusal and replies; its admission is the participant gate.
Every first-game cheat release test passes unchanged.

### Second-game commands

| Command | Behaviour |
|---|---|
| any line, locked Player, campaign | nothing, unless it starts with the unlock prefix: cheat state set, reply 5. The unlock runs no command |
| unlock line, unlocked Player | nothing (no table row matches) |
| line without `#` | not shown; the line is not relayed |
| `#create [N] Gold` | gold compared ignoring case; N added to the Player purse; reply 7 |
| `#create [N] name` | the named item with quantity N into the hero's inventory; reply 7; unknown name reply 6 |
| `#create` with no living hero | reply 6 |
| `#modify self/army +god` | six protection words and six bytes set to 100, then the ordinary derive; reply 7 |
| `#modify self +spell N`, `+spells` | spell N, or spells 1..29, into the hero's book when the hero holds a book or was built as a mage; reply 7 |
| `#modify self +knowledge` | nothing, no reply |
| `#summon [N] [hero] name` | N creatures, else humans, by exact installed name, near the hero, owned by the hero's owner, in a new group; no reply |
| `#killall`, `#kill all` | health -50 on every actor of each Player hostile to the sender; reply 7 |
| `#kill cheaters` | every other unlocked Player locked and its actors set to -50; no reply |
| `#kill name` | the exact-named Player's actors set to -50; reply 7 naming that Player; no match, no reply |
| `#pickup all` | every sack's gold and items to the hero; reply 7 |
| `#show map`, `#hide map` | reveal on, every cell explored; reveal off; reply 7 |
| `#victory` | the ordinary success panel and campaign completion |
| `#event N` | N raised through the mission's script message route, the same route a script raise takes |
| `#kick`, `#locate`, `#set latency`, `#show latency` | outside the campaign only; no connection facility, nothing changes |

Replies use the installed main.txt lines 221-226 around the Player name on
the mission message line for 5 seconds.

Campaign mode: a mission opened while the town holds the destinations
campaign, which New Game and every campaign LOAD do. A direct mission entry
outside it admits only the host and latency texts.

Actor names: the second game keys a placement by the row's server id. The
cheat actor builder now keys a second-game placement by the named row's server
id over a collection that shows only that row, so two rows sharing one server
id cannot swap. `pkg/data` exposes the two server id columns.

### Debug letters

For an unlocked Player Alt+D and Alt+T toggle the turn and script trace flags,
Alt+Q toggles the safe-mode override. None prints a line. Alt+H, Alt+I and
Alt+U do nothing. A locked Player's letters do nothing.

### Launch flag

On a second-game root the starter checkbox and `-chicken` submit the unlock
line at every fresh campaign mission start and at every LOAD into a campaign
mission; the ordinary unlock posts reply 5. Off by default. On a first-game
root it does what it did: `#Chicken` at fresh mission starts only.

### SAV

The cheat state, the reveal and the debug toggles are not saved. Money,
`+god`, inventory, spells and summoned actors reach the SAV through the
ordinary writer and come back on cold LOAD.

## Proof

Release witnesses, run on `rom2-en` and `rom2-ru` in the `TestReleaseSecondCampaign` group:

- `TestReleaseSecondCampaignCheatCommands`: through App input in campaign
  mission 10 with a mage hero. Locked: seven lines and every Alt letter change
  nothing. Unlock with reply 5; the unlock line again and the first game's
  unlock word do nothing. Each command with its state change and reply,
  including the refusal, two summons (humans and a creature), a pickup over an
  ordinary gold drop, the kills against a living hostile Player, the event and
  the victory panel. Alt+D, T, Q toggle and print nothing. SAVE, then cold LOAD
  through the App: purse, god block, inventory, spells and summons equal;
  every actor equal field by field; the loaded mission is locked.
- `TestReleaseSecondCampaignCheatLaunchFlag`: flag on, the fresh mission is
  unlocked with reply 5; a LOAD with the flag on is unlocked with reply 5; a
  LOAD with the flag off is locked; a direct mission entry outside the campaign
  ignores the unlock line and the four host and latency texts.

Focused: `TestChatCommandTablesKeepTheirOrder` (both tables' texts and order,
the sixteen second-game command texts and four modifiers), `TestChatAdapterFollowsTheCampaignService`,
`TestCheatItemKeepsFactoryEffectsAndExactInstalledNames` and
`TestCheatActorUsesItsExactRowAndCompleteConstructor` (second-game item and
server-id actor keying). First-game cheat witnesses: every
`TestReleaseCheat*` and `TestReleaseChicken*` test, unchanged.

## Open debt

- DIV-2774: the launch flag is owner direction (ACCEPTED).
- DIV-2775: help, statistics and toggle text unread; Alt+H, I, U do nothing.
- DIV-2776: host, locate and latency commands have no connection facility.
- DIV-2777: name sets, kind byte, summon placement, `+knowledge`, kill timing,
  the reveal grid and the mage book bridge.
- A summoned actor's source spellbook flag reads back set after SAVE and LOAD,
  in both games: the writer gives every actor an empty present book. The
  witness excepts that one field.
- The engine's second-game hero has no spellbook model; `+spell` treats the
  mage-built hero as holding one.
