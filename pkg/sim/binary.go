package sim

import (
	"encoding/binary"
	"fmt"
)

// formatVersion is the canonical base byte form. Explicit global-healing
// policies add form96 through autohealingbinary.go; absent policies keep their
// historical base bytes. For the base, every version paragraph
// in this comment is the argument for refusing an older form here, and it still
// holds for everything before oldestReadableVersion (saveform.go): the
// shipped versions 1 through 4 included, and every version below 50, still fail
// loudly here instead of decoding to a world whose routing mode, grid, stall
// counts, stored routes, health, script, membership or any of the fields the
// paragraphs below name were invented by the decoder that read it.
//
// Version 8 is where the GROUP RATE TERM arrives, and the trade is the one every
// version before it made. A version-7 form says nothing about a group, and "no
// term" is not a gap for a reader to fill: it is a claim about the simulation —
// every unit in this save moves at its own class speed — and it is exactly the
// claim a save taken mid-formation contradicts. A decoder that asserted it would
// hand back a world whose group walks at the wrong speed for the rest of its
// order, silently, and whose digest then means something else. Version 7 was
// refused on the same trade over the rate and the crossing, version 6's over the
// movement domain and version 5's over health; the answer has not changed.
//
// Version 9 is where THE MISSION SCRIPT arrives — the compiled program, the
// hundred registers, the thousand fire-once latches, the two outcome counters and
// the outcome — and the trade is the sharpest one yet. A version-8 form says
// nothing about a script, and "no script" is not a gap a reader may fill: a world
// resumed with empty latches re-fires every one-shot trigger the mission has
// already spent, so an accepted version-8 form would replay a mission's whole
// script from the beginning while looking exactly like a resumed one.
//
// Version 10 is where the ATTACK ORDER, its cycle and the seven numbers a blow
// reads arrive, and the trade is the one every version before it made. A
// version-9 form says nothing about a fight, and "no fight" is not a gap for a
// reader to fill: it is a claim about the simulation — nobody in this save is
// attacking anybody, and every unit in it is unarmed — and it is exactly the
// claim a save taken mid-swing contradicts. A decoder that asserted it would
// hand back a world where a blow already charged never lands, silently, and
// whose digest then means something else.
//
// The two newest sections are in DIFFERENT PLACES and neither moved the other:
// the script closes the whole form, after the routes, while the attack block is
// the entity record's own tail. That is why version 9 and version 10 could be
// written by two hands at once and still compose — but only ONE number can name
// the result, and this is it.
//
// Version 11 is where GROUP MEMBERSHIP arrives, and it widens TWO records in one
// bump: the entity's, whose tail takes the group word, and the compiled check's,
// whose tail takes the group a check names and the byte saying whether it names
// one. Both are tails, so no offset before them moves. The trade is the one every
// version before it made and it is worth stating in its own terms: a version-10
// form says nothing about membership, and "no membership" is not a gap for a
// reader to fill — every entity in such a form would read as a member of group
// zero, which is a real group, so a script's group check would count units that
// were never in it. That is a wrong answer rather than a missing one, and it is
// why the earlier version is refused rather than migrated.
//
// Version 13 is where THE COST PLANE AND THE HEIGHT PLANE arrive, and it is the
// first bump since version 8 that widens the form OUTSIDE the entity record.
// Both planes follow the block plane, each of the length the header's one cell
// count already declares — so no header field is added, no offset before the
// entity records moves, and the record itself does not change by one byte.
//
// One declared count serves all three planes deliberately. Three counts would be
// three numbers free to disagree, and a decoder would then have to decide which
// one wins; with one, a plane of the wrong length is not representable.
//
// The trade is the usual one and it bites in the usual place. A version-12 form
// says nothing about either plane, and "no cost plane" is not a gap for a reader
// to fill: the two planes are read by a search and by the rate law, so a decoder
// that supplied its own would hand back a world whose units route differently
// and cross at different speeds from the world the bytes were cut from. It is
// also structural twice over, as version 12's was: the records move by twice the
// cell count, so a version-12 buffer read against these offsets is a misparse
// rather than a wrong plane, and both planes are canonical state that enters the
// digest.

// Version 12 is where OWNERSHIP arrives, and it widens two records in one bump
// for the reason version 11 did: the entity's, whose tail takes the owner slot,
// and the compiled INSTANT's, whose tail takes the unit, group and player an
// instant names and a presence byte for each. Both are tails, so no offset before
// them moves.
//
// Its trade differs from every one above it in a way worth stating exactly,
// because the usual argument does not apply. Nothing in this build READS an
// owner, so a version-11 form accepted here would not make any tick answer
// differently — the argument that a wrong answer is worse than a missing one has
// nothing to bite on. The version still moves, on the two grounds that survive:
// the two records changed WIDTH, so a version-11 buffer read against version 12's
// offsets is not a wrong owner but a misparse of every record after the first;
// and an owner is canonical state that enters the digest, so a form that carried
// none would decode to a world whose digest is not the digest of the world it was
// cut from. Both are structural, and neither needs a reader to exist.
//
// Version 14 is where THE FACING arrives — one byte at the entity record's own
// new tail, so no offset before it moves and only the record's WIDTH does. Its
// trade is version 12's and 13's rather than the eleven versions' before them:
// nothing in this build READS a facing, so a version-13 form accepted here would
// not make any tick answer differently, and the argument that a wrong answer is
// worse than a missing one has nothing to bite on. The version moves on the two
// grounds that survive without a reader — the record changed WIDTH, so a
// version-13 buffer read against these offsets is not a wrong facing but a
// misparse of every record after the first; and a facing is canonical state that
// enters the digest, so a form carrying none would decode to a world whose
// digest is not the digest of the world it was cut from.
//
// A VERSION-13 BUFFER IS THE SHARP CASE and it is worth naming, because it is
// the one an earlier build of this tree actually wrote. Its header, its three
// planes and every offset inside its first record are identical to this
// version's; what differs is one byte at the end of each record, so the SECOND
// record and every record after it would be read one byte early, out of the
// middle of its neighbour, and the first would survive intact. Nothing but the
// version byte separates a correct decode from that.
//
// Version 16 is where THE RELATION arrives — the fixed 2500-byte matrix saying
// which roster slot treats which as an enemy — and it is the second bump since
// version 8 to widen the form outside the entity record. The block CLOSES the
// form, after the script section, at a length this build fixes at compile time,
// so it declares no count of its own: a count would be a number free to disagree
// with the constant, and a decoder would then have to pick a winner. Closing the
// form is also what leaves every offset above it where version 14 put it. Version
// 15 is allocated to another lane and this build never wrote one — see
// the layout note below for why a section indexed by neither header count is the
// one that belongs there.
//
// Its trade is the eleven versions' rather than 12's, 13's and 14's, because
// this one has a READER. A version-14 form says nothing about a relation, and
// "no relation" is not a gap a reader may fill: it is the claim that nobody in
// this save is hostile to anybody, and a save cut mid-battle contradicts it
// exactly. A decoder that supplied it would hand back a world in which every
// fight in progress is the last fight — the attackers keep their orders and
// nothing is ever selected again — silently, and whose digest then means
// something else.
//
// It is structural as well, which is what a bump needs when a reader alone is
// not enough — though the structure it breaks is a LENGTH and not an offset,
// which makes this bump the odd one of the sixteen. The block closes the form,
// so nothing above it moved; a version-15 buffer is simply 2500 bytes shorter
// than this reader requires, and it fails the script section's own
// exact-consumption check rather than parsing into a wrong world.
//
// Version 17 is THE DECAY LADDER — the stage a body is at, the dwell it still
// owes before it is torn down, and the dying time a fresh body takes that dwell
// from — and it is back inside the entity record, where every bump but 16's has
// been.
//
// It carries all three of the grounds a bump has ever rested on, which no bump
// since 11 has. There is a READER, and a sharp one: a version-16 form says every
// body in it is freshly fallen and none has begun to decay, which is a CLAIM and
// not a gap — a save cut a minute after a battle contradicts it exactly, and a
// decoder that supplied it would hand back a world whose skeletons stand up
// again and whose released ground is blocked. It is STRUCTURAL: the record goes
// from 92 bytes to 99, so a version-16 buffer read against these offsets is a
// misparse of every record after the first. And the stage is canonical state
// that enters the DIGEST, so a form carrying none decodes to a world whose
// digest is not the digest of the world it was cut from.
//
// Version 18 is where THE SIGHT RANGE arrives — one byte at the entity record's
// new tail, so no offset before it moves and only the record's WIDTH does. It is
// version 14's shape in every structural respect and version 16's and 17's in
// the one that matters: this block HAS A READER. A version-17 form says nothing
// about a range, and "no range" is not a gap a reader may fill. Filled with zero
// it is the claim that nobody in this save can see past the cell they stand on,
// so every guarding group stops acquiring the moment the bytes are read back;
// filled with any constant it is the claim this story exists to delete, and a
// decoder that supplied one would hand back a world whose groups engage a
// different set from the world the bytes were cut from — silently, and with a
// digest that then means something else.
//
// THE TWO ARRIVED IN PARALLEL, in two lanes off one master, and the record's
// tail is where both landed: 17 appended seven bytes and 18 appended one after
// them, so the record went 92 to 99 to 100 and no offset in front of the decay
// stage moved for either. Only ONE number can name the result and it is 18 — the
// same rule versions 9 and 10 were resolved under.
//
// Version 19 is where THE GROUP SECTION arrives — one record per owned
// (owner, group) pair, each carrying that group's notice base FROZEN when the
// world was built — and it is unlike every bump since 8: it is neither a tail
// on the entity record nor a block closing the whole form, but a new COUNTED
// section of its own, sitting between the routes and the script section.
//
// It fits there because both of its neighbours still hold their own contract
// exactly: the routes are read by their own reader, which reports how much of
// the buffer it consumed, and the script section is still required to consume
// what is left of the buffer to the byte — a contract that is unaffected by
// where its own beginning is measured from. Every offset the previous version
// fixed — the header, the three planes, the entity records and the routes — is
// therefore unmoved; what moves is where the script section and the relation
// after it are found, and both are already found by measurement rather than by
// a fixed offset.
//
// Its trade is the sharpest kind there is: a version-18 form says nothing
// about a group's frozen radius, and "no record" is not a gap a reader may
// fill — every group in such a form would have to have its base invented, and
// an invented base is exactly the divergence this story exists to remove. A
// decoder that supplied one would hand back a world whose guarding groups
// notice a different set from the world the bytes were cut from, silently,
// and whose digest then means something else.
//
// Version 20 is where THE GROUP ORDER AND THE COMMANDED CELL arrive — a byte
// and a coordinate pair on every record the group section already carries —
// and it GROWS THAT SECTION IN PLACE rather than adding a new one: the group
// record's own count, its position between the routes and the script section,
// and every offset before it are unmoved, exactly as version 19 left them.
// Only the record's own WIDTH changes, from 9 bytes to 18.
//
// Its trade is the sharpest kind there is, for the same reason version 19's
// was: a version-19 form says nothing about a group's order, and "no order"
// is not a gap a reader may fill — every group in such a form would have
// to have its order invented, and an invented order is exactly the
// divergence this story exists to remove (a decision now reads the STORED
// order, where nothing before this version stored one at all). A decoder
// that supplied one would hand back a world whose groups decide under a
// different order from the world the bytes were cut from, silently, and
// whose digest then means something else. The commanded cell rides the same
// bump for the same reason: it is state the record did not carry a moment
// ago, and "uncommanded" is a claim a script-driven save can contradict.
//
// Version 21 is where THE ACTOR STATE arrives — the per-entity byte that
// runs a state machine when its group is not deciding for it, and the
// two-cell ring with the leg that names an actor's current waypoint
// (0099) — and it is the entity record's own new tail, appended after the
// sight range on version 14's and 18's own grounds: no offset before it
// moves and only the record's WIDTH does, from 100 bytes to 118.
//
// THE SAME BUMP IS WHERE GROUP ORDER 0 BECOMES LEGAL STORED STATE, and that
// half widens no record and moves no offset at all — the group record's
// order byte has sat at this place since version 19 and at this width since
// version 20, and what changes is the SET this reader accepts there, not the
// bytes.
//
// A version-20 form says nothing about an actor's own state, and "no
// state" is not a gap a reader may fill: it is the claim that no entity in
// this save is anything but guard, and it is exactly the claim a save cut
// mid-patrol contradicts. A decoder that supplied guard to every entity
// would hand back a world in which a patrolling actor has forgotten its
// ring and its leg, silently, and whose digest then means something else.
//
// Version 22 is where THE GROUND SACKS arrive — the map's own authored
// loot, one entry per occupied cell, its cell, its purse and its item codes
// — and it is unlike every bump since 8 except one: it is neither a tail
// on the entity record nor a block closing the whole form, but a new COUNTED
// section of its own, sitting between the GROUP section and the script
// section — exactly where version 19's own section landed, and for the
// same reason (0095). It fits there because the group section already
// reports how much of the buffer it consumed and the script section is still
// required to consume what is left exactly, and neither contract cares where
// its own start is measured from. Every offset the previous version fixed
// — the header, the three planes, the entity records, the routes and the
// group section — is therefore unmoved; what moves is where the script
// section and the relation after it are found, and both are already found by
// measurement rather than by a fixed offset.
//
// Its trade is the sharpest kind there is, on the group section's own
// grounds: a version-21 form says nothing about a ground sack, and "no
// sacks" is not a gap a reader may fill — thirty of thirty-eight shipped
// maps author one, and a script that gates a trigger on "the sack is still
// there" (the check this story also gives an arm) would read every such
// cell as empty on a save cut after a sack had already been placed there. A
// decoder that supplied no sacks would hand back a world whose ground holds
// less than the world the bytes were cut from, silently, and whose digest
// then means something else.
//
// Version 23 is where THE REACH arrives — one byte at the entity record's new
// tail, so no offset before it moves and only the record's WIDTH does, from
// 118 to 119 (0104). It is version 18's shape in every structural respect —
// no offset in front of the patrol leg moves, only the record's WIDTH does —
// and its trade is the one a bump has whenever the block HAS A READER. A
// version-22 form says nothing about a reach, and "no reach" is not a gap a
// reader may fill: filled with the constructor's own floor of 1 it is the
// claim that no unit in this save carries a weapon with any range at all —
// every archer, every sonic attacker, every siege engine reads as a
// spearman — which is exactly the claim this story exists to correct, and a
// save holding one bow-armed unit contradicts it. A decoder that supplied
// one would hand back a world whose bow-carrying units close to melee range
// and strike from a different cell than the world the bytes were cut from,
// silently, and whose digest then means something else.
//
// A VERSION-22 BUFFER IS THIS ONE'S SHARP CASE, on the facing's and the sight
// range's own ground: header, planes and every offset inside the first
// record's first 118 bytes are identical, so the SECOND record and every one
// after it would be read one byte early, out of the middle of its neighbour,
// while the first survives intact. Nothing but the version byte separates a
// correct decode from that.
//
// Version 24 is where THE POST arrives — two int32 at the entity record's
// new tail, so no offset before it moves and only the record's WIDTH does,
// from 119 to 127 (0106). It is version 21's shape in every structural
// respect — no offset in front of the reach moves, only the record's WIDTH
// does — and its trade is the one a bump has whenever the block HAS A
// READER: the guard stance's own walk home, which this story adds beside
// the field (engage.go's walkHome). A version-23 form says nothing about a
// post, and "no post" is not a gap a reader may fill: it is the claim that
// every idle guard's anchor is whatever the decoder happens to invent, and a
// save cut with a group standing off its post contradicts it exactly. A
// decoder that supplied one would hand back a world whose idle guards walk
// home to a different cell than the world the bytes were cut from,
// silently, and whose digest then means something else.
//
// A VERSION-23 BUFFER IS THIS ONE'S SHARP CASE, on the reach's own ground:
// header, planes and every offset inside the first record's first 119 bytes
// are identical, so the SECOND record and every one after it would be read
// eight bytes early, out of the middle of its neighbour, while the first
// survives intact. Nothing but the version byte separates a correct decode
// from that.
//
// Version 25 is where REGENERATION arrives — the mana pool, the mana
// maximum, the two regeneration periods and the two hundredths remainders,
// eighteen bytes at the entity record's new tail, so no offset in front of
// them moves and only the record's WIDTH does, from 127 to 145. It is
// version 21's and version 24's shape in every structural respect — no
// offset in front of the reach moves, only the record's WIDTH does — and
// its trade is the one a bump has whenever the block is canonical state: a
// version-24 form says nothing about a pool or a remainder, and "no mana" is
// not a gap a reader may fill — a decoder that supplied one would hand
// back a world whose units heal on a different schedule than the world the
// bytes were cut from, silently, and whose digest then means something else.
//
// A VERSION-24 BUFFER IS THIS ONE'S SHARP CASE, on the post's own ground:
// header, planes and every offset inside the first record's first 127 bytes
// are identical, so the SECOND record and every one after it would be read
// starting eighteen bytes into its neighbour, while the first survives
// intact. Nothing but the version byte separates a correct decode from
// that.
//
// Version 25 appends THE REGENERATION BLOCK at +127, on the tail's own
// terms again (0109): no offset in front of it moves, and only the
// record's WIDTH does, from 127 to 145. Mana sits at +127, MaxMana at
// +131, HealthRegenPeriod at +135, ManaRegenPeriod at +139,
// HealthHundredths at +143 and ManaHundredths at +144. The four int32 are
// carried WHOLE with no value refused, on the health pair's own ground:
// every int32 is a state the constructor can build, so refusing one here
// would make a world this package can produce a world it cannot read
// back. THE TWO REMAINDERS ARE THE ONE PAIR HERE THAT IS CHECKED — each
// against 99, regenFault's whole content — because a value above 99 has
// no fraction of a point left to name; the constructor folds the same
// shape to 0 rather than refusing it, on reachFault's own split.
//
// A VERSION-24 BUFFER IS THIS ONE'S SHARP CASE too, on the record side:
// its header, planes and every offset inside the first record's first
// 127 bytes are identical, so the SECOND record and every one after it
// would again be read starting eighteen bytes into its neighbour.
// Nothing but the version byte separates a correct decode from that.
//
// Version 26 is where THE CONTAINER AND THE PURSE arrive — what an actor
// carries and what a roster slot holds in gold — and it is unlike every
// bump since 22 except that one: two new sections, both sitting between the
// sack section and the script section rather than a tail on the entity
// record or a block closing the form. Both fit there on the sack section's
// own grounds: the sack reader reports how much of the buffer it consumed,
// and the script section is still required to consume exactly what is left
// after them, so a pair of sections arriving between two contracts that are
// each found by measurement moves nothing either side of it. Every offset
// the previous version fixed — the header, the three planes, the entity
// records, the routes, the groups and the sacks — is therefore unmoved.
//
// THE CARRY SECTION carries one record per entity, in entity order — the
// route section's own rule, restated for a different field: it declares
// no count of its own, because the entity count the header already
// declares is the same number and a second count would be free to
// disagree with it. Each record is a uint32 code count then that many
// little-endian uint16 codes, and an entity carrying nothing writes a
// count of 0 and nothing after it — the same "no absent case" the route
// section and the sack section's own item lists already write under.
//
// THE PURSE SECTION follows it, a fixed relationSlots-wide block of
// little-endian uint32 with no length of its own — the relation block's
// own rule, restated for a section that does not close the form: its
// length is a compile-time constant of this build, so it is not a number
// free to disagree with anything, and it is written WHOLE for every
// world exactly as the relation is.
//
// Its trade is the sharpest kind there is, on the sack section's own
// grounds: a version-25 form says nothing about a container or a purse,
// and "nothing carried, no gold" is not a gap a reader may fill — it is
// the claim that no actor in this save has ever picked anything up, and
// a save cut after a pick-up contradicts it exactly. A decoder that
// supplied empty containers and empty purses would hand back a world
// whose actors hold less than the world the bytes were cut from,
// silently, and whose digest then means something else.
//
// A VERSION-25 BUFFER IS THIS ONE'S SHARP CASE, on the sack section's own
// kind: header, planes, every entity record, the routes, the groups and
// the sacks are this version's exactly, so only the CARRY SECTION, THE
// PURSE, THE SCRIPT SECTION AND THE RELATION move — a version-25 buffer
// is short by however many bytes this reader now expects before the
// script section, and it fails either the carry section's own per-record
// bounds check or the script section's own exact-consumption check,
// rather than parsing into a world with a container or a purse that is
// not there.
//
// Version 31 is where a compiled CHECK'S SECOND PLAYER REFERENCE arrives —
// Player and Player2, carried like Group rather than resolved like Unit,
// because check opcode 10 is the one arm of the vocabulary that names two
// players at once. It is a tail append on the check record alone: Player at
// +63, Player2 at +67, HasPlayer at +71 and HasPlayer2 at +72, so every
// offset the previous version fixed — the header, the planes, the entity
// records, the routes, the groups, the sacks, the carry section, the purse
// and even the group pair right in front of it on the same record — is
// unmoved. Only the check record's WIDTH changes, from 63 to 73.
//
// VERSIONS 27 THROUGH 30 ARE SKIPPED, and that is disclosed rather than
// left for a reader to wonder about: three of them are already reported
// out to sibling lanes and the fourth is unaccounted for, on the ground
// version 16's own comment above gives for 15 — allocated elsewhere and
// this build never wrote one. THE NUMBER ITSELF IS UNALLOCATED, and that
// is a different fact from the four gaps beside it: this story was built
// with no channel back to the seat that hands version numbers out, so 31
// was picked in the lane, chosen above every number already known to be
// out to a sibling. It is written in exactly ONE place — this constant —
// and renumbering it, should the seat that allocates them pick
// differently, is a single edit here; every digest that names it moves
// with it, recomputed by running the tests rather than by hand.
//
// Its trade is the sharpest kind there is, on the owner and the purse's
// own grounds: a version-30-or-earlier form says nothing about a player
// reference, and "no player" is not a gap a reader may fill — it is the
// claim that no check in that program measures anything about a player,
// and a program authoring check 8, 10 or 15 over one contradicts it
// exactly. A decoder that supplied absent references would hand back a
// world whose checks measure less than the world the bytes were cut from,
// silently, and whose digest then means something else.
//
// A VERSION-26 BUFFER IS THIS ONE'S SHARP CASE, on version 13's own kind:
// header, planes, every entity record, the routes, the groups, the sacks,
// the carry section and the purse are this version's exactly, and so is
// the FIRST check record's first 63 bytes — so a script with no check, or
// only one, still parses. The SECOND check record and every one after it,
// and the instant and trigger arrays that follow them, are read starting
// ten bytes into their neighbour. Nothing but the version byte separates a
// correct decode from that.
//
// Version 32 is where THE COMMAND GROUP arrives (0117) — one word at the
// entity record's new tail, `CommandGroup` at +145, on version 12's and
// 14's own terms: no offset in front of it moves and only the record's
// WIDTH does, from 145 to 149.
//
// IT IS THE THIRD NUMBER THIS STORY HAS CARRIED, and saying so is the
// point rather than an apology. The field was written against 27, merged
// forward onto 26's landing as 27, and lands here at 32 because 31 landed
// first; the story's own layout never moved through any of it, because the
// three stories it merged past — the container and the purse (sections
// after the sacks), the compiled check's player pair (a tail on the CHECK
// record) and this one (a tail on the ENTITY record) — widen three
// different places in the form and meet only in this constant and in the
// pins downstream of them. A version number is allocation order and
// nothing else; what a merge has to establish is that the LAYOUTS do not
// collide, and here they do not.
//
// Its trade is the sharpest kind there is: a version-31 form says nothing
// about a command group, and "no command group" is not a gap a reader may
// fill — it is the claim that no actor in this save has ever been given an
// order, and a save cut after one contradicts it exactly. A decoder that
// supplied a zero would hand back a world whose commanded actors are back
// in the groups the map placed them in, deciding under the stance the
// constructor froze rather than the order their orders built.
//
// A VERSION-31 BUFFER IS THIS ONE'S SHARP CASE, on the regeneration
// block's own ground: header, planes and every offset inside the first
// record's first 145 bytes are identical, so the SECOND record and every
// one after it would be read starting four bytes into its neighbour, while
// the first survives intact — and the routes, groups, sacks, carry, purse
// and script blocks past them are then all found by measurement from the
// wrong place. Nothing but the version byte separates a correct decode
// from that.
//
// VERSIONS 33 AND BELOW THAT THIS BUILD NEVER WROTE ARE ALLOCATED
// ELSEWHERE, on version 16's own precedent — its own paragraph names
// version 15 as one that build never wrote. 27 through 30 are the four
// the version-31 paragraph above already discloses; 33 is out to a lane
// running in parallel off the same master this one branched from. This
// build carries none of their sections.
//
// Version 34 is where EQUIPMENT arrives — twelve item-code slots per
// entity, in slot order — a fifth block sitting between THE CARRY SECTION
// and THE PURSE SECTION rather than a tail on the entity record or a block
// closing the form. It fits there on the carry section's own grounds:
// neither of its neighbours carries a length a section arriving between them
// could disturb — decodeCarried reports how much of the buffer it
// consumed, and the purse section is a fixed compile-time width the same as
// the relation's — so a block landing between two contracts that are each
// found by measurement or by constant moves nothing either side of it. Every
// offset the previous version fixed — the header, the three planes, the
// entity records, the routes, the groups, the sacks and the carry section
// — is therefore unmoved.
//
// UNLIKE THE CARRY SECTION IT DECLARES NO PER-RECORD COUNT EITHER, which is
// what makes it the purse section's shape and not the carry section's own: a
// slot's NUMBER is the whole of what it means, so every one of the twelve is
// written whether it is empty or not, on the passability grid's own reason
// for writing every cell whether it is open or not — a section that
// omitted the empty slots could not say which of the eleven it had omitted.
// equipRecordLen is therefore a plain compile-time constant, EquipSlots*2,
// exactly as purseLen is relationSlots*4, and one record per entity in
// entity order is the section's whole length: nothing here is free to
// disagree with the entity count the header already declares.
//
// ZERO IS EMPTY AND NOTHING IS REFUSED, which is where this bump's trade
// inverts THE CARRY SECTION's own. A decoded carried code of zero is
// refused, because a container can never legitimately hold one (carryFault)
// — but an item code's class field never resolves to a real item at row 0
// either (ITEM-CODE-029), so the two fields answer the same underlying fact
// in opposite directions: a container slot can never legitimately hold a
// zero, and an equipment slot can never legitimately hold anything BUT one
// at rest. decodeEquipment therefore checks nothing past the section's own
// span — every uint16 is a legal slot, empty included.
//
// Its trade is nonetheless the sharpest kind there is, on the carry
// section's own argument: a version-33 form says nothing about what an
// actor is WEARING, and "nothing worn" is not a gap a reader may fill — it
// is the claim that no fight in this save has ever been fought in anything
// but bare hands, and a save cut after a single equip contradicts it
// exactly. A decoder that supplied empty equipment would hand back a world
// whose actors fight with different numbers than the world the bytes were
// cut from, silently, and whose digest then means something else.
//
// A VERSION-33 BUFFER THAT IS OTHERWISE WELL FORMED IS THIS ONE'S SHARP
// CASE, on the carry section's own kind: header, planes, every entity
// record, the routes, the groups, the sacks and the carry section are this
// version's exactly, so only THE EQUIPMENT SECTION, THE PURSE, THE SCRIPT
// SECTION AND THE RELATION move — such a buffer is short by exactly
// equipRecordLen*len(entities) bytes, and it fails the purse section's own
// fixed-width check or the script section's own exact-consumption check,
// rather than parsing into a world whose actors are all unarmed when the
// bytes they were cut from say otherwise.
//
// THIS STORY'S OWN NUMBER IS 35, allocated to it by the seat that hands
// version numbers out rather than picked in the lane: 33 went to 0119,
// whose chargen work never touches this file, so it never gained a
// paragraph of its own here; 34 is 0124's EQUIPMENT bump, immediately
// above, and it is the live version this branch merges onto rather than a
// number to route around. 35 sits directly above it, with no gap between
// them. It is written in exactly ONE place — this constant — and every
// digest that names it was obtained by running the tests, never invented
// and never carried over from a stale pin.
//
// Version 35 is where AN ENTITY'S EXPERIENCE FROM USE arrives (0125) —
// the six per-slot experience integers it has earned, its own Mind, its
// own experience value, the slot a gain it earns is credited to, and
// whether its class gains at all — thirty-four bytes at the entity
// record's new tail, on version 32's own terms: no offset in front of it
// moves, and only the record's WIDTH does, from 149 to 183.
//
// Its trade is the sharpest kind there is: a version-34 form says nothing
// about a unit's experience, and "no experience" is not a gap a reader may
// fill — it is the claim that no unit in this save has ever landed or
// taken a blow, and a save cut mid-mission contradicts it exactly. A
// decoder that supplied six zeros, a zero Mind, a zero experience value, a
// credited slot of General and a gains flag of false would hand back a
// world whose panel states a different number for the same character than
// the world the bytes were cut from, silently, and whose digest then means
// something else.
//
// A VERSION-34 BUFFER IS THIS ONE'S SHARP CASE, on the command group's own
// ground: header, planes and every offset inside the first record's first
// 149 bytes are identical, so the SECOND record and every one after it
// would be read starting thirty-four bytes into its neighbour, while the
// first survives intact. Nothing but the version byte separates a correct
// decode from that.
//
// THIS STORY'S OWN NUMBER IS 36, allocated to it by the seat that hands
// version numbers out rather than picked in the lane: 35 is 0125's own
// EXPERIENCE bump, immediately above, and it is the live version this
// story merges onto rather than a number to route around. 36 sits
// directly above it, with no gap between them. It is written in exactly
// ONE place — this constant — and every digest that names it was obtained
// by running the tests, never invented and never carried over from a
// stale pin.
//
// Version 36 is where AN ENTITY'S OWN SPELLBOOK AND THE WORLD'S SPELL TABLE
// arrive (FR-4b) — a bitmask appended to every entity record and a new
// counted section appended to the form, both on version 32's own terms: no
// offset in front of either moves, and only the record's WIDTH does, from
// 183 to 187, and the form's own length past the purse section.
//
// A VERSION-35 BUFFER IS THIS ONE'S SHARP CASE, and it is two sharp cases
// rather than one, on the command group's own kind for the first and the
// carry section's own kind for the second. On the record side: header,
// planes and every offset inside the first record's first 183 bytes are
// identical, so the SECOND record and every one after it would be read
// starting four bytes into its neighbour, while the first survives
// intact. On the section side: even a version-35 buffer whose every
// record decoded correctly carries no spell table at all, so what this
// reader takes for the table's own count is the script section's first
// two bytes — a register, misread — and nothing but the version check
// stands between that stream and a decode that puts the rest of the
// script where the spell table's own records belong.
//
// Version 38 is where THE INSTANT'S SECOND UNIT REFERENCE arrives — Unit2
// and HasUnit2, appended to the instant record's own tail on the terms the
// check record's own tail additions already set: a uint32 value at +59, its
// presence flag at +63, with no offset in front of either moving and the
// record's own width going from 59 to 64.
//
// THIS STORY'S OWN NUMBER IS 38, and 37 is deliberately skipped rather than
// taken as the version-36 form's plain successor: 37 is allocated to a
// story running in parallel off the same master this one branched from,
// and this tree carries none of its change, so there is no version-37 form
// this build ever wrote. A buffer that merely DECLARES 37 is refused for
// that reason alone and not because its bytes are malformed on this
// tree's own terms — it may be a perfectly well-formed form under the
// branch that actually defines it, one this decoder was never told how to
// read, and the version check turns that into a refusal instead of a
// silent misreading the way every version gap above it already does.
//
// Version 39 is where AN ENTITY'S SIX SKILL LEVELS arrive — one int32 per
// slot, appended to the entity record's own tail on every widening's own
// terms: no offset in front of it moves, and only the record's WIDTH does,
// from 187 to 211.
//
// Its trade is the sharpest kind there is, on the experience block's own
// argument (version 35 above): a version-38 form says nothing about a unit's
// skill levels, and "no levels" is not a gap a reader may fill — a level
// is state in its own right once a slot has been raised once, so an absent
// level and a level recomputed from experience would silently disagree with
// each other the moment the two diverge. A decoder that supplied six zeros
// would hand back a world whose panel states a different level for the same
// character than the world the bytes were cut from, silently, and whose
// digest then means something else.
//
// A VERSION-38 BUFFER IS THIS ONE'S SHARP CASE, on the known-spells tail's
// own ground: header, planes and every offset inside the first record's
// first 187 bytes are identical, so the SECOND record and every one after
// it would be read starting twenty-four bytes into its neighbour, while
// the first survives intact. Nothing but the version byte separates a
// correct decode from that.
//
// Version 41 is where A WEAPON'S OWN SPELL arrives on an entity — the id
// and the level a caster's weapon-borne cast releases in place of a strike,
// appended to the entity record's own tail on version 39's own terms: no
// offset in front of either moves, and only the record's WIDTH does, from
// 211 to 217 — WeaponSpell a uint16 at +211, WeaponSpellLevel an int32 at
// +213. THE FOURTH ATTACK PHASE ARRIVES BESIDE IT, on no offset and no width
// of its own: AttackCasting is a value the phase byte at +49 already had
// room for, so an entity mid-cast changes nothing about the record's SHAPE,
// only about which of that byte's values a decoded actor may hold —
// attackFault's own new arm (combat.go), not a new offset here.
//
// THIS STORY'S OWN NUMBER IS 41, and 40 is deliberately skipped rather
// than taken as the version-39 form's plain successor: 40 is allocated to
// a story running in parallel off the same master this one branched from,
// and this tree carries none of its change, so there is no version-40
// form this build ever wrote. A buffer that merely DECLARES 40 is refused
// for that reason alone and not because its bytes are malformed on this
// tree's own terms — it may be a perfectly well-formed form under the
// branch that actually defines it, one this decoder was never told how to
// read, and the version check turns that into a refusal instead of a
// silent misreading the way every version gap above it already does.
//
// A VERSION-39 BUFFER IS THIS ONE'S SHARP CASE, on the skill block's own
// kind: header, planes and every offset inside the first record's first
// 211 bytes are identical, so the SECOND record and every one after it
// would be read starting six bytes into its neighbour, while the first
// survives intact. Nothing but the version byte separates a correct decode
// from that.
//
// Version 45 is where AN ENTITY'S AUTOCAST AND ITS SPELL EFFECT MARK arrive
// — the spell it casts unbidden, the ticks before it may do so again, and
// the ticks and spell id of the mark a landed cast leaves on it — appended
// to the entity record's own tail on version 41's own terms: no offset in
// front of them moves, and only the record's WIDTH does, from 217 to 222.
//
// Its trade is the one every version above it made, and it bites in two
// places. A version-44 form says nothing about an autocast, and "no
// autocast" is not a gap a reader may fill: the toggle is the player's own
// setting and a save taken with it on contradicts the claim exactly. A
// decoder that supplied a zero id for every entity would hand back a world
// where a unit the player put on repeat stands still, silently, and whose
// digest then means something else. The mark is the same trade at a
// shorter range: a form supplying zeros for a world stepped mid-cast would
// clear an effect the bytes were cut with.
//
// 42 AND 43 WERE ALLOCATED TO PARALLEL WORK AND SKIPPED, and 45 is this
// story's own number rather than 44's plain successor for version 41's own
// reason: a buffer merely DECLARING a skipped number is refused because
// this tree was never told how to read it, not because its bytes are
// malformed on this tree's terms.
//
// A VERSION-44 BUFFER IS THIS ONE'S SHARP CASE, on the weapon spell's own
// kind: header, planes and every offset inside the first record's first 217
// bytes are identical, so the SECOND record and every one after it would be
// read starting five bytes into its neighbour, while the first survives
// intact. Nothing but the version byte separates a correct decode from that.
// Version 46 is where THE COMPILED INSTANT'S ITEM REFERENCE arrives: a
// packed item code and its presence flag, three bytes appended at the
// instant record's own tail on version 45's own terms — no offset in front
// of them moves, and only the instant record's WIDTH does, from 64 bytes to
// 67.
//
// Its trade is the one every version above it made. A version-45 form says
// nothing about a script node's item, and "no item" is not a gap a reader
// may fill: it is the claim that no instant in this save's compiled program
// hands out or takes away an item, and a save taken inside any of the eight
// maps that author one contradicts it. A decoder that supplied a zero code
// and a clear flag would hand back a world whose quest item is never
// created and never consumed, silently, and whose digest then means
// something else.
//
// A VERSION-45 BUFFER IS THIS ONE'S SHARP CASE, on the autocast's own kind:
// header, planes, every entity record and every byte up to the instant
// array are identical, so the FIRST instant record decodes intact and the
// SECOND and every one after it are read starting three bytes into a
// neighbour. Nothing but the version byte separates a correct decode from
// that.
//
// 46 IS 45'S PLAIN SUCCESSOR and skips nothing. This story was written
// against a tree at 44 while 45 was out to parallel work, and it took 46 on
// that ground; 0154 has since landed and this tree now carries its change,
// so the gap the number was chosen to leave is closed and 46 is simply the
// next version. Version 48 appends THE OFF-MAP BIT to the entity record: one
// 0/1 byte at +222, widening the record from 222 to 223.
//
// Its trade is the one every version above it made. A version-46 form says
// nothing about map presence, and "on the map" is not a gap a reader may
// fill: it is the claim that no unit in this save was taken off the map by
// the mission script, and a save taken inside any of the twelve maps that
// author a removal contradicts it. A decoder that supplied a clear bit
// would hand back a world in which a unit the mission removed stands on the
// map again, silently, and whose digest then means something else.
//
// A VERSION-46 BUFFER IS THIS ONE'S SHARP CASE, on the item code's own
// kind: header, planes, every byte of the FIRST entity record and every
// offset in front of +222 are identical, so the first record decodes intact
// and the SECOND and every one after it are read starting one byte into a
// neighbour. Nothing but the version byte separates a correct decode from
// that.
//
// 48 SKIPS 47, and the skip is deliberate: 47 was allocated to parallel work
// at the same story boundary this one was, and a number returned by a story
// that does not use it is not reused. The authority for the live version is
// this constant, never a summary elsewhere.
//
// 49 IS 0165: the CASTING SECTION between the spell table and the script — the
// mission script's pending casts and the area effects standing on cells — and
// four bytes appended to every spell record for the area-duration column, with
// two more bits defined in its flags byte. A VERSION-48 BUFFER IS THIS ONE'S
// SHARP CASE: the header, the planes, every entity record and every section down
// to the spell count are identical, so a decoder that skipped the version byte
// would read the first spell record intact and every record after it starting
// four bytes into a neighbour. Nothing but the version byte separates a correct
// decode from that.
//
// 51 AND 52 WERE ALLOCATED AND RETURNED UNUSED — 51 at the 0168 boundary, 52 by
// 0169, whose whole effect is client-side. A returned number is not reused.
//
// 53 IS 1001: the CASTING SECTION gains a pending book cast per actor and the
// attached actor effects, the area record carries its phase, direction, source,
// magnitude and its accepted cell list, and the ENTITY RECORD grows a THIRTY-byte
// tail at +230 — Protection [5]int32 filling +230 through +249, then TokenSize
// and SeeInvisible at +250 and +251, Reaction at +252 and Spirit at +256 — which
// takes the record from 230 bytes to 260. A VERSION-50 BUFFER IS THIS
// ONE'S SHARP CASE, on the same shape 46 and 48 have: the header, the planes and
// every offset in front of the tail are identical, so the FIRST entity record
// decodes intact and the second and every one after it are read starting inside
// a neighbour. Nothing but the version byte separates a correct decode from that.
//
// WHAT 54 DOES NOT CARRY is the `Ghost` template Control Spirit raises from. It
// is install-derived input on the world, not state, so a decode keeps the
// receiving world's own — which is why a resume raises and a decode into a fresh
// world refuses the raise at admission (docs/DIVERGENCES.md, DIV-041).
//
// VERSION 54 adds SuppressCorpseLoot at entity offset +260. It is the stable
// template property that decides whether death creates item loot, so omitting it
// would let a save or replay change the world's sack state at the next death.
//
// VERSION 55 appends delayed kill attribution at +261: source EntityID, a
// presence byte and the signed spell-id byte. It is history consumed by a later
// death, so omitting it can credit a different skill after resume.
//
// VERSION 56 IS 1025, and it changes the form in two places. The ENTITY RECORD
// grows from 267 bytes to 275: Load at +267 and Capacity at +271, both int32,
// the actor's carried load and his carry capacity. And a NEW ITEM-WEIGHT
// SECTION sits between the spell table and the casting section: a uint16 count,
// then that many six-byte records of code and per-unit weight, which is the
// table every load is derived against.
//
// A VERSION-55 BUFFER IS THIS ONE'S SHARP CASE ON BOTH HALVES. On the record
// side it is 46's, 48's and 53's shape again: the header, the planes and every
// offset in front of +267 are identical, so the FIRST entity record decodes
// intact and the second and every one after it start eight bytes inside a
// neighbour. On the section side the spell table and everything in front of it
// are identical, so a decoder that skipped the version byte would read the
// item-weight count out of the casting section's own first two bytes. Nothing
// but the version byte separates a correct decode from either.
//
// VERSION 57 IS 1029, and it changes the form in two places for one story. The
// ENTITY RECORD grows from 275 bytes to 277: MapUnitID at +275, uint16, the
// authored map id a placement was made under. And the SCRIPT SECTION's check
// record grows from 73 bytes to 76: the item code at +73 as a uint16 and its
// presence byte at +75, the reference check opcode 17 reads. Both are one
// story's state and they move the version once between them.
//
// A VERSION-56 BUFFER IS THIS ONE'S SHARP CASE ON BOTH HALVES, on 46's, 48's,
// 53's and 56's own shape. On the record side every offset in front of +275 is
// identical, so the first entity record decodes intact and the second and every
// one after it start two bytes inside a neighbour. On the script side the
// section's own length test catches a short check array, but a version-56
// buffer whose check count is 0 differs from a version-57 one in no byte after
// the header. Nothing but the version byte separates a correct decode from
// either.
//
// VERSION 58 IS 1033, and it changes the form in ONE place. A NEW STRUCTURE
// SECTION sits between the script-state section and the script section: a
// uint32 count, then that many six-byte records of id and Field42, one per
// placed type-4 record on the map the world was built from. The entity record
// is UNCHANGED at 277 bytes, which is what makes this version unlike 56 and
// 57 — nothing in front of the new section moves.
//
// A VERSION-57 BUFFER IS THIS ONE'S SHARP CASE, and the shape is the section
// half of 56's and 57's without the record half. Every offset from the header
// through the script-state section is identical, so a decoder that skipped the
// version byte would read the structure count out of the SCRIPT SECTION's own
// first four bytes and then read script records at an offset four bytes and
// six per phantom structure too far. The count is bounded against the
// remaining buffer, so a wild count is refused rather than allocated; a small
// one is not, and a version-57 buffer whose script section opens with a small
// count decodes into structures that were never written. Nothing but the
// version byte separates a correct decode from that.
//
// VERSION 59 IS 1039, and it appends the five weapon damage-kind resistance
// bytes to every entity record at +277 through +281. The record grows from
// 277 bytes to 282; no offset in front of the new tail moves. Blade, Axe,
// Bludgeon, Pike and Shooting are stored in that order and carried whole:
// byte width is the original's modulo-256 fold, so every byte is legal.
//
// A VERSION-58 BUFFER IS THIS ONE'S SHARP CASE. Its header, planes, sections
// and every byte inside the first record's first 277 bytes are identical, so
// the second record and every one after it would be read five bytes into its
// neighbour. Nothing but the version byte separates a correct decode from
// that. A form older than oldestReadableVersion is refused.
//
// VERSION 60 IS 1037, and it appends Withdraw and Wimpy as signed int32 values
// at +282 and +286. The record grows from 282 bytes to 290 and no prior offset
// moves. A version-59 form never carried either threshold and is no longer
// read.
//
// VERSION 61 IS 1040. It appends canonical complete item-instance state for
// sacks, carry and equipment, followed by every entity's WeaponSpellSource
// tag and item-derived state. Secondary damage is one byte-width base, spread
// and selector triple followed by zero padding; the decoder rejects a selector
// outside 0..4 and any nonzero padding, so the wire form cannot admit a second
// simultaneous school. The older code-only fields remain validated projections
// for old readers; they never overwrite the canonical records.
//
// VERSION 62 IS 1045. It appends the Humanoid classification to every entity,
// Complication to every spell-table record, and the phase, retry progress,
// completion and retained-order bits to every pending book-cast record. All
// three change later deterministic action timing. The entity and spell values
// append to their records; the book lifecycle appends to its tagged record, so
// no earlier field moves.
//
// VERSION 63 IS 1047. It appends DesiredFacing and TurnRemaining to every
// entity. Together with the established Facing byte they preserve a physical
// turn across save/load and enter the deterministic digest. The inactive form
// repeats current Facing as DesiredFacing; an active desired byte is one of the
// eight direction quanta and the remainder is the actor ticks still owed.
//
// VERSION 64 IS 1047's pass-3 correction. It appends TurnTotal to every
// entity. The value is the request-time duration of an active turn and remains
// fixed when a later equipment recompute changes RotationSpeed. The inactive
// form writes zero. Version 63 migration derives the total that reproduces its
// saved renderer input while preserving TurnRemaining unchanged.
//
// VERSION 65 IS 1052. It widens each structure record from 6 to 22 bytes with
// maximum health and its authored target footprint. Current health remains at
// +4, so migration preserves damage already taken and repairs only immutable
// placement data from the mission being resumed.
//
// VERSION 66 IS 1063. Its byte widths do not move. It changes the meaning of a
// cloud area record's Remaining word from this build's former `counter + 1`
// representation to ROM1's raw `effect+0x4c` counter. A zero counter remains
// zero.
// Version 67 tags the existing attack-presence byte: 0 absent, 1 unit,
// 2 structure. Record widths do not change; all older attacks are units.
// Version 68 appends permanent Potion gains and derived cap headroom. Older
// saves default both arrays to zero; their mission derive seeds headroom.
// Version 69 appends the source Human movement context: presence byte, signed
// speed word, then native Speed, Load and Capacity int32. Older forms default
// to absent; their native overload policy is unchanged.
// Version 70 retains those entity widths and adds original-dead provenance
// and terminal retention: 173-byte records and a four-byte span before relation.
// Version 71 appends a 113-byte per-actor spellbook: explicit legacy/absent/
// present state and 28 (cached range, raw Defensive, mana word) slots. Earlier
// forms append zero: their table-backed book semantics remain unchanged.
// Version 72 appends the saved second physical base/spread as two bytes.
// Form-71 books (113 bytes) and original-dead records (173 bytes) do not change.
// Version 73 appends CurrentProfileBasis at entity+456. Zero means the native
// sheet used by every older save; 1 is an imported current sheet, 2 its explicit
// retirement by a native rebuild. No source process data is persisted.
// Form 74 adds sparse explicit instance weights after original-dead provenance.
// Form 75 adds current actor/container bookkeeping after instance weights.
// Form77 leaves all form76 fields and variable tails in place, then appends
// source-clock presence and the independent FullTick (sessionclockbinary.go).
// Form79 retains a saved Building roster, source metadata, and explicit cell
// links after the complete Group footer. Absent means legacy spatial policy.
// Form80 retains exact Player containers and monotonic Group identities after
// the complete form79 payload. Old saves keep absent container provenance.
// Form81 appends sparse last-rated-stride provenance after complete form80.
// Historical forms retain absent provenance, including during active transit.
// Form82 owns imported physical motion and independent typed actor cell slots.
// Form83 appends current original cell planes. Form84 appends the explicit
// saved-object registry and sparse live item/Sack identities.
//
// Form85 appends the raw session spans, the cell-record residue (DIV-932),
// the top-level SpellEffect graph (DIV-938/DIV-939), the Projectiles store
// (DIV-944) and the Diary content (DIV-956) — five fields world.go's own
// struct comments named carried-not-wire-form, each carried ACROSS a decode
// from the receiver's own prior value rather than read from the byte form
// before this story, so a decode into a fresh receiver lost all five
// silently. carriedresumebinary.go carries the complete wire layout. A zero
// span means every field is absent/empty. Form86 appends sparse action-end deadlines for idle
// regeneration. Form87 adds the sparse native Group Roam
// counter footer. Form88 adds persistent scenery scorch cells after the
// Group counter footer. Form90 appends exact Player formations and current
// Group owner identities. Form92 keeps the form91 layout, but area order is
// causal and Cells owns only the current painted layer. Form93 retains native remembered-attacker cells
// and candidate-build ages. Form94 retains the entity allocation floor after
// final actor removal. Form95 preserves prepared deliveries and admission
// payment across native saves.
const formatVersion = 95

// The form's two fixed sizes. A header, then the grid, then one record per
// entity, then one route per entity in the same order:
//
//	off              width              field
//	0                1                  format version
//	1                8                  tick, uint64
//	9                8                  RNG state, uint64
//	17               4 + 4              bounds Width, Height, int32
//	25               4                  entity count, uint32
//	29               1                  routing mode
//	30               4                  plane cell count, uint32 - ALL THREE
//	34               cells              block plane, row-major, one byte each
//	34+cells         cells              cost plane, same order and length
//	34+2*cells       cells              height plane, same order and length
//	34+3*cells+294*i 4+4+4+4+4+4+1+1    ID uint32; X, Y, TargetX, TargetY, Class
//	                 +4+4+1+4+2+2+1     int32; HasTarget 0/1; stall count;
//	                 +4+1+1+4           HP, MaxHP int32; movement domain; Speed
//	                 +4+4+4+4+4+4+4+1   int32; Transit, TransitTotal uint16;
//	                 +4                 GroupSpeed uint8; AttackTarget uint32;
//	                                    target tag 0 absent/1 unit/2 structure; attack phase;
//	                                    AttackCountdown, AttackCharge,
//	                                    AttackRelax, ToHit, Defence, Absorption,
//	                                    DamageBase, DamageSpread int32;
//	                 +1                 AlwaysHits 0/1; Group uint32;
//	                 +1+2+4             Owner uint32; Facing uint8; decay stage;
//	                                    Dwell uint16; DyingTime int32;
//	                 +1                 ScanRange uint8
//	                 +1+4+4+4+4+1       ActorState uint8; PatrolHeadX,
//	                                    PatrolHeadY, PatrolTailX, PatrolTailY
//	                                    int32; PatrolLeg uint8
//	                 +1                 Reach uint8
//	                 +4+4               PostX, PostY int32
//	                 +4+4+4+4+1+1       Mana, MaxMana int32;
//	                                    HealthRegenPeriod, ManaRegenPeriod
//	                                    int32; HealthHundredths,
//	                                    ManaHundredths uint8
//	                 +4                 CommandGroup uint32,
//	                                    carried whole with no value refused
//	                 +4*6+4+4+1+1       SkillXP [6]int32; Mind, XPValue
//	                                    int32; XPSlot uint8; GainsXP 0/1
//
// , carried whole
//
//	                   except the two refusals experienceFault
//	                   names
//	+4                 KnownSpells uint32 (0127 FR-4b), a
//	                   bitmask subscripted by spell id,
//	                   carried whole with no value refused
//	+4*6               Skill [6]int32,
//	                   one level per slot in slot order,
//	                   carried whole with no value refused
//	+2+4               WeaponSpell uint16, WeaponSpellLevel
//	                   int32, the
//	                   spell a weapon-borne cast releases
//	                   and the level it releases it at,
//	                   carried whole with no value refused —
//	                   zero id means none, on the spell
//	                   collection's own reserved row 0
//	+2+1+2+1           AutoSpell uint16, CastWait uint8,
//	                   SpellFX uint16, SpellFXSpell uint8
//
// , the spell cast
//
//	                                    unbidden, the ticks before the next
//	                                    unbidden attempt, and the ticks and
//	                                    spell of the mark a landed cast left
//	                                    — carried whole with no value refused
//	                 +1+4+1             OffMap 0/1; EscortTarget uint32;
//	                                    HasEscortTarget 0/1
//	                 +1                 EscortRange uint8
//	                 +4*5               Protection [5]int32
//	                 +1+1+4+4           TokenSize, SeeInvisible uint8;
//	                                    Reaction, Spirit int32
//	                 +1                 SuppressCorpseLoot 0/1
//	                 +4+1+1             KillCreditSource uint32,
//	                                    HasKillCredit 0/1,
//	                                    KillCreditSpell int8
//	                 +4+4               Load, Capacity int32
//	                 +2                 MapUnitID uint16, the
//	                                    authored map id the placement was made
//	                                    under, carried whole with no value
//	                                    refused - zero says the entity carries
//	                                    none
//	                 +1*5               Resistance [5]uint8, Blade, Axe,
//	                                    Bludgeon, Pike, Shooting order
//	                 +4+4               Withdraw, Wimpy int32, absolute-health
//	                                    AI thresholds
//	                 +1                 Humanoid 0/1, the action-cadence class
//	                 +1+1+1             DesiredFacing, TurnRemaining,
//	                                    TurnTotal uint8
//	after the last   4                  route cell count, uint32
//	record, per      8 each             x, y int32, in walking order
//	entity in order                     — a count of 0 is a unit holding no route
//	after the routes 4                  group record count, uint32
//	                 4+4+1+1+4+4 each   owner, group uint32; base uint8; order
//	                                    uint8; commandedX, commandedY int32 —
//	                                    ascending (owner, group), one per owned
//	                                    (owner, group) pair the entity records
//	                                    name
//	after the groups 4                  sack count, uint32
//	                 4+4+4+4 each,      X, Y int32; Gold uint32; item count
//	                 +2*items           uint32; that many item codes uint16 —
//	                                    ascending (Y, X), the merged, sorted
//	                                    list normaliseSacks produces
//	after the sacks  4 + 2*codes each,  carried[i]'s code count, uint32; that
//	                                    many item codes uint16 — one record
//	                                    per entity, entity order, no section
//	                                    count of its own
//	after the carry  2*EquipSlots each  equipment[i], twelve little-endian
//	                                    uint16 codes in slot order, one
//	                                    record per entity, entity order, no
//	                                    count of its own — every slot
//	                                    written whether empty or not, and
//	                                    zero left unrefused
//	after the        4*relationSlots    the purse, uint32 per roster slot,
//	equipment                           slot order, fixed width, no length
//	                                    of its own
//	after the purse  2 + 17*count       the spell table: spell
//	                                    count uint16, then that many
//	                                    records — id uint16; mana cost,
//	                                    damageMin, damageMax int32; school,
//	                                    maxRange, flags uint8, in the
//	                                    caller's own order
//	the last 2500                       the relation, row-major at stride 50
//
// THE GROUP SECTION sits between the routes and the script section — a
// COUNTED block of its own rather than a tail on the entity record or a block
// closing the form, which is what a bump has not been since version 16's
// relation. It fits there because the route reader reports what it consumed
// and the script section is required to consume what is left of the buffer
// exactly, and neither contract cares where its own start is measured from.
// Version 20 GROWS THIS SECTION IN PLACE — the record widens from 9 bytes to
// 18, the count and the section's own position are untouched — rather than
// adding a section of its own, on the same argument: the record's own two new
// fields are exactly what a decision now reads (order) and what a command
// will write into (the cell), and neither is a fact a reader may invent.
//
// THE SACK SECTION sits right after the group section and before the script
// section — a second COUNTED block rather than a third tail or a second
// block closing the form, on the group section's own argument: the group
// reader reports what it consumed, so the sack reader's own start is found
// by measurement, and the script section is still required to consume
// exactly what is left after both. It is written whole for every world,
// including one that authors no sack: an empty section is its count alone,
// four zero bytes, which is what makes a world built naming no sack and one
// built over an empty slice the same bytes and not merely the same
// behaviour.
//
// THE CARRY SECTION sits right after the sack section and before the
// EQUIPMENT SECTION — a third block in the middle rather than a fourth tail
// or a third block closing the form, on the sack section's own argument: the
// sack reader reports what it consumed, so the carry reader's own start is
// found by measurement. Unlike the group and sack sections it carries no
// OVERALL count of its own — one record per entity, in entity order, the
// route section's own rule, so there is nothing here free to disagree with
// the entity count the header already declares.
//
// THE EQUIPMENT SECTION follows it immediately, a fourth block and the first
// one this story adds: EquipSlots little-endian uint16 codes per entity, in
// slot order, one record per entity in entity order — the carry section's
// own placement rule, but the PURSE SECTION's own shape rather than the
// carry section's: no count of its own, per record or overall, because every
// one of the twelve slots is written whether it is empty or not, on the
// passability grid's own reason for writing every cell whether it is open or
// not. It fits between the carry section and the purse section because
// neither neighbour's own contract cares what sits between them — the
// carry reader reports what it consumed, and the purse section is a fixed
// compile-time width found from the far end alongside the relation, not from
// where it starts.
//
// THE PURSE SECTION follows it, a fifth block: relationSlots little-endian
// uint32, fixed at compile time exactly as the relation is, so it too
// carries no length of its own. All three of the carry, equipment and
// purse sections are written WHOLE for every world, including one that
// authors no stock, equips nothing and credits no gold: an entity carrying
// nothing writes a code count of 0, an entity wearing nothing writes twelve
// zeroed codes, and the purse section is relationSlots zeroed dwords —
// which is what makes a world built naming none of the three and one built
// over their absence materialising the same bytes and not merely the same
// behaviour.
//
// THE RELATION CLOSES THE FORM, after the script section, and it is the one
// block whose position is given from the end rather than from the start. Its
// length is a compile-time constant of this build rather than a function of
// either count the header declares, so it is the one section that can sit there
// without carrying a length of its own; and sitting there is what leaves every
// offset above it exactly where version 14 put it. It is written WHOLE for every
// world, including every world that will never author one, which is the rule the
// three planes are written under and what makes a world built naming no relation
// and one built over the matrix its absence materialises the same bytes rather
// than merely the same behaviour.
//
// Everything is little-endian. Positions and bounds share one width and one
// signedness so that the clamp compares like with like, and 32 bits hold any
// cell a shifted uint32 map position can name. The tick is 64-bit: it is an
// index, and an index does not wrap. The class sits at record offset +20, as
// wide and as signed as its source field's sign extension needs; the presence
// byte at +24 and the stall count at +25. Target presence is a byte of its own
// because a target may name any cell, so no coordinate value is free to stand
// for "none".
//
// THE HEALTH PAIR SITS AT THE RECORD'S TAIL, HP at +26 and MaxHP at +30, both
// signed and little-endian like every coordinate above them. At the tail rather
// than before the presence byte because inserting them anywhere earlier would
// move two offsets to no purpose. Both are carried WHOLE and neither is range
// checked: every int32 pair is a state the constructor can build, so refusing
// one here would make a world this package can produce a world it cannot read
// back.
//
// THE MOVEMENT DOMAIN sits at +34, the RATE AND THE CROSSING at +35, +39 and
// +41, and THE GROUP RATE TERM is the new tail: GroupSpeed, one byte, at +43.
// Each went to the tail as it arrived, for the reason the health pair did:
// putting one anywhere earlier would move every offset after it and buy nothing.
//
// The term is ONE BYTE and not four, and that is the field's own width rather
// than a saving: the value is a byte where it comes from, its own minimum loop
// stores a byte, and a wider field here would be able to hold values the rule
// that produces it cannot.
//
// THE DECAY BLOCK IS THE NEW TAIL, 7 bytes from +92: the stage at +92, the dwell
// at +93 as a word, the dying time at +95 as a signed dword. It went to the tail
// as every block before it did and for the same reason.
//
// The three widths are the three values'. The stage has five meanings and one of
// them is never stored, so a byte holds it with room the decoder refuses rather
// than folds. The dwell is a tick count, and a word bounds it at 65535 ticks —
// longer than any mission this package advances — where a byte would bound it at
// 255 and silently reinterpret a class that names more. The dying time is the
// definition column's own int32, carried whole and range checked NOWHERE, on the
// rule the health pair and the seven combat numbers already take: every int32 is
// a state the constructor accepts, so refusing one here would make a world this
// package can produce a world it cannot read back. What IS refused is the
// PAIRING — a stage on a living unit, no stage on a body, a dwell at a stage
// that owes none — which the constructor normalises, so the two stay in the
// relation every residue field is already in.
//
// Speed is carried WHOLE and range checked nowhere, exactly as the health pair
// is: every int32 is a state the constructor accepts, so refusing one here would
// make a world this package can produce a world it cannot read back. The transit
// PAIR is checked, by the same function the constructor uses, because only some
// pairs are states a tick can produce — the asymmetry is the fields' and not the
// reader's. The group term is likewise carried whole — every byte is a term some
// group of speeds produces — with ONE shape refused: a nonzero one on a unit that
// is not alive, which is a felled member still linked to a group.
//
// The pair is uint16 and not one byte: the longest transit the law can produce
// is the sub-cell grid over the smallest rate the clamp permits, which is 256
// and does not fit in a byte. A pair of bytes with 256 spelled as 0 would make
// the longest crossing and no crossing at all the same two bytes.
//
// The three planes sit BEFORE the records because a decode has to find where the
// records begin, and their length comes from the bounds rather than from any
// count the data declares. The ROUTES sit after them for the mirror reason: they are
// the section whose length nothing else can give, so they go where the end of
// the buffer can measure them. Put before the records, the declared entity count
// would have to be trusted with nothing left to check it against.
//
// THE ATTACK BLOCK IS THE NEW TAIL, 39 bytes from +44: the order and its cycle
// at +44…+53 — victim id, presence byte, phase byte, count owed — then the two
// cadence numbers at +54 and +58, then the five a blow reads at +62…+81, then
// the always-hits byte at +82. It went to the tail as every block before it did,
// and for the same reason: putting it anywhere earlier would move every offset
// after it and buy nothing.
//
// The seven numbers are carried WHOLE and range checked nowhere, exactly as
// Speed and the health pair are. The presence and always-hits bytes are checked,
// as the target-presence byte already is, and for its reason rather than for a
// shape's: a byte read as truthy would map two byte forms onto one world. The
// phase byte is checked against the three this build defines, the count owed
// against zero from below, and the order itself against four shapes a tick
// cannot produce — an order on a unit that is not alive, one naming its own
// unit, one naming a unit the form does not hold, and any of the three cycle
// fields nonzero on a unit holding no order.
//
// The declared count is therefore load-bearing now and is still not trusted: the
// record span is multiplied out in int64 and required to fit, and the route
// section is then required to consume what the SCRIPT SECTION's own fixed head
// leaves. A wrong count fails one of those two.
//
// THE SCRIPT SECTION CLOSES THE FORM, after the routes, and its layout is
// scriptbinary.go's. It is a fixed-width volatile half — registers, latches,
// counters, outcome — then three counted arrays of compiled records, and the
// last of those is what now has to consume the buffer exactly.
//
// The two tails are INDEPENDENT and version 10 is where that shows: the script
// closes the whole form and the attack block closes each entity record, so
// version 9 moved nothing inside a record and version 10 moves nothing after the
// routes. Every offset in the header and the grid is where version 8 put it, and
// every offset in the record up to +43 with them.
//
// Version 11 appends the group word at +83, past the attack block, so every
// offset version 10 fixed is still where it put it and only the record's WIDTH
// moves. Version 12 appends THE OWNER SLOT at +87 on the same terms, and it is
// carried at the placed record's own 32-bit width — the same width, and for the
// same reason, as the group word beside it.
//
// Version 14 appends THE FACING at +91 on the same terms again, at ONE BYTE —
// the field's own width where it comes from, eight directions in 32-unit steps
// over 256, and not a saving. A wider field could hold values no direction names;
// this one cannot, which is why every byte of it is legal and none is refused. It
// is the first record tail since version 8's group term to be a byte rather than
// a word, and for that same reason: the source field is one. Version 13 widened
// no record at all — it tripled the PLANES, before the records — so the record's
// own offsets run unbroken from version 12 to here.
//
// Version 17 appends THE DECAY LADDER at +92, +93 and +95 — the stage as a byte,
// the dwell owed as a uint16 and the dying time as an int32 — on the same terms
// again, and it is the first tail since version 12's to be wider than a byte.
//
// Version 18 appends THE SIGHT RANGE at +99, BEHIND IT, at one byte and for the
// facing's own reason: the source field is a byte, its guard-leash reader is a
// byte compare, and every value it can hold is a range, so nothing here is
// refused either. What differs from the facing is only that this one is READ —
// by the group's own march and by the notice radius — which is why the version's
// trade is argued from a wrong answer rather than from a misparse.
//
// THE TWO TAILS WERE WRITTEN IN PARALLEL and compose because both are tails: 17
// took +92 through +98 and 18 took +99, so nothing in front of the decay stage
// moved for either and the record went 92 to 99 to 100 without a single
// documented offset being restated. The same is true of the SECOND ordering, and
// that is what makes the composition a property rather than a coincidence — had
// either lane put its field anywhere but the end, the other's offsets would have
// had to move.
//
// A VERSION-17 BUFFER IS THIS ONE'S SHARP CASE, as a version-13 buffer was
// version 14's, and for the identical reason: header, planes and every offset
// inside the first record are the same, one byte differs at the end of each
// record, so the SECOND record and every record after it would be read one byte
// early out of the middle of its neighbour while the first survived intact.
// Nothing but the version byte separates a correct decode from that. A
// version-16 buffer is eight bytes short per record rather than one and is
// refused by the same comparison.
//
// Version 21 appends THE ACTOR STATE, THE RING AND THE LEG at +100 through
// +117 — the state byte, the head and tail coordinate pairs and the leg
// byte, in that order, on the tail's own terms again: no offset in front of
// the sight range moves, and only the record's WIDTH does, from 100 to 118.
// The GROUP RECORD carried by the same version moves no offset at all: its
// order byte has sat at +9 of its own record since version 20, and this
// version widens only the SET decodeGroups accepts there.
//
// A VERSION-20 BUFFER IS THIS ONE'S SHARP CASE too, on the record side: its
// header, planes and every offset inside the first record's first 100 bytes
// are identical, so the SECOND record and every one after it would again be
// read starting 18 bytes into its neighbour. Nothing but the version byte
// separates a correct decode from that.
//
// Version 22 appends THE SACK SECTION, a second counted block sitting right
// after the group section and before the script section (0103) — no offset
// above it moves, on the group section's own grounds: neither block carries a
// fixed length, so a section arriving between two contracts that are each
// found by measurement moves nothing either side of it. A sack record is
// variable-width — an X, a Y, a purse, an item count and that many item
// codes — so there is no per-record width to name here the way
// groupRecordLen names one; decodeSacks reads each record's own count as
// decodeRoutes already reads each route's.
//
// A VERSION-21 BUFFER IS THIS ONE'S SHARP CASE, and it is the group
// section's own kind of sharp rather than the per-record kind above: header,
// planes, every entity record and the group section itself are this
// version's exactly, so only the SCRIPT SECTION AND THE RELATION move — a
// version-21 buffer is four bytes shorter than this reader requires (the
// empty sack count a world with no sacks would still write), and it fails
// the script section's own exact-consumption check rather than parsing into
// a world with sacks that are not there.
//
// Version 23 appends THE REACH at +118, on the tail's own terms again
// (0104): no offset in front of the patrol leg moves, and only the record's
// WIDTH does, from 118 to 119.
//
// A VERSION-22 BUFFER IS THIS ONE'S SHARP CASE too, on the record side: its
// header, planes and every offset inside the first record's first 118 bytes
// are identical, so the SECOND record and every one after it would again be
// read starting one byte into its neighbour. Nothing but the version byte
// separates a correct decode from that.
//
// Version 24 appends THE POST at +119, on the tail's own terms again
// (0106): no offset in front of it moves, and only the record's WIDTH does,
// from 119 to 127. It is carried WHOLE, exactly as the commanded cell and
// the patrol ring already are — no coordinate refused, folded or clamped —
// on their own ground: every coordinate pair is a state a setter (the
// constructor or a stance command) can leave behind, so refusing one here
// would make a world this package produces a world it will not read back.
//
// A VERSION-23 BUFFER IS THIS ONE'S SHARP CASE too, on the record side: its
// header, planes and every offset inside the first record's first 119 bytes
// are identical, so the SECOND record and every one after it would again be
// read starting eight bytes into its neighbour. Nothing but the version
// byte separates a correct decode from that.
//
// Version 25 appends THE REGENERATION BLOCK at +127, on the tail's own
// terms again (0109): no offset in front of it moves, and only the
// record's WIDTH does, from 127 to 145. Mana sits at +127, MaxMana at
// +131, HealthRegenPeriod at +135, ManaRegenPeriod at +139,
// HealthHundredths at +143 and ManaHundredths at +144. The four int32 are
// carried WHOLE with no value refused, on the health pair's own ground:
// every int32 is a state the constructor can build, so refusing one here
// would make a world this package can produce a world it cannot read
// back. THE TWO REMAINDERS ARE THE ONE PAIR HERE THAT IS CHECKED — each
// against 99, regenFault's whole content — because a value above 99 has
// no fraction of a point left to name; the constructor folds the same
// shape to 0 rather than refusing it, on reachFault's own split.
//
// A VERSION-24 BUFFER IS THIS ONE'S SHARP CASE too, on the record side:
// its header, planes and every offset inside the first record's first
// 127 bytes are identical, so the SECOND record and every one after it
// would again be read starting eighteen bytes into its neighbour.
// Nothing but the version byte separates a correct decode from that.
//
// Version 26 appends THE CARRY SECTION AND THE PURSE SECTION, a third and
// a fourth counted-or-fixed block sitting right after the sack section and
// before the script section (0112) — no offset above it moves, on the sack
// section's own grounds: none of the three blocks now between the routes
// and the script section carries a fixed length, so a pair of sections
// arriving between two contracts that are each found by measurement moves
// nothing either side of it. THE CARRY SECTION is one record per entity in
// entity order, so there is no per-record width to name here either, the
// way entityLen or groupRecordLen do; decodeCarried reads each entity's own
// count as decodeRoutes already reads each route's, and there is no OVERALL
// count in front of the records because the entity count the header
// already declares is that number. THE PURSE SECTION follows it at a fixed
// purseLen, relationSlots little-endian uint32, the relation's own kind of
// block: sized by a compile-time constant rather than by any count the data
// declares, so it carries no length of its own either.
//
// A VERSION-25 BUFFER IS THIS ONE'S SHARP CASE, and it is the sack
// section's own kind of sharp rather than the per-record kind above:
// header, planes, every entity record, the routes, the groups and the
// sacks are this version's exactly, so only the CARRY SECTION, THE PURSE,
// THE SCRIPT SECTION AND THE RELATION move — a version-25 buffer is short
// by however many bytes the carry and purse sections now occupy, and it
// fails either the carry section's own per-record bounds check or the
// script section's own exact-consumption check, rather than parsing into a
// world with items or gold that are not there.
//
// Version 32 appends THE COMMAND GROUP at +145, on the tail's own terms
// again (0117): no offset in front of it moves, and only the record's
// WIDTH does, from 145 to 149 — so the carry, purse, group, sack and
// script blocks past the records are all found from four bytes per entity
// further along, and nothing inside any of them changes shape. It is
// carried WHOLE with no value refused, on the owner slot's own ground
// rather than the health pair's: every uint32 is an id the allocator can
// hand out or the zero that means none, so refusing one here would make a
// world this package produces a world it cannot read back.
//
// A VERSION-31 BUFFER IS THIS ONE'S SHARP CASE too, on the record side:
// its header, planes and every offset inside the first record's first
// 145 bytes are identical, so the SECOND record and every one after it
// would again be read starting four bytes into its neighbour. Nothing but
// the version byte separates a correct decode from that.
//
// Version 35 appends AN ENTITY'S EXPERIENCE FROM USE at +149, on the
// tail's own terms again (0125): no offset in front of it moves, and only
// the record's WIDTH does, from 149 to 183 — so the routes, groups, sacks,
// carry, purse and script blocks past the records are all found from
// thirty-four bytes per entity further along, and nothing inside any of
// them changes shape. SkillXP's six int32 sit at +149 through +172, Mind at
// +173, XPValue at +177, XPSlot at +181 and GainsXP at +182. Mind and
// XPValue are carried WHOLE, on the owner slot's own ground: both are
// table columns an allocator or a map placement can hand this field, so
// refusing one here would make a world this package produces a world it
// cannot read back. SkillXP and XPSlot are the two shapes experienceFault
// refuses; GainsXP's own byte is the one shape the decoder's switch
// refuses directly, on the target-presence byte's own ground.
//
// A VERSION-32 BUFFER IS THIS ONE'S SHARP CASE too, on the record side:
// its header, planes and every offset inside the first record's first
// 149 bytes are identical, so the SECOND record and every one after it
// would again be read starting thirty-four bytes into its neighbour.
// Nothing but the version byte separates a correct decode from that.
//
// Version 39 appends AN ENTITY'S SIX SKILL LEVELS at +187, on the tail's
// own terms again (0135): no offset in front of it moves, and only the
// record's WIDTH does, from 187 to 211 — one int32 per slot, carried
// WHOLE with no value refused, on Mind's and XPValue's own ground: a
// definition column and a loadout bonus both reach it, so every int32
// this field can hold is a state some caller can hand the constructor.
//
// A VERSION-36 BUFFER IS THIS ONE'S SHARP CASE too, on the record side:
// its header, planes and every offset inside the first record's first
// 187 bytes are identical, so the SECOND record and every one after it
// would again be read starting twenty-four bytes into its neighbour.
// Nothing but the version byte separates a correct decode from that.
//
// Version 41 appends A WEAPON'S OWN SPELL at +211, on the tail's own terms
// again (0139): no offset in front of it moves, and only the record's
// WIDTH does, from 211 to 217 — WeaponSpell a uint16 at +211,
// WeaponSpellLevel an int32 at +213, both carried WHOLE with no value
// refused, on Mind's and XPValue's own ground: a weapon's spell id and
// level are numbers a fold (pkg/data) or a decode can hand this field, so
// refusing either here would make a world this package can produce a world
// it cannot read back.
//
// A VERSION-39 BUFFER IS THIS ONE'S SHARP CASE too, on the record side:
// its header, planes and every offset inside the first record's first
// 211 bytes are identical, so the SECOND record and every one after it
// would again be read starting six bytes into its neighbour. Nothing but
// the version byte separates a correct decode from that.
//
// Version 45 appends THE AUTOCAST PAIR AND THE SPELL EFFECT MARK at +217, on
// the tail's own terms again: no offset in front of them moves, and only the
// record's WIDTH does, from 217 to 222 — AutoSpell a uint16 at +217,
// CastWait a byte at +219, SpellFX a byte at +220 and SpellFXSpell a byte at
// +221. All four are carried WHOLE with no value refused, on WeaponSpell's
// own ground: an id naming no row of this world's table is a runtime answer
// (autoCast simply finds nothing to cast) and not a decode-time fault.
//
// A VERSION-41 BUFFER IS THIS ONE'S SHARP CASE too, on the record side:
// its header, planes and every offset inside the first record's first
// 217 bytes are identical, so the SECOND record and every one after it
// would again be read starting five bytes into its neighbour. Nothing but
// the version byte separates a correct decode from that.
const (
	headerLen    = 34
	entityLenV67 = 294
	entityLenV68 = entityLenV67 + 32
	entityLenV69 = entityLenV68 + 15
	entityLenV70 = entityLenV69
	entityLenV71 = entityLenV70 + spellbookRecordLen
	entityLenV72 = entityLenV71 + 2
	entityLenV75 = entityLenV72 + 1
	entityLen    = entityLenV75 + sourceBindingLen
	// routeCountLen is the width of one route's cell count, and cellLen the
	// width of one cell in it.
	routeCountLen = 4
	cellLen       = 8
	// groupCountLen is the width of the group section's own record count, and
	// groupRecordLen the width of one record in it: owner uint32, group
	// uint32, base uint8, order uint8, commandedX int32, commandedY int32 —
	// 18 bytes since version 20 (0096), grown in place from version 19's 9.
	groupCountLen  = 4
	groupRecordLen = 18
	// sackCountLen is the width of the sack section's own record count, and
	// sackHeaderLen the width of one sack's fixed head — X, Y, Gold, item
	// count — before its variable item list (0103). There is no fixed
	// per-record width: each sack's own item count says how many uint16
	// codes follow its head.
	sackCountLen  = 4
	sackHeaderLen = 16
	// carryCountLen is the width of one entity's carried-code count in the
	// CARRY SECTION — the route section's own routeCountLen, restated for a
	// different field: a uint32, then that many little-endian uint16 codes.
	// There is no OVERALL count beside it: the section carries one record per
	// entity, in entity order, and the entity count the header already declares
	// is the record count.
	carryCountLen = 4
	// legacyEquipRecordLen is the version-60 equipment width. Version 61 writes
	// twelve variable item records instead. SECTION: EquipSlots little-endian
	// uint16 codes, one per slot in slot order, with no count of its own —
	// the purse section's own rule and not the carry section's, restated for a
	// record sized by the entity count rather than by relationSlots: there is
	// no OVERALL count either, because the entity count the header already
	// declares is the record count.
	legacyEquipRecordLen = EquipSlots * 2
	// equipRecordLen remains the version-60 fixture width used by migration
	// tests; current equipment records are variable.
	equipRecordLen = legacyEquipRecordLen
	itemHeadLen    = 11 // code u16, kind u8, price i32, effect count u32
	itemEffectLen  = 6  // kind u8, mode u8, operand u32
	stackCountLen  = 4
	// treasureRecordLen is the version-44 fixed record beside each entity:
	// TypeID, GoldChance, TreasureMin and TreasureMax as four int32 values.
	treasureRecordLen = 16
	// purseLen is the PURSE SECTION's own fixed width: relationSlots
	// little-endian uint32s, one per roster slot in slot order, with no
	// length of its own — the relation block's own rule, restated for a
	// section that does not close the form.
	purseLen = relationSlots * 4
	// spellCountLen is the width of the SPELL TABLE's own record count, and
	// spellRecordLen the width of one record in it: id uint16; mana cost,
	// damageMin, damageMax int32; school, maxRange, flags uint8 — 2 + 4 + 4 +
	// 4 + 1 + 1 + 1 = 17 bytes.
	//
	// VERSION 49 APPENDS THE AREA DURATION as an int32, taking the record to 21
	// bytes. Every offset before +17 is unmoved.
	spellCountLen  = 2
	spellRecordLen = 36
	// The item-weight section's two widths (version 56): a uint16 count, then a
	// code and a signed weight per record. The count is two bytes on the spell
	// table's own reason -- the table is bounded by how many distinct item
	// codes a world can name, which is at most the 65 536 the code word itself
	// allows, and in practice the few dozen a mission's containers and script
	// literals between them mention.
	itemWeightCountLen  = 2
	itemWeightRecordLen = 6
	// The flags byte's four defined bits: bit 0 is TargetsUnit and bit 1 is
	// Damaging; bit 2 is Restorative and bit 3 is Area. Every other bit is
	// undefined and refused by decodeSpells rather than folded — a byte
	// normalised here would map two byte forms onto one world, on the
	// target-presence byte's own ground.
	//
	// BIT 2 REPAIRS A ROW THAT DID NOT ROUND-TRIP. Restorative was added to
	// SpellRule by 0154 and reached no bit of this byte, so a world holding
	// a heal row decoded with the flag clear and its heal arm gone. 0165's
	// own point-cast arm (scriptcast.go) forks on that flag, so the hole is
	// load-bearing here rather than incidental, and this version is the one
	// that was already moving the record.
	spellFlagTargetsUnit uint8 = 1 << 0
	spellFlagDamaging    uint8 = 1 << 1
	spellFlagRestorative uint8 = 1 << 2
	spellFlagArea        uint8 = 1 << 3
	spellFlagDefensive   uint8 = 1 << 4
	spellFlagRowSpecific uint8 = 1 << 5
	spellFlagAreaHitsLo  uint8 = 1 << 6
	spellFlagAreaHitsHi  uint8 = 1 << 7
)

// raysFlag: bit 7 on a point Prismatic Spray record puts the cap in the radius byte.
func raysFlag(flags uint8, id uint16) bool {
	return id == prismaticSpellID && flags&(spellFlagArea|spellFlagAreaHitsLo|spellFlagAreaHitsHi) == spellFlagAreaHitsHi
}

func itemByteLen(item ItemInstance) int {
	return itemHeadLen + itemEffectLen*len(item.Effects)
}

func appendItemBytes(out []byte, item ItemInstance) []byte {
	out = binary.LittleEndian.AppendUint16(out, item.Code)
	out = append(out, item.Kind)
	out = binary.LittleEndian.AppendUint32(out, uint32(item.Price))
	out = binary.LittleEndian.AppendUint32(out, uint32(len(item.Effects)))
	for _, effect := range item.Effects {
		out = append(out, effect.Kind, effect.Mode)
		out = binary.LittleEndian.AppendUint32(out, effect.Operand)
	}
	return out
}

func decodeItemBytes(data []byte, what string) (ItemInstance, int, error) {
	if len(data) < itemHeadLen {
		return ItemInstance{}, 0, fmt.Errorf("sim: byte form truncated: %s head needs %d byte(s), %d left", what, itemHeadLen, len(data))
	}
	item := ItemInstance{
		Code:  binary.LittleEndian.Uint16(data[0:2]),
		Kind:  data[2],
		Price: int32(binary.LittleEndian.Uint32(data[3:7])),
	}
	count := binary.LittleEndian.Uint32(data[7:11])
	if need := int64(count) * itemEffectLen; need > int64(len(data)-itemHeadLen) {
		return ItemInstance{}, 0, fmt.Errorf("sim: byte form truncated: %s declares %d effect(s), which need %d byte(s), %d left", what, count, need, len(data)-itemHeadLen)
	}
	item.Effects = make([]ItemEffect, count)
	off := itemHeadLen
	for i := range item.Effects {
		item.Effects[i] = ItemEffect{Kind: data[off], Mode: data[off+1], Operand: binary.LittleEndian.Uint32(data[off+2 : off+6])}
		off += itemEffectLen
	}
	if item.Code == 0 && !item.Empty() {
		return ItemInstance{}, 0, fmt.Errorf("sim: %s has empty code with item metadata", what)
	}
	return item, off, nil
}

// encode writes w's whole canonical state as the byte form above. It cannot
// fail, and it is the single traversal behind both MarshalBinary and Hash — so
// the digest covers exactly what the byte form carries, by construction rather
// than by a second traversal that could drift from the first.
//
// It reads w.routes[i] for every entity, so it rests on the two slices being
// parallel — which every world this package can build or decode satisfies, and
// which the pin on World's field set is what keeps visible.
func (w *World) encode() []byte {
	return w.encodeInto(nil)
}

func (w *World) binaryBodyLen() int {
	n := headerLen + len(w.grid) + len(w.cost) + len(w.height) + relationLen +
		entityLen*len(w.entities)
	for _, r := range w.routes {
		n += routeCountLen + cellLen*len(r)
	}
	n += groupCountLen + groupRecordLen*len(w.groups)
	n += sackCountLen
	for _, s := range w.sacks {
		n += sackHeaderLen + 2*len(s.Items)
	}
	// THE CARRY SECTION IS SIZED IN UNITS AND NOT IN ELEMENTS (0138 D-5):
	// the section writes a container's FLAT EXPANSION, so what is reserved
	// here is Σ Count and not len — the one arithmetic that has to change
	// for this story, and the one that would silently under-reserve the
	// buffer if it did not, because before 0138 the two numbers were equal.
	for _, stacks := range w.carried {
		n += carryCountLen + 2*int(containerUnits(stacks))
	}
	n += equipRecordLen * len(w.equipment)
	n += treasureRecordLen * len(w.entities)
	n += purseLen
	n += spellCountLen + spellRecordLen*len(w.spells)
	n += itemWeightCountLen + itemWeightRecordLen*len(w.itemWeights)
	n += w.castingSectionLen()
	n += w.scrollSectionLen()
	n += w.scriptStateSectionLen()
	n += w.structureSectionLen()
	n += w.itemStateSectionLen()
	n += w.scriptSectionLen()
	n += w.originalDeadSectionLen()
	n += w.actorLoadSectionLen()
	n += w.instanceWeightSectionLen()
	return n
}

func (w *World) encodeInto(b []byte) []byte {
	n := w.binaryBodyLen()
	if cap(b) < n {
		b = make([]byte, n)
	} else {
		b = b[:n]
		clear(b)
	}

	b[0] = formatVersion
	binary.LittleEndian.PutUint64(b[1:9], w.tick)
	binary.LittleEndian.PutUint64(b[9:17], w.rng.state)
	binary.LittleEndian.PutUint32(b[17:21], uint32(w.bounds.Width))
	binary.LittleEndian.PutUint32(b[21:25], uint32(w.bounds.Height))
	binary.LittleEndian.PutUint32(b[25:29], uint32(len(w.entities)))
	b[29] = byte(w.mode)
	binary.LittleEndian.PutUint32(b[30:34], uint32(len(w.grid)))
	// All three planes are always materialised, so this writes W*H cells apiece
	// for every world and there is no absent case to encode differently. That is
	// what makes a world built with no plane and one built over the plane its
	// absence materialises the same bytes rather than merely the same behaviour.
	copy(b[headerLen:], w.grid)
	copy(b[headerLen+len(w.grid):], w.cost)
	copy(b[headerLen+len(w.grid)+len(w.cost):], w.height)

	records := headerLen + len(w.grid) + len(w.cost) + len(w.height)
	for i, e := range w.entities {
		o := records + entityLen*i
		binary.LittleEndian.PutUint32(b[o:o+4], uint32(e.ID))
		binary.LittleEndian.PutUint32(b[o+4:o+8], uint32(e.X))
		binary.LittleEndian.PutUint32(b[o+8:o+12], uint32(e.Y))
		binary.LittleEndian.PutUint32(b[o+12:o+16], uint32(e.TargetX))
		binary.LittleEndian.PutUint32(b[o+16:o+20], uint32(e.TargetY))
		binary.LittleEndian.PutUint32(b[o+20:o+24], uint32(e.Class))
		if e.HasTarget {
			b[o+24] = 1
		}
		if e.HasPendingAttackTarget {
			binary.LittleEndian.PutUint32(b[o+12:o+16], uint32(e.PendingAttackTarget))
			b[o+24] = 2 + byte(e.PendingAttackTargetKind)
		}
		b[o+25] = e.Stall
		binary.LittleEndian.PutUint32(b[o+26:o+30], uint32(e.HP))
		binary.LittleEndian.PutUint32(b[o+30:o+34], uint32(e.MaxHP))
		b[o+34] = byte(e.Domain)
		binary.LittleEndian.PutUint32(b[o+35:o+39], uint32(e.Speed))
		binary.LittleEndian.PutUint16(b[o+39:o+41], e.Transit)
		binary.LittleEndian.PutUint16(b[o+41:o+43], e.TransitTotal)
		b[o+43] = e.GroupSpeed
		binary.LittleEndian.PutUint32(b[o+44:o+48], uint32(e.AttackTarget))
		if e.HasAttackTarget {
			b[o+48] = byte(e.AttackTargetKind) + 1
			if e.AcquirePursuit {
				b[o+48] = 3
			}
			if e.PursuitIdle {
				b[o+48] = 4
			}
		}
		b[o+49] = byte(e.AttackPhase)
		binary.LittleEndian.PutUint32(b[o+50:o+54], uint32(e.AttackCountdown))
		binary.LittleEndian.PutUint32(b[o+54:o+58], uint32(e.AttackCharge))
		binary.LittleEndian.PutUint32(b[o+58:o+62], uint32(e.AttackRelax))
		binary.LittleEndian.PutUint32(b[o+62:o+66], uint32(e.ToHit))
		binary.LittleEndian.PutUint32(b[o+66:o+70], uint32(e.Defence))
		binary.LittleEndian.PutUint32(b[o+70:o+74], uint32(e.Absorption))
		binary.LittleEndian.PutUint32(b[o+74:o+78], uint32(e.DamageBase))
		binary.LittleEndian.PutUint32(b[o+78:o+82], uint32(e.DamageSpread))
		if e.AlwaysHits {
			b[o+82] = 1
		}
		binary.LittleEndian.PutUint32(b[o+83:o+87], e.Group)
		binary.LittleEndian.PutUint32(b[o+87:o+91], e.Owner)
		b[o+91] = e.Facing
		b[o+92] = byte(e.Decay)
		binary.LittleEndian.PutUint16(b[o+93:o+95], e.Dwell)
		binary.LittleEndian.PutUint32(b[o+95:o+99], uint32(e.DyingTime))
		b[o+99] = e.ScanRange
		// THE ACTOR STATE, THE RING AND THE LEG are the new tail (version
		// 21, 0099): the state byte at +100, the head and the tail at
		// +101…+116, the leg at +117.
		b[o+100] = e.ActorState
		if e.PendingOrder.Kind == PendingPickupComplete {
			b[o+100] = actorStateGuard
		}
		if e.PendingOrder.Kind != PendingNone && e.Retreat.Known {
			b[o+100] = actorStateRetreat
		}
		binary.LittleEndian.PutUint32(b[o+101:o+105], uint32(e.PatrolHeadX))
		binary.LittleEndian.PutUint32(b[o+105:o+109], uint32(e.PatrolHeadY))
		binary.LittleEndian.PutUint32(b[o+109:o+113], uint32(e.PatrolTailX))
		binary.LittleEndian.PutUint32(b[o+113:o+117], uint32(e.PatrolTailY))
		b[o+117] = e.PatrolLeg
		b[o+118] = e.Reach
		// THE POST IS THE RECORD'S NEW TAIL (version 24, 0106): PostX at +119,
		// PostY at +123, carried WHOLE — no coordinate refused, folded or
		// clamped, on the commanded cell's and the patrol ring's own rule.
		binary.LittleEndian.PutUint32(b[o+119:o+123], uint32(e.PostX))
		binary.LittleEndian.PutUint32(b[o+123:o+127], uint32(e.PostY))
		// THE REGENERATION BLOCK IS THE RECORD'S NEW TAIL (version 25,
		// 0109): Mana at +127, MaxMana at +131, HealthRegenPeriod at
		// +135, ManaRegenPeriod at +139, HealthHundredths at +143,
		// ManaHundredths at +144. The four int32 are carried WHOLE, on
		// the health pair's own rule; the two remainders are refused by
		// the decoder rather than written checked — regenFault below.
		binary.LittleEndian.PutUint32(b[o+127:o+131], uint32(e.Mana))
		binary.LittleEndian.PutUint32(b[o+131:o+135], uint32(e.MaxMana))
		binary.LittleEndian.PutUint32(b[o+135:o+139], uint32(e.HealthRegenPeriod))
		binary.LittleEndian.PutUint32(b[o+139:o+143], uint32(e.ManaRegenPeriod))
		b[o+143] = e.HealthHundredths
		b[o+144] = e.ManaHundredths
		// THE COMMAND GROUP IS THE RECORD'S NEW TAIL (version 27): one word at
		// +145, carried WHOLE with no value refused — this task gives it no
		// writer and so no rule to check it against, on the owner slot's own
		// ground (0106's own doc above).
		binary.LittleEndian.PutUint32(b[o+145:o+149], e.CommandGroup)
		// AN ENTITY'S EXPERIENCE FROM USE IS THE RECORD'S NEW TAIL (version 35):
		// the six slot experiences at +149 through +172, Mind at +173, the
		// experience value at +177, the credited slot at +181 and the gains flag
		// at +182. The two int32 are carried WHOLE, on the owner slot's own rule
		// above; the slot and the six experiences are refused rather than written
		// checked — experienceFault below — and the flag is a plain 0/1 byte,
		// on the target-presence byte's own rule.
		for k, xp := range e.SkillXP {
			binary.LittleEndian.PutUint32(b[o+149+4*k:o+153+4*k], uint32(xp))
		}
		binary.LittleEndian.PutUint32(b[o+173:o+177], uint32(e.Mind))
		binary.LittleEndian.PutUint32(b[o+177:o+181], uint32(e.XPValue))
		b[o+181] = e.XPSlot
		if e.GainsXP {
			b[o+182] = 1
		}
		// AN ENTITY'S OWN SPELLBOOK IS THE RECORD'S NEW TAIL (version 36,
		// 0127 FR-4b): a bitmask subscripted by spell id, carried WHOLE
		// with no value refused — KnownSpells' own doc (world.go) says
		// why: a bit outside the shipped id space names a spell no table
		// this build loads can ever hold a row for, so the cast's own
		// linear lookup already answers "no such row" to it without this
		// record having to.
		binary.LittleEndian.PutUint32(b[o+183:o+187], e.KnownSpells)
		// AN ENTITY'S SIX SKILL LEVELS ARE THE RECORD'S NEW TAIL (version 39): one
		// int32 per slot at +187 through +210, carried WHOLE with no value refused
		// — Skill's own doc (world.go) says why: a definition column and a
		// loadout bonus both reach it, so every int32 this field can hold is a
		// state some caller can hand the constructor.
		for k, lvl := range e.Skill {
			binary.LittleEndian.PutUint32(b[o+187+4*k:o+191+4*k], uint32(lvl))
		}
		// A WEAPON'S OWN SPELL IS THE RECORD'S NEW TAIL (version 41): the id at
		// +211, a uint16, and the level at +213, an int32, both carried WHOLE with
		// no value refused — WeaponSpell's own doc (world.go) says why: an id
		// naming no row this world's table loaded is a runtime answer
		// (weaponSpell, spell.go), not a decode-time fault.
		binary.LittleEndian.PutUint16(b[o+211:o+213], e.WeaponSpell)
		binary.LittleEndian.PutUint32(b[o+213:o+217], uint32(e.WeaponSpellLevel))
		// THE AUTOCAST PAIR AND THE SPELL EFFECT MARK ARE THE RECORD'S NEW TAIL
		// (version 45): the autocast id at +217, a uint16, its wait at +219, and
		// the mark's remaining ticks and spell id at +220 and +221. All four are
		// carried WHOLE with no value refused, on WeaponSpell's own ground
		// immediately above: an id naming no row this world's table loaded is a
		// runtime answer (autoCast, spell.go) and not a decode-time fault.
		binary.LittleEndian.PutUint16(b[o+217:o+219], e.AutoSpell)
		b[o+219] = e.CastWait
		// THE MARK'S REMAINING TICKS ARE A WORD SINCE VERSION 50. It renders the
		// attached effect's own duration field, which is a word in the thing being
		// reconstructed and which instant 30 writes `(u16)p1` into; the campaign
		// authors 60000, which the byte this used to be would have carried as 48.
		// Everything after it moved up one.
		binary.LittleEndian.PutUint16(b[o+220:o+222], e.SpellFX)
		b[o+222] = e.SpellFXSpell
		// THE OFF-MAP BIT (version 48): one 0/1 byte, on the target-presence
		// byte's own rule — a byte read as merely truthy would map two byte
		// forms onto one world, so the decoder's own switch refuses everything but
		// 0 and 1.
		if e.OffMap {
			b[o+223] = 1
		}
		// THE ESCORT TRIPLE IS THE RECORD'S NEW TAIL (version 50): the target id
		// at +224, its presence bit at +228 and the range at +229.
		//
		// THE PRESENCE BIT IS ITS OWN BYTE and is refused outside 0 and 1,
		// on the off-map byte's rule directly above and for the reason every
		// reference in this package carries one: entity id zero is a real
		// entity, so no id value is free to mean "none". THE RANGE IS
		// CARRIED WHOLE with no value refused — the field is a byte, every
		// byte is a range, and the 0-to-3 coercion belongs to the arm that
		// writes it and not to the form that stores it.
		binary.LittleEndian.PutUint32(b[o+224:o+228], uint32(e.EscortTarget))
		if e.HasEscortTarget {
			b[o+228] = 1
		}
		b[o+229] = e.EscortRange
		for k, p := range e.Protection {
			binary.LittleEndian.PutUint32(b[o+230+4*k:o+234+4*k], uint32(p))
		}
		b[o+250] = e.TokenSize
		// SeeInvisible is the version-53 tail at +251. It is canonical actor
		// state because it changes deterministic target admission.
		b[o+251] = e.SeeInvisible
		// Reaction and Spirit complete the version-53 Control Spirit source
		// state. Mind remains at its established +173 offset.
		binary.LittleEndian.PutUint32(b[o+252:o+256], uint32(e.Reaction))
		binary.LittleEndian.PutUint32(b[o+256:o+260], uint32(e.Spirit))
		// SuppressCorpseLoot is the version-54 tail at +260. It is a 0/1
		// byte because the death pass reads it to decide canonical sack state.
		if e.SuppressCorpseLoot {
			b[o+260] = 1
		}
		// Delayed kill attribution is the version-55 tail. The source has a
		// presence bit because entity id zero is real; the signed spell byte is
		// stored by its bit pattern and carried whole.
		binary.LittleEndian.PutUint32(b[o+261:o+265], uint32(e.KillCreditSource))
		if e.HasKillCredit {
			b[o+265] = 1
		}
		b[o+266] = byte(e.KillCreditSpell)
		// VERSION 56's own eight bytes. Both are written WHOLE with no value
		// refused: a load is a sum over what an actor holds and a capacity is
		// `Body x 10 + 1` or the zero that says nothing derived one, and neither
		// has a value to fold onto.
		binary.LittleEndian.PutUint32(b[o+267:o+271], uint32(e.Load))
		binary.LittleEndian.PutUint32(b[o+271:o+275], uint32(e.Capacity))
		// VERSION 57's own two bytes. The authored map id is written whole with no
		// value refused: any word a map record carries is one this field may hold,
		// and zero says the entity carries none.
		binary.LittleEndian.PutUint16(b[o+275:o+277], e.MapUnitID)
		// VERSION 59's own five-byte tail (1039): the unsigned weapon-kind
		// resistances in Blade..Shooting order. Every byte is carried whole;
		// modulo narrowing happened at the data-to-simulation boundary.
		copy(b[o+277:o+282], e.Resistance[:])
		// VERSION 60's own eight-byte tail (1037): the signed absolute-health
		// thresholds, carried whole. Their zero value is inert for a living
		// actor and every int32 is a value the Units stream may author.
		binary.LittleEndian.PutUint32(b[o+282:o+286], uint32(e.Withdraw))
		binary.LittleEndian.PutUint32(b[o+286:o+290], uint32(e.Wimpy))
		// VERSION 62's entity tail: the class predicate read by the canonical
		// action-recovery formula. Refuse non-booleans on decode so one world has
		// one byte form.
		if e.Humanoid {
			b[o+290] = 1
		}
		// VERSION 63's turn pair and VERSION 64's duration tail. DesiredFacing
		// is repeated even for an inactive turn and both counts are zero, so the
		// form has one null shape rather than three bytes of residue.
		b[o+291] = e.DesiredFacing
		b[o+292] = e.TurnRemaining
		b[o+293] = e.TurnTotal
		for k := 0; k < 4; k++ {
			binary.LittleEndian.PutUint32(b[o+294+4*k:], uint32(e.PotionStats[k]))
			binary.LittleEndian.PutUint32(b[o+310+4*k:], uint32(e.PotionHeadroom[k]))
		}
		if e.HumanMovement.Present {
			b[o+326] = 1
		}
		binary.LittleEndian.PutUint16(b[o+327:], uint16(e.HumanMovement.RawSpeed))
		binary.LittleEndian.PutUint32(b[o+329:], uint32(e.HumanMovement.NativeSpeed))
		binary.LittleEndian.PutUint32(b[o+333:], uint32(e.HumanMovement.Load))
		binary.LittleEndian.PutUint32(b[o+337:], uint32(e.HumanMovement.Capacity))
		encodeSpellbook(b[o+341:o+entityLenV71], e.Book)
		b[o+454], b[o+455] = e.SecondBase, e.SecondSpread
		b[o+456] = byte(e.CurrentProfileBasis)
		_, _ = binary.Encode(b[o+entityLenV75:o+entityLen], binary.LittleEndian, e.SourceBinding)
	}

	// One route per entity, in the same order, each a count and that many cells.
	// A unit holding none writes the count 0 and nothing after it, so a world
	// whose routes are all empty and one whose route slice holds nothing but
	// nils are the same bytes — there is no absent case here either.
	o := records + entityLen*len(w.entities)
	for _, r := range w.routes {
		binary.LittleEndian.PutUint32(b[o:o+4], uint32(len(r)))
		o += routeCountLen
		for _, c := range r {
			binary.LittleEndian.PutUint32(b[o:o+4], uint32(c.x))
			binary.LittleEndian.PutUint32(b[o+4:o+8], uint32(c.y))
			o += cellLen
		}
	}
	// THE GROUP SECTION, between the routes and the script section: a count,
	// then that many records, ascending — the order freezeGroups already
	// produced, so this loop writes them exactly as it built them. The order
	// and the commanded cell are the record's new tail (version 20, 0096); the
	// owner, the group and the base keep the offsets version 19 gave them.
	binary.LittleEndian.PutUint32(b[o:o+4], uint32(len(w.groups)))
	o += groupCountLen
	for _, g := range w.groups {
		binary.LittleEndian.PutUint32(b[o:o+4], g.owner)
		binary.LittleEndian.PutUint32(b[o+4:o+8], g.group)
		b[o+8] = g.base
		b[o+9] = g.order
		binary.LittleEndian.PutUint32(b[o+10:o+14], uint32(g.commandedX))
		binary.LittleEndian.PutUint32(b[o+14:o+18], uint32(g.commandedY))
		o += groupRecordLen
	}
	// THE SACK SECTION, between the group section and the script section: a
	// count, then that many sacks, ascending — the order normaliseSacks
	// already produced, so this loop writes them exactly as it built them.
	// A sack with no item names an item count of 0 and writes nothing after
	// its head, on the route section's own rule: there is no absent case
	// here either, and a world naming no sack writes this count alone.
	binary.LittleEndian.PutUint32(b[o:o+4], uint32(len(w.sacks)))
	o += sackCountLen
	for _, s := range w.sacks {
		binary.LittleEndian.PutUint32(b[o:o+4], uint32(s.X))
		binary.LittleEndian.PutUint32(b[o+4:o+8], uint32(s.Y))
		binary.LittleEndian.PutUint32(b[o+8:o+12], s.Gold)
		binary.LittleEndian.PutUint32(b[o+12:o+16], uint32(len(s.Items)))
		o += sackHeaderLen
		for _, code := range s.Items {
			binary.LittleEndian.PutUint16(b[o:o+2], code)
			o += 2
		}
	}
	// THE CARRY SECTION, right after the sack section: one record per entity,
	// in entity order — carried's own shape, parallel to entities exactly as
	// routes already is — each a code count then that many codes, on the
	// route section's own rule: there is no section count of its own to
	// disagree with the entity count the header already declares, and an
	// entity carrying nothing writes a count of 0 and nothing after it, so
	// there is no absent case here either.
	//
	// IT WRITES THE FLAT EXPANSION, WHICH IS WHY 0138 MOVED NO OFFSET AND NO
	// VERSION. This section's bytes have always been "a code count, then that
	// many codes"; a stack is a GROUPING of those codes and not a field beside
	// them, so an element of count 3 writes its code three times and every
	// offset, every width and every existing payload's meaning stay exactly as
	// they were. Widening the record to code,count pairs was rejected: it would
	// be a version bump for state the form already carries.
	for _, stacks := range w.carried {
		codes := expandContainer(stacks)
		binary.LittleEndian.PutUint32(b[o:o+4], uint32(len(codes)))
		o += carryCountLen
		for _, code := range codes {
			binary.LittleEndian.PutUint16(b[o:o+2], code)
			o += 2
		}
	}
	// THE EQUIPMENT SECTION, right after the carry section: one fixed-width
	// record per entity, in entity order — carried's own shape, but the purse
	// section's own rule for the record itself: no count of any kind, because
	// every one of the twelve slots is written whether it is empty (the zero
	// code) or not, so there is nothing here for a count to disagree with.
	for _, eq := range w.equipment {
		for _, item := range eq {
			binary.LittleEndian.PutUint16(b[o:o+2], item.Code)
			o += 2
		}
	}
	// THE DEATH-GOLD SECTION is version 44's fixed record per entity. Versions
	// 42 and 43 were allocated to parallel work and are deliberately skipped.
	for _, e := range w.entities {
		binary.LittleEndian.PutUint32(b[o:o+4], uint32(e.TypeID))
		binary.LittleEndian.PutUint32(b[o+4:o+8], uint32(e.GoldChance))
		binary.LittleEndian.PutUint32(b[o+8:o+12], uint32(e.TreasureMin))
		binary.LittleEndian.PutUint32(b[o+12:o+16], uint32(e.TreasureMax))
		o += treasureRecordLen
	}
	// THE PURSE SECTION, right after the death-gold section: relationSlots
	// little-endian uint32s, one per roster slot in slot order, with no
	// length of its own — the relation block's own rule, restated for a
	// section that does not close the form: it is written WHOLE for every
	// world, including one that credits no gold to any slot, on the
	// relation's own reason.
	for i, p := range w.purses {
		binary.LittleEndian.PutUint32(b[o+4*i:o+4*i+4], p)
	}
	o += purseLen
	// THE SPELL TABLE, right after the purse section and before the script
	// section (version 36): a uint16 count, then that many 17-byte records,
	// written in the caller's own order — normaliseSpells copies but does not
	// sort, so this loop writes the table exactly as it was built.
	binary.LittleEndian.PutUint16(b[o:o+2], uint16(len(w.spells)))
	o += spellCountLen
	for _, sp := range w.spells {
		binary.LittleEndian.PutUint16(b[o:o+2], sp.ID)
		binary.LittleEndian.PutUint32(b[o+2:o+6], uint32(sp.ManaCost))
		binary.LittleEndian.PutUint32(b[o+6:o+10], uint32(sp.DamageMin))
		binary.LittleEndian.PutUint32(b[o+10:o+14], uint32(sp.DamageMax))
		b[o+14] = sp.School
		b[o+15] = sp.MaxRange
		var flags uint8
		if sp.TargetsUnit {
			flags |= spellFlagTargetsUnit
		}
		if sp.Damaging {
			flags |= spellFlagDamaging
		}
		if sp.Restorative {
			flags |= spellFlagRestorative
		}
		if sp.Area {
			flags |= spellFlagArea
		}
		if sp.Defensive {
			flags |= spellFlagDefensive
		}
		if sp.HealHostile || sp.SelfCast {
			flags |= spellFlagRowSpecific
		}
		flags |= uint8(sp.AreaHits) << 6
		if sp.Rays != 0 {
			flags |= spellFlagAreaHitsHi
		}
		b[o+16] = flags
		binary.LittleEndian.PutUint32(b[o+17:o+21], uint32(sp.AreaDuration))
		b[o+21] = sp.Distribution
		b[o+22] = sp.Radius
		if sp.Rays != 0 {
			b[o+22] = sp.Rays
		}
		binary.LittleEndian.PutUint32(b[o+23:o+27], uint32(sp.SpellDuration))
		b[o+27] = byte(sp.EffectKind)
		b[o+28] = byte(sp.EffectMode)
		binary.LittleEndian.PutUint32(b[o+29:o+33], uint32(sp.EffectMagnitude))
		binary.LittleEndian.PutUint16(b[o+33:o+35], sp.EffectDuration)
		b[o+35] = sp.Complication
		o += spellRecordLen
	}
	// THE ITEM-WEIGHT SECTION, right after the spell table and before the
	// casting section (version 56): a uint16 count, then that many records of
	// code and per-unit weight, in the table's own code order.
	//
	// IT IS A TABLE THE WORLD WAS HANDED, the spell table's own kind, and it
	// is written for the same reason that one is rather than carried across a
	// decode the way the ghost template is: every actor's load is derived
	// against it, so a world decoded without it would recompute every load to
	// the empty sum the next time anything moved an item, and the resumed
	// mission's movement would diverge from the saved one's. The ghost's
	// fallback costs a raise that does not happen; this one would cost the
	// digest.
	binary.LittleEndian.PutUint16(b[o:o+2], uint16(len(w.itemWeights)))
	o += itemWeightCountLen
	for _, iw := range w.itemWeights {
		binary.LittleEndian.PutUint16(b[o:o+2], iw.Code)
		binary.LittleEndian.PutUint32(b[o+2:o+6], uint32(iw.Weight))
		o += itemWeightRecordLen
	}
	// THE CASTING SECTION, right after the spell table and before the script
	// section (version 49): the mission script's pending casts and the area
	// effects standing on cells. It sits HERE rather than past the script
	// because decodeScript consumes the rest of the buffer and returns no used
	// count — a section behind it would have to give it one, which is a
	// change to a section this story has no business in.
	o = w.encodeCasting(b, o)
	// THE SCRIPT-STATE SECTION, right after the casting section and before the
	// script section (version 50): the per-player formation modes and the
	// cell-record tails. It sits here for the casting section's own reason --
	// decodeScript consumes the rest of the buffer.
	o = w.encodeScriptState(b, o)
	// THE STRUCTURE SECTION, right after the script-state section and before
	// the script section (version 58, 1033 B3): one record per placed type-4
	// record on the map this world was built from. It sits here for the
	// casting section's own reason above — decodeScript consumes the rest of
	// the buffer.
	o = w.encodeStructures(b, o)
	o = w.encodeItemState(b, o)
	o = w.encodeScrolls(b, o)
	// The script section, then THE RELATION, close the form — so every offset
	// above them is where it was in version 8 and the tail is where a reader
	// looks for what is new.
	//
	// The relation is LAST rather than beside the planes it most resembles, and
	// what decides that is what the form is indexed by. The two sections between
	// the header and here are each sized by a count the header declares — cells
	// and entities — and every offset inside them is documented against that
	// count. This block is sized by neither: its length is a compile-time
	// constant of this build. In the middle it would move sixty-eight documented
	// offsets and buy nothing; at the end it moves none, and the two counted
	// sections keep meaning exactly what the header says they mean.
	o = w.encodeScript(b, o)
	o = w.encodeOriginalDead(b, o)
	o = w.encodeInstanceWeights(b, o)
	w.encodeActorLoads(b, o)
	w.relations.encodeInto(b[len(b)-relationLen:])
	payload := w.appendAttackNotices(w.appendSavedWorldEffects(w.appendSavedFormations(w.appendStructureUses(w.appendScorched(w.appendGroupRoam(w.appendActionClocks(w.appendCarriedResumeState(w.appendSavedObjects(w.appendSavedCellPlanes(w.appendSavedMotions(w.appendNativeStrides(w.appendSavedGroupPlayerSection(w.appendSavedStructureSection(w.appendSavedGroups(w.appendSessionClock(b))))))))))))))))
	payload = w.appendNativeItemRecords(w.appendNativeLiveBlocks(w.appendPlayerParticipants(w.appendNativeActorBases(w.appendBookSelections(w.appendNativeClasses(w.appendROM2ScriptState(w.appendNativeTraining(w.appendAreaCosts(w.appendCreatureSpells(w.appendPendingOrders(w.appendTactical(w.appendStructureBlocking(w.appendCurrentTerminalActors(w.appendSavedSpellGraph(w.appendAutoHealing(w.appendSpellDeliveries(w.appendEntityIDFloor(payload))))))))))))))))))
	return w.appendNativeScalars(payload)
}

// MarshalBinary returns the world's canonical byte form: versioned,
// self-contained, and carrying every field the world's identity consists of.
//
// The bytes depend on the logical world alone. Entities are stored in ascending
// id order, so two worlds holding the same entities marshal identically however
// they were built.
func (w *World) MarshalBinary() ([]byte, error) {
	return w.MarshalBinaryInto(nil)
}

// MarshalBinaryInto writes the full canonical byte form from offset zero,
// reusing dst's capacity when possible. The result may alias dst. The caller
// owns this storage; the World never retains it. An error leaves dst unchanged.
// MarshalBinary returns independent storage on every call.
func (w *World) MarshalBinaryInto(dst []byte) ([]byte, error) {
	if err := w.nativeItemsFault(); err != nil {
		return nil, err
	}
	if err := w.currentPlayersFault(); err != nil {
		return nil, err
	}
	if err := w.nativeBasisFault(); err != nil {
		return nil, err
	}
	for _, e := range w.entities {
		if e.AdmittedBookSpell > 28 {
			return nil, fmt.Errorf("invalid admitted book selection")
		}
	}
	if err := w.nativeTrainingFault(); err != nil {
		return nil, err
	}
	if err := w.nativeClassFault(); err != nil {
		return nil, err
	}
	if err := w.pendingOrderFault(); err != nil {
		return nil, err
	}
	if err := w.tacticalFault(); err != nil {
		return nil, err
	}
	if err := w.pendingAttackFault(); err != nil {
		return nil, err
	}
	if err := w.spellDeliveryFault(); err != nil {
		return nil, err
	}
	if w.entityIDFloor > entityIDLimit {
		return nil, fmt.Errorf("sim: entity identity floor exceeds namespace")
	}
	if err := w.areaOwnershipFault(); err != nil {
		return nil, err
	}
	if err := w.structureUseFault(); err != nil {
		return nil, err
	}
	if err := w.worldEffectOrderFault(); err != nil {
		return nil, err
	}
	if err := w.savedSpellGraphFault(true); err != nil {
		return nil, err
	}
	if err := w.savedWorldEffectsFault(); err != nil {
		return nil, err
	}
	if err := currentTerminalActorsFault(w.currentTerminalActors, w.bounds); err != nil {
		return nil, err
	}
	for _, row := range w.currentTerminalActors {
		if indexOfEntity(w.entities, row.ID) >= 0 {
			return nil, fmt.Errorf("sim: current terminal actor %d collides with a live entity", row.ID)
		}
		for _, dead := range w.originalDead {
			if dead.ID == row.ID {
				return nil, fmt.Errorf("sim: current terminal actor %d collides with a retained dead actor", row.ID)
			}
		}
	}
	if err := w.carriedResumeStateFault(); err != nil {
		return nil, err
	}
	if err := w.validateSavedObjects(); err != nil {
		return nil, err
	}
	if _, err := w.savedObjectsBinarySize(); err != nil {
		return nil, err
	}
	if err := w.savedCellPlaneStateFault(); err != nil {
		return nil, err
	}
	if err := w.savedMotionFault(); err != nil {
		return nil, err
	}
	if w.hasSavedStructures {
		if err := validateSavedStructures(w.structures, w.savedStructures, w.savedStructureCells); err != nil {
			return nil, err
		}
	} else if len(w.savedStructures) != 0 || len(w.savedStructureCells) != 0 {
		return nil, fmt.Errorf("sim: absent saved structure mode carries state")
	}
	if err := savedGroupsFault(w.savedGroups, w.entities, w.originalDead); err != nil {
		return nil, err
	}
	if err := w.sessionClockFault(); err != nil {
		return nil, err
	}
	if err := w.rom2ScriptStateFault(); err != nil {
		return nil, err
	}
	for _, e := range w.entities {
		if !e.ActionClock.Known && e.ActionClock.End != 0 {
			return nil, fmt.Errorf("sim: absent action clock carries a deadline")
		}
	}
	if err := sourceBindingsFault(w.entities); err != nil {
		return nil, err
	}
	if err := w.sourceEquipmentFault(); err != nil {
		return nil, err
	}
	if err := w.itemWeightsFault(); err != nil {
		return nil, err
	}
	for i := range w.entities {
		if err := w.entities[i].SourceBinding.Validate(w.entities[i]); err != nil {
			return nil, err
		}
		if err := w.entities[i].ActorLoad.Validate(); err != nil {
			return nil, err
		}
		if state := w.entities[i].CurrentActorLoad(); state != nil {
			if err := state.Validate(); err != nil {
				return nil, err
			}
			if !state.Inventory.ContainerPresent && len(w.carried[i]) != 0 {
				return nil, fmt.Errorf("sim: absent actor container carries items")
			}
		}
		if err := turnFault(w.entities[i]); err != nil {
			return nil, fmt.Errorf("sim: entity %d: %w", w.entities[i].ID, err)
		}
		if err := strideFault(w.entities[i]); err != nil {
			return nil, fmt.Errorf("sim: entity %d: %w", w.entities[i].ID, err)
		}
	}
	return w.encodeInto(dst), nil
}

// UnmarshalBinary replaces the whole of w with the world encoded in data.
//
// It is MarshalBinary's inverse and not a field writer: it is the one exported
// call besides Step that changes a world, and it changes all of it at once. A
// caller still cannot set a position, the tick, the bounds or the RNG
// individually.
//
// A ROUTE is refused on five counts besides, and each is a state the tick cannot
// produce: a route on a unit holding no target, a route whose last cell is not
// that unit's target, a cell out of bounds, a cell its own unit's DOMAIN cannot
// cross, and two consecutive cells that are not neighbours. A unit with a target
// and NO route is accepted — that is the state before a first routing, and it is
// what every world this package constructs begins in.
//
// The lengths are checked without trusting the declared count. The grid's length
// comes from the bounds; the record span is the declared count multiplied by the
// record width in int64, which is required to fit in what is left; and the route
// section is then required to consume the remainder exactly. A count too large
// fails the first of those, a count too small leaves the route reader parsing
// record bytes and failing the second, and truncated and over-long stay the same
// comparison.
func (w *World) UnmarshalBinary(data []byte) error {
	if len(data) > 0 && data[0] == nativeScalarFormVersion {
		return w.unmarshalNativeScalars(data)
	}
	if len(data) > 0 && data[0] == nativeItemFormVersion {
		return w.unmarshalNativeItemRecords(data)
	}
	if len(data) > 0 && data[0] == nativeLiveFormVersion {
		return w.unmarshalNativeLiveBlocks(data)
	}
	if len(data) > 0 && data[0] == currentPlayerFormVersion {
		return w.unmarshalCurrentPlayers(data)
	}
	if len(data) > 0 && data[0] == playerParticipantFormVersion {
		return w.unmarshalPlayerParticipants(data)
	}
	if len(data) > 0 && data[0] == nativeBasisFormVersion {
		return w.unmarshalNativeActorBases(data)
	}
	if len(data) > 0 && data[0] == bookSelectionFormVersion {
		return w.unmarshalBookSelections(data)
	}
	if len(data) > 0 && data[0] == nativeClassFormVersion {
		return w.unmarshalNativeClasses(data)
	}
	if len(data) > 0 && data[0] == rom2ScriptFormVersion {
		return w.unmarshalROM2ScriptState(data)
	}
	if len(data) > 0 && data[0] == nativeTrainingFormVersion {
		return w.unmarshalNativeTraining(data)
	}
	if len(data) > 0 && data[0] == areaCostFormVersion {
		return w.unmarshalAreaCosts(data)
	}
	if len(data) > 0 && data[0] == creatureSpellFormVersion {
		return w.unmarshalCreatureSpells(data)
	}
	if len(data) > 0 && data[0] == pendingOrderFormVersion {
		return w.unmarshalPendingOrders(data)
	}
	if len(data) > 0 && data[0] == tacticalFormVersion {
		return w.unmarshalTactical(data)
	}
	if len(data) > 0 && data[0] == structureBlockingFormVersion {
		return w.unmarshalStructureBlocking(data)
	}
	if len(data) > 0 && data[0] == currentTerminalFormVersion {
		return w.unmarshalCurrentTerminalActors(data)
	}
	if len(data) > 0 && (data[0] == spellGraphFormVersion || data[0] == currentAreaFormVersion) {
		return w.unmarshalSavedSpellGraph(data)
	}
	if len(data) > 0 && data[0] == autoHealingFormVersion {
		return w.unmarshalAutoHealing(data)
	}
	if len(data) > 0 && data[0] != formatVersion {
		return fmt.Errorf("sim: unknown byte form version %d, this build writes and reads %d only", data[0], formatVersion)
	}
	return w.unmarshalBinary(data)
}

func (w *World) unmarshalBinary(data []byte) error {
	if len(data) < headerLen {
		return fmt.Errorf("sim: byte form truncated: %d byte(s), header alone is %d", len(data), headerLen)
	}
	if data[0] != formatVersion && data[0] != 94 {
		return fmt.Errorf("sim: unknown byte form version %d, this build writes and reads %d only",
			data[0], formatVersion)
	}
	var deliveryState spellDeliveryState
	if data[0] == formatVersion {
		var err error
		data, deliveryState, err = splitSpellDeliveries(data)
		if err != nil {
			return err
		}
	}
	data, entityIDFloor, err := splitEntityIDFloor(data)
	if err != nil {
		return err
	}
	data, notices, err := splitAttackNotices(data)
	if err != nil {
		return err
	}
	data, worldEffects, err := splitSavedWorldEffects(data)
	if err != nil {
		return err
	}
	data, exactFormations, formationOwners, haveFormations, err := splitSavedFormations(data)
	if err != nil {
		return err
	}
	data, structureMetadata, structureUses, err := splitStructureUses(data)
	if err != nil {
		return err
	}
	data, scorched, err := splitScorched(data)
	if err != nil {
		return err
	}
	data, groupRoam, err := splitGroupRoam(data)
	if err != nil {
		return err
	}
	data, actionClocks, err := splitActionClocks(data)
	if err != nil {
		return err
	}
	data, carriedResume, err := splitCarriedResumeState(data)
	if err != nil {
		return err
	}
	data, objects, err := splitSavedObjects(data)
	if err != nil {
		return err
	}
	data, planes, err := splitSavedCellPlanes(data)
	if err != nil {
		return err
	}
	data, motion, err := splitSavedMotions(data)
	if err != nil {
		return err
	}
	data, strides, err := splitNativeStrides(data)
	if err != nil {
		return err
	}
	data, playerSection, err := splitSavedGroupPlayerSection(data)
	if err != nil {
		return err
	}
	data, hasStructures, savedStructures, savedCells, err := splitSavedStructureSection(data)
	if err != nil {
		return err
	}
	data, saved, err := splitSavedGroups(data)
	if err != nil {
		return err
	}
	if err := applySavedGroupPlayerSection(saved, playerSection); err != nil {
		return err
	}
	if haveFormations {
		binding := &World{savedGroups: saved}
		if err := binding.ImportSavedPlayerFormations(exactFormations, formationOwners); err != nil {
			return err
		}
		saved = binding.savedGroups
	}
	body, hasClock, fullTick, err := splitSessionClock(data)
	if err != nil {
		return err
	}
	data = body

	mode := Mode(data[29])
	if !mode.defined() {
		return fmt.Errorf("sim: byte form names routing mode %d, which is not defined", data[29])
	}

	b := Bounds{
		Width:  int32(binary.LittleEndian.Uint32(data[17:21])),
		Height: int32(binary.LittleEndian.Uint32(data[21:25])),
	}
	cells := gridCells(b)
	if declared := binary.LittleEndian.Uint32(data[30:34]); int64(declared) != cells {
		return fmt.Errorf("sim: byte form declares %d grid cell(s), want %d for bounds %dx%d",
			declared, cells, b.Width, b.Height)
	}
	// The relation is counted in with the planes even though it sits at the far
	// end, because this is the one comparison holding the buffer's whole length.
	// A buffer too short for both fails here rather than at a slice expression
	// further down, and the block declares no length of its own to disagree with
	// the constant.
	if int64(len(data)-headerLen) < 3*cells+relationLen {
		return fmt.Errorf("sim: byte form truncated: %d byte(s) after a %d-byte header, and its three planes and relation are %d",
			len(data)-headerLen, headerLen, 3*cells+relationLen)
	}
	// The comparison above bounds cells by the buffer's own length, so the
	// narrowing is safe here and nowhere earlier.
	n1 := headerLen + int(cells)
	n2 := n1 + int(cells)
	records := n2 + int(cells)

	// newGrid and newPlane are the constructor's own checks, so a plane that
	// reaches a world by being decoded is held to what a plane that reaches a
	// world by being passed is held to. Both copy, so nothing in the decoded
	// world reaches back into data.
	grid, err := newGrid(b, data[headerLen:n1], hasStructures || planes != nil)
	if err != nil {
		return err
	}
	cost, err := newPlane(b, data[n1:n2], defaultCost, "cost")
	if err != nil {
		return err
	}
	height, err := newPlane(b, data[n2:records], 0, "height")
	if err != nil {
		return err
	}
	// THE RELATION IS TAKEN OFF THE END, before the sections in front of it are
	// read, so that each of those still sees a buffer ending where its own
	// contract says it ends — the route reader consuming what is left after the
	// records, and the script section required to consume the remainder exactly.
	// The length check above is what makes this slice expression safe.
	//
	// NewRelations is the constructor's own check, so a relation that reaches a
	// world by being decoded is held to what one that reaches a world by being
	// passed is held to, and it copies, so nothing in the decoded world reaches
	// back into data. Its length cannot be wrong here: the slice is exactly
	// relationLen by construction, and the error arm is the totality that keeps
	// this the same call the constructor makes rather than a second rule.
	rel, err := NewRelations(data[len(data)-relationLen:])
	if err != nil {
		return err
	}

	// The record span is the declared count multiplied out in int64, so two
	// int32-sized factors cannot wrap into a small plausible span, and it is
	// required to fit in what the buffer still holds.
	declared := binary.LittleEndian.Uint32(data[25:29])
	span := int64(declared) * entityLen
	if avail := int64(len(data) - records); span > avail {
		return fmt.Errorf("sim: byte form declares %d entities, whose records are %d byte(s), "+
			"and carries %d byte(s) after its planes", declared, span, avail)
	}
	n := int(declared)

	ents := make([]Entity, n)
	for i := range ents {
		o := records + entityLen*i
		e := Entity{
			ID:      EntityID(binary.LittleEndian.Uint32(data[o : o+4])),
			X:       int32(binary.LittleEndian.Uint32(data[o+4 : o+8])),
			Y:       int32(binary.LittleEndian.Uint32(data[o+8 : o+12])),
			TargetX: int32(binary.LittleEndian.Uint32(data[o+12 : o+16])),
			TargetY: int32(binary.LittleEndian.Uint32(data[o+16 : o+20])),
			Class:   int32(binary.LittleEndian.Uint32(data[o+20 : o+24])),
			HP:      int32(binary.LittleEndian.Uint32(data[o+26 : o+30])),
			MaxHP:   int32(binary.LittleEndian.Uint32(data[o+30 : o+34])),
			Domain:  Domain(data[o+34]),
			Speed:   int32(binary.LittleEndian.Uint32(data[o+35 : o+39])),

			Transit:      binary.LittleEndian.Uint16(data[o+39 : o+41]),
			TransitTotal: binary.LittleEndian.Uint16(data[o+41 : o+43]),
			GroupSpeed:   data[o+43],

			AttackTarget:    EntityID(binary.LittleEndian.Uint32(data[o+44 : o+48])),
			AttackPhase:     AttackPhase(data[o+49]),
			AttackCountdown: int32(binary.LittleEndian.Uint32(data[o+50 : o+54])),
			AttackCharge:    int32(binary.LittleEndian.Uint32(data[o+54 : o+58])),
			AttackRelax:     int32(binary.LittleEndian.Uint32(data[o+58 : o+62])),
			ToHit:           int32(binary.LittleEndian.Uint32(data[o+62 : o+66])),
			Defence:         int32(binary.LittleEndian.Uint32(data[o+66 : o+70])),
			Absorption:      int32(binary.LittleEndian.Uint32(data[o+70 : o+74])),
			DamageBase:      int32(binary.LittleEndian.Uint32(data[o+74 : o+78])),
			DamageSpread:    int32(binary.LittleEndian.Uint32(data[o+78 : o+82])),

			Group: binary.LittleEndian.Uint32(data[o+83 : o+87]),
			Owner: binary.LittleEndian.Uint32(data[o+87 : o+91]),

			// THE FACING IS CARRIED WHOLE AND NO VALUE IS REFUSED, which is the
			// one tail here that needs no check at all. Every byte names one of
			// the eight directions under FacingDir, so there is nothing to fold
			// and nothing to reject — and a refusal would make a state the
			// constructor accepts a state this reader will not read back, which
			// is the asymmetry every clause below exists to prevent.
			Facing: data[o+91],

			// THE DECAY BLOCK: the stage, the dwell it still owes, and the
			// dying time a fresh body takes its dwell from. Each is carried
			// whole; what is refused is below, beside the residue rules the
			// other tails already stand in.
			Decay:     DecayStage(data[o+92]),
			Dwell:     binary.LittleEndian.Uint16(data[o+93 : o+95]),
			DyingTime: int32(binary.LittleEndian.Uint32(data[o+95 : o+99])),

			// THE SIGHT RANGE IS THE RECORD'S LAST BYTE, CARRIED WHOLE AND WITH
			// NO VALUE REFUSED, on the facing's own grounds and not on a second
			// set: every byte is a range this build can march at, so there is
			// nothing to fold and nothing to reject. Zero included — a unit at
			// zero lights the cell it stands in and no other, which is a state
			// the constructor accepts and therefore a state this reader must
			// read back.
			ScanRange: data[o+99],

			// THE ACTOR STATE, THE RING AND THE LEG are the record's new
			// tail (version 21, 0099). All six are carried whole here; what
			// is refused is below, in patrolFault's own four shapes.
			ActorState:  data[o+100],
			PatrolHeadX: int32(binary.LittleEndian.Uint32(data[o+101 : o+105])),
			PatrolHeadY: int32(binary.LittleEndian.Uint32(data[o+105 : o+109])),
			PatrolTailX: int32(binary.LittleEndian.Uint32(data[o+109 : o+113])),
			PatrolTailY: int32(binary.LittleEndian.Uint32(data[o+113 : o+117])),
			PatrolLeg:   data[o+117],

			// THE REACH IS AT +118 (version 23, 0104): carried whole here;
			// what is refused is below, in reachFault's one shape.
			Reach: data[o+118],

			// THE POST (version 24, 0106): PostX at +119, PostY at +123, carried
			// whole with no value refused — every coordinate pair is a state a
			// setter can leave behind, so there is no fault shape for it below, on
			// the commanded cell's own ground.
			PostX: int32(binary.LittleEndian.Uint32(data[o+119 : o+123])),
			PostY: int32(binary.LittleEndian.Uint32(data[o+123 : o+127])),

			// THE REGENERATION BLOCK IS THE RECORD'S NEW LAST FIELD
			// (version 25, 0109): the four int32 carried whole here,
			// exactly as the post is, on the health pair's own ground;
			// the two remainders are carried whole here too — what is
			// refused is below, in regenFault's two shapes.
			Mana:              int32(binary.LittleEndian.Uint32(data[o+127 : o+131])),
			MaxMana:           int32(binary.LittleEndian.Uint32(data[o+131 : o+135])),
			HealthRegenPeriod: int32(binary.LittleEndian.Uint32(data[o+135 : o+139])),
			ManaRegenPeriod:   int32(binary.LittleEndian.Uint32(data[o+139 : o+143])),
			HealthHundredths:  data[o+143],
			ManaHundredths:    data[o+144],

			// THE COMMAND GROUP IS THE RECORD'S NEW LAST FIELD (version 27): carried
			// whole here with no value refused — this task gives it no writer and
			// so nothing to check it against below, on the owner slot's own ground.
			CommandGroup: binary.LittleEndian.Uint32(data[o+145 : o+149]),

			// AN ENTITY'S EXPERIENCE FROM USE IS THE RECORD'S NEW LAST FIELD (version
			// 35): the six slot experiences at +149 through +172, Mind at +173 and
			// the experience value at +177, both carried whole here on the command
			// group's own ground. The credited slot is read whole too; what is
			// refused is below, in experienceFault's two shapes. GainsXP is NOT here
			// — its own byte is read by the switch below, on the target-presence
			// byte's own rule.
			SkillXP: [skillSlots]int32{
				int32(binary.LittleEndian.Uint32(data[o+149 : o+153])),
				int32(binary.LittleEndian.Uint32(data[o+153 : o+157])),
				int32(binary.LittleEndian.Uint32(data[o+157 : o+161])),
				int32(binary.LittleEndian.Uint32(data[o+161 : o+165])),
				int32(binary.LittleEndian.Uint32(data[o+165 : o+169])),
				int32(binary.LittleEndian.Uint32(data[o+169 : o+173])),
			},
			Mind:    int32(binary.LittleEndian.Uint32(data[o+173 : o+177])),
			XPValue: int32(binary.LittleEndian.Uint32(data[o+177 : o+181])),
			XPSlot:  data[o+181],

			// AN ENTITY'S OWN SPELLBOOK IS THE RECORD'S LAST FIELD BEFORE
			// THIS STORY'S OWN TAIL (version 36, 0127 FR-4b): carried
			// whole here with no value refused — KnownSpells' own doc
			// (world.go) says why: every bit is a legal state the
			// constructor accepts, so refusing any here would make a
			// world this package can produce a world it cannot read back.
			KnownSpells: binary.LittleEndian.Uint32(data[o+183 : o+187]),

			// AN ENTITY'S SIX SKILL LEVELS ARE THE RECORD'S NEW LAST FIELD (version
			// 39): one int32 per slot at +187 through +210, carried whole here with
			// no value refused — Skill's own doc (world.go) and experienceFault's
			// silence on it say why: every int32 is a state the constructor accepts,
			// so refusing one here would make a world this package can produce a
			// world it cannot read back.
			Skill: [skillSlots]int32{
				int32(binary.LittleEndian.Uint32(data[o+187 : o+191])),
				int32(binary.LittleEndian.Uint32(data[o+191 : o+195])),
				int32(binary.LittleEndian.Uint32(data[o+195 : o+199])),
				int32(binary.LittleEndian.Uint32(data[o+199 : o+203])),
				int32(binary.LittleEndian.Uint32(data[o+203 : o+207])),
				int32(binary.LittleEndian.Uint32(data[o+207 : o+211])),
			},

			// A WEAPON'S OWN SPELL IS THE RECORD'S NEW LAST FIELD (version 41): the
			// id, a uint16, and the level, an int32, both carried whole here with no
			// value refused — WeaponSpell's own doc (world.go) says why: an id
			// naming no row this world's table loaded is weaponSpell's own runtime
			// answer, not a decode-time fault.
			WeaponSpell:      binary.LittleEndian.Uint16(data[o+211 : o+213]),
			WeaponSpellLevel: int32(binary.LittleEndian.Uint32(data[o+213 : o+217])),
			// THE AUTOCAST PAIR AND THE SPELL EFFECT MARK (version 45), read back
			// whole with no value refused on WeaponSpell's own ground above.
			AutoSpell: binary.LittleEndian.Uint16(data[o+217 : o+219]),
			CastWait:  data[o+219],
			// THE MARK'S REMAINING TICKS ARE A WORD SINCE VERSION 50, which moved the
			// two bytes after it up one.
			SpellFX:      binary.LittleEndian.Uint16(data[o+220 : o+222]),
			SpellFXSpell: data[o+222],
			// THE ESCORT TRIPLE (version 50), the id and the range read back whole
			// here; the presence bit is a switch below with the record's other three.
			EscortTarget: EntityID(binary.LittleEndian.Uint32(data[o+224 : o+228])),
			EscortRange:  data[o+229],
			Protection: [5]int32{
				int32(binary.LittleEndian.Uint32(data[o+230 : o+234])),
				int32(binary.LittleEndian.Uint32(data[o+234 : o+238])),
				int32(binary.LittleEndian.Uint32(data[o+238 : o+242])),
				int32(binary.LittleEndian.Uint32(data[o+242 : o+246])),
				int32(binary.LittleEndian.Uint32(data[o+246 : o+250])),
			},
			TokenSize:        data[o+250],
			SeeInvisible:     data[o+251],
			Reaction:         int32(binary.LittleEndian.Uint32(data[o+252 : o+256])),
			Spirit:           int32(binary.LittleEndian.Uint32(data[o+256 : o+260])),
			KillCreditSource: EntityID(binary.LittleEndian.Uint32(data[o+261 : o+265])),
			KillCreditSpell:  int8(data[o+266]),
			Load:             int32(binary.LittleEndian.Uint32(data[o+267 : o+271])),
			Capacity:         int32(binary.LittleEndian.Uint32(data[o+271 : o+275])),
			MapUnitID:        binary.LittleEndian.Uint16(data[o+275 : o+277]),
			Resistance: [5]uint8{
				data[o+277], data[o+278], data[o+279], data[o+280], data[o+281],
			},
			Withdraw:      int32(binary.LittleEndian.Uint32(data[o+282 : o+286])),
			Wimpy:         int32(binary.LittleEndian.Uint32(data[o+286 : o+290])),
			DesiredFacing: data[o+291],
			TurnRemaining: data[o+292],
			TurnTotal:     data[o+293],
		}
		for k := 0; k < 4; k++ {
			e.PotionStats[k] = int32(binary.LittleEndian.Uint32(data[o+294+4*k:]))
			e.PotionHeadroom[k] = int32(binary.LittleEndian.Uint32(data[o+310+4*k:]))
			if e.PotionStats[k] < 0 || e.PotionStats[k] > 100 || e.PotionHeadroom[k] < 0 || e.PotionHeadroom[k] > 100 {
				return fmt.Errorf("sim: entity %d has invalid Potion attribute state", e.ID)
			}
		}
		e.HumanMovement = HumanMovement{
			Present:     data[o+326] == 1,
			RawSpeed:    int16(binary.LittleEndian.Uint16(data[o+327:])),
			NativeSpeed: int32(binary.LittleEndian.Uint32(data[o+329:])),
			Load:        int32(binary.LittleEndian.Uint32(data[o+333:])),
			Capacity:    int32(binary.LittleEndian.Uint32(data[o+337:])),
		}
		if data[o+326] > 1 || (!e.HumanMovement.Present && e.HumanMovement != (HumanMovement{})) ||
			(e.HumanMovement.Present && (e.HumanMovement.NativeSpeed != e.Speed || e.HumanMovement.Load != e.Load || e.HumanMovement.Capacity != e.Capacity)) {
			return fmt.Errorf("sim: entity %d has invalid retained Human movement context", e.ID)
		}
		e.SecondBase, e.SecondSpread = data[o+454], data[o+455]
		e.CurrentProfileBasis = CurrentProfileBasis(data[o+456])
		if data[o+entityLen-1] > 1 {
			return fmt.Errorf("sim: invalid source Group owner presence")
		}
		if _, err := binary.Decode(data[o+entityLenV75:o+entityLen], binary.LittleEndian, &e.SourceBinding); err != nil {
			return err
		}
		book, err := decodeSpellbook(data[o+341:o+entityLenV71], e.KnownSpells)
		if err != nil {
			return fmt.Errorf("sim: entity %d: %w", e.ID, err)
		}
		e.Book = book
		// The two 0/1 bytes, refused rather than read as truthy, for the reason
		// the target-presence byte below is: any other reading would map two byte
		// forms onto one world and a pinned form would stop meaning exactly one.
		switch data[o+48] {
		case 0:
		case 1, 2:
			e.HasAttackTarget = true
			e.AttackTargetKind = AttackTargetKind(data[o+48] - 1)
		case 3:
			e.HasAttackTarget, e.AcquirePursuit = true, true
			e.AttackTargetKind = AttackTargetUnit
		case 4:
			e.HasAttackTarget, e.PursuitIdle = true, true
			e.AttackTargetKind = AttackTargetUnit
		default:
			return fmt.Errorf("sim: entity record %d: attack target tag is %d, want 0, 1, 2, 3 or 4", i, data[o+48])
		}
		switch data[o+82] {
		case 0:
		case 1:
			e.AlwaysHits = true
		default:
			return fmt.Errorf("sim: entity record %d: always-hits byte is %d, want 0 or 1", i, data[o+82])
		}
		switch data[o+290] {
		case 0:
		case 1:
			e.Humanoid = true
		default:
			return fmt.Errorf("sim: entity record %d: Humanoid byte is %d, want 0 or 1", i, data[o+290])
		}
		// turnFault is NOT checked here: RotationSpeed is a derived-state field
		// decoded later in itembinary.go's decodeItemState, so an entity read to
		// this point always carries RotationSpeed's zero value. decodeItemState
		// checks turnFault once RotationSpeed is actually populated. THE OFF-MAP
		// BIT (version 48), refused rather than read as truthy on the two bytes
		// above's own ground. There is no fault function beside it: no combination
		// of this bit with any other field is illegal — an off-map unit may hold
		// any cell, any health, any order and any group, which is precisely the
		// arm's untouched list.
		switch data[o+223] {
		case 0:
		case 1:
			e.OffMap = true
		default:
			return fmt.Errorf("sim: entity record %d: off-map byte is %d, want 0 or 1", i, data[o+223])
		}
		// THE ESCORT-PRESENCE BIT (version 50), refused rather than read as truthy
		// on the three bytes above's own ground. What combinations of it with the
		// actor state are illegal is patrolFault's, called below with the rest of
		// the record's shape rules.
		switch data[o+228] {
		case 0:
		case 1:
			e.HasEscortTarget = true
		default:
			return fmt.Errorf("sim: entity record %d: escort-presence byte is %d, want 0 or 1", i, data[o+228])
		}
		switch data[o+260] {
		case 0:
		case 1:
			e.SuppressCorpseLoot = true
		default:
			return fmt.Errorf("sim: entity record %d: corpse-loot suppression byte is %d, want 0 or 1", i, data[o+260])
		}
		switch data[o+265] {
		case 0:
		case 1:
			e.HasKillCredit = true
		default:
			return fmt.Errorf("sim: entity record %d: kill-credit presence byte is %d, want 0 or 1", i, data[o+265])
		}
		if !e.HasKillCredit && e.KillCreditSource != 0 {
			return fmt.Errorf("sim: entity record %d: absent kill credit retains source %d", i, e.KillCreditSource)
		}
		// The same check the constructor makes, so a cycle that reaches a world
		// by being decoded is held to what one that reaches a world by being
		// passed is held to.
		if err := attackFault(e); err != nil {
			return fmt.Errorf("sim: entity record %d: %w", i, err)
		}
		// The four shapes the constructor NORMALISES, refused here — which keeps
		// the two in the relation every other order field already stands in.
		// An order naming its own unit is one no order can set; the three cycle
		// fields on a unit holding no order are residue of an order that ended.
		if e.HasAttackTarget && e.AttackTargetKind == AttackTargetUnit && e.AttackTarget == e.ID {
			return fmt.Errorf("sim: entity record %d: an attack order naming its own unit", i)
		}
		if !e.HasAttackTarget &&
			(e.AttackTarget != 0 || e.AttackPhase != AttackReady || e.AttackCountdown != 0) {
			return fmt.Errorf("sim: entity record %d: victim %d, phase %d and %d tick(s) owed on a unit holding no attack order",
				i, e.AttackTarget, e.AttackPhase, e.AttackCountdown)
		}
		// The same check the constructor makes, so a pair that reaches a world
		// by being decoded is held to what a pair that reaches one by being
		// passed is held to.
		if err := transitFault(e); err != nil {
			return fmt.Errorf("sim: entity record %d: %w", i, err)
		}
		// Refused rather than folded into the ground domain, for the reason the
		// target-presence byte below is refused rather than read as truthy: a
		// byte normalised here would map two forms onto one world.
		if !e.Domain.defined() {
			return fmt.Errorf("sim: entity record %d: movement domain is %d, which is not defined",
				i, data[o+34])
		}
		switch data[o+24] {
		case 0:
		case 1:
			e.HasTarget = true
		case 2, 3:
			if e.TargetY != 0 {
				return fmt.Errorf("sim: pending attack carries a second destination operand")
			}
			e.PendingAttackTarget = EntityID(uint32(e.TargetX))
			e.PendingAttackTargetKind, e.HasPendingAttackTarget = AttackTargetKind(data[o+24]-2), true
			e.TargetX = 0
		default:
			// Not a shape refusal, but the one that keeps the encoding
			// injective: were any nonzero byte read as true, two different byte
			// forms would decode to one world and a pinned form would stop
			// meaning exactly one.
			return fmt.Errorf("sim: entity record %d: destination tag is %d, want 0, 1, 2 or 3", i, data[o+24])
		}
		// A count at the limit is a state the tick that raised it already spent,
		// and a count on an entity with no target is residue of a target that was
		// cleared. Neither can be written by this package, so both are refused
		// rather than normalised: normalising them would map two byte forms onto
		// one world exactly as a truthy presence byte would.
		e.Stall = data[o+25]
		if e.Stall >= stallLimit {
			return fmt.Errorf("sim: entity record %d: stall count is %d, the limit is %d",
				i, e.Stall, stallLimit)
		}
		if e.Stall != 0 && !e.HasTarget {
			return fmt.Errorf("sim: entity record %d: stall count is %d on an entity with no target",
				i, e.Stall)
		}
		// A unit that is not alive takes no order, so a target on one is residue
		// of a state it has left. It is refused here for the reason the two counts
		// above are, and it is the decoder's half of a rule the constructor
		// NORMALISES: the two therefore stay in the relation they already stood in
		// — the constructor cannot produce what this reader will not read back.
		//
		// ONE refusal covers all three shapes the contract names, because the
		// other two cannot occur without this one. A stall count on a unit holding
		// no target is refused on the line above, and a route on one is refused by
		// decodeRoutes on the same ground — so a stall count or a route on a unit
		// that is not alive is refused either here, when the unit holds a target,
		// or by one of those two when it does not.
		if !e.Alive() && e.HasTarget {
			return fmt.Errorf("sim: entity record %d: a target at (%d,%d) on a unit at %d/%d, which is not alive",
				i, e.TargetX, e.TargetY, e.HP, e.MaxHP)
		}
		// A CROSSING on a unit that is not alive is refused on the same ground
		// and needs a clause of its own: a felled mover holds no order, so the
		// target refusal above cannot reach it, and its crossing is dropped
		// where the blow lands rather than left to run out.
		if !e.Alive() && (e.Transit != 0 || e.TransitTotal != 0) {
			return fmt.Errorf("sim: entity record %d: a transit of %d/%d on a unit at %d/%d, which is not alive",
				i, e.Transit, e.TransitTotal, e.HP, e.MaxHP)
		}
		// A GROUP TERM on a unit that is not alive is refused on the same
		// ground and needs its own clause for the same reason: a felled member
		// is unlinked from its group, and neither the target nor the transit
		// refusal above can reach a unit whose only residue is this byte.
		if !e.Alive() && e.GroupSpeed != 0 {
			return fmt.Errorf("sim: entity record %d: a group speed of %d on a unit at %d/%d, which is not alive",
				i, e.GroupSpeed, e.HP, e.MaxHP)
		}
		// THE GROUP WORD TAKES NO SUCH CLAUSE, and the omission is deliberate
		// enough to be written down beside four refusals it resembles. Those
		// four refuse residue — a state the tick that left it has already
		// spent. Membership is not residue: a felled unit is still the unit the
		// map placed in that group, and the count the script asks for is a count
		// of the LIVING members of a group whose dead members are still in it.
		// A refusal here would make "the last member dies and the count reaches
		// zero" a form this package cannot write down.
		//
		// NEITHER DOES THE OWNER SLOT, on the same ground and with a sharper
		// consequence. The dead stay on their roster exactly as they stay in
		// their group, and both hand-over arms are specified to write a felled
		// entity; a refusal here would make the world those arms produce one this
		// package cannot read back, which is the asymmetry every clause above
		// exists to prevent rather than to create. No value of the word is
		// refused either: the constructor accepts every one, so refusing any here
		// would break the same rule.
		//
		// AND NEITHER DOES THE FACING, on the group word's ground and with one
		// more of its own. A body faces the way it fell, so a facing on a unit
		// that is not alive is a fact about it rather than residue of a state it
		// has left; and no VALUE of it is refused because every byte names a
		// direction, where the mode, the domain and the phase each have bytes
		// that name nothing and are refused for it.
		// AN ATTACK ORDER on a unit that is not alive is refused on the same
		// ground and needs its own clause for the same reason: a felled attacker
		// is dropped out of its cycle where the blow lands, and none of the three
		// refusals above can reach a unit whose only residue is a victim. The
		// cycle fields need no clause of their own — a unit that is not alive and
		// holds no order is caught by the residue refusal already made.
		// DYING IS EXEMPT, on HERO-DYINGTICK-145's own ground: the order block
		// survives the window in which a body lies where it fell, unlike every
		// other residue this function refuses, so an attack order there is a
		// fact about that window and not leftover from one the entity has left.
		if !e.Alive() && !e.Dying() && e.HasAttackTarget {
			return fmt.Errorf("sim: entity record %d: an attack order on entity %d, by a unit at %d/%d, which is not alive",
				i, e.AttackTarget, e.HP, e.MaxHP)
		}
		// CAST RECOVERY on a unit that is not alive is refused on the same
		// ground as its attack cycle. clearFelled and the constructor both
		// clear it, so accepting it here would admit a form no tick produces.
		if !e.Alive() && e.CastWait != 0 {
			return fmt.Errorf("sim: entity record %d: cast recovery of %d tick(s) on a unit at %d/%d, which is not alive",
				i, e.CastWait, e.HP, e.MaxHP)
		}
		// The same check the constructor makes, so a stage that reaches a world
		// by being decoded is held to what one that reaches a world by being
		// passed is held to. It refuses the byte with no world to name — the
		// stage that means removal, and every byte above it.
		if err := decayFault(e); err != nil {
			return fmt.Errorf("sim: entity record %d: %w", i, err)
		}
		// THE PAIRING RULE, in the two directions the constructor NORMALISES —
		// which keeps the two in the relation every order field already stands
		// in. A positive stage and being not alive hold of the same entities, so
		// a living unit carrying a stage and a body carrying none are each a
		// world this package cannot produce.
		if e.Decay != DecayNone && e.Alive() {
			return fmt.Errorf("sim: entity record %d: decay stage %d on a unit at %d/%d, which is alive",
				i, e.Decay, e.HP, e.MaxHP)
		}
		if e.Decay == DecayNone && !e.Alive() {
			return fmt.Errorf("sim: entity record %d: no decay stage on a unit at %d/%d, which is not alive",
				i, e.HP, e.MaxHP)
		}
		// And a dwell on any stage but the first is residue of a dwell that ran
		// out, refused on the ground the stall count and the crossing already
		// are: only the first stage owes one.
		if e.Dwell != 0 && e.Decay != DecayFallen {
			return fmt.Errorf("sim: entity record %d: a dwell of %d at decay stage %d, which owes none",
				i, e.Dwell, e.Decay)
		}
		// THE ACTOR STATE, THE RING AND THE LEG: the same check the
		// constructor calls to learn what to normalise, called here to
		// refuse instead — patrolFault's four shapes are residue and
		// undefined-value refusals of exactly the kind every clause above
		// already stands in, so one function answers for both callers.
		if err := patrolFault(e); err != nil {
			return fmt.Errorf("sim: entity record %d: %w", i, err)
		}
		// The same check the constructor calls to learn what to normalise, called
		// here to refuse instead: a decoded reach of 0 has no value to fold onto,
		// on the strike distance's own ground. The same check the constructor
		// calls to learn what to fold, called here to refuse instead: a decoded
		// remainder above 99 has no fraction of a point left to name.
		if err := regenFault(e); err != nil {
			return fmt.Errorf("sim: entity record %d: %w", i, err)
		}
		// GAINS-XP, refused rather than read as truthy, for the reason the
		// target-presence byte below is: any other reading would map two byte
		// forms onto one world and a pinned form would stop meaning exactly one.
		switch data[o+182] {
		case 0:
		case 1:
			e.GainsXP = true
		default:
			return fmt.Errorf("sim: entity record %d: gains-xp byte is %d, want 0 or 1", i, data[o+182])
		}
		// The same check the constructor calls to learn what to refuse, called
		// here for the remaining malformed shape: a credited slot outside 0..5 has
		// no value to fold onto. Signed slot experience is the item-spell path's
		// observed state.
		if err := experienceFault(e); err != nil {
			return fmt.Errorf("sim: entity record %d: %w", i, err)
		}
		if i > 0 && e.ID <= ents[i-1].ID {
			return fmt.Errorf("sim: entity record %d: id %d does not ascend past %d", i, e.ID, ents[i-1].ID)
		}
		ents[i] = e
	}

	if err := applyNativeStrides(ents, strides); err != nil {
		return err
	}
	if err := applyActionClocks(ents, actionClocks); err != nil {
		return err
	}

	// A SECOND PASS over the records, because an attack order is the one field
	// whose legality depends on the rest of them: a victim no record declares is
	// an order pointing at nothing, and it cannot be tested until every id has
	// been read. The constructor normalises this shape and this refuses it, which
	// is the relation the shapes above already stand in.
	for i := range ents {
		if ents[i].HasKillCredit && indexOfEntity(ents, ents[i].KillCreditSource) < 0 {
			return fmt.Errorf("sim: entity record %d: kill-credit source %d is absent", i, ents[i].KillCreditSource)
		}
		if ents[i].HasAttackTarget && ents[i].AttackTargetKind == AttackTargetUnit && indexOfEntity(ents, ents[i].AttackTarget) < 0 {
			return fmt.Errorf("sim: entity record %d: an attack order on entity %d, which this form does not hold",
				i, ents[i].AttackTarget)
		}
	}

	loadEnd := len(data) - relationLen - 4
	if loadEnd < records+int(span) {
		return fmt.Errorf("sim: missing actor-load span")
	}
	loadHeader := binary.LittleEndian.Uint32(data[loadEnd:])
	loadLen := uint64(loadHeader & 0x7fffffff)
	if loadLen > uint64(loadEnd-records-int(span)) {
		return fmt.Errorf("sim: invalid actor-load span")
	}
	loadStart := loadEnd - int(loadLen)
	sourceEquipmentStart, sourceEquipmentEnd, actorsStart := loadStart, loadStart, loadStart
	if loadHeader>>31 != 0 {
		if loadLen < 4 {
			return fmt.Errorf("sim: missing source equipment prefix")
		}
		span := uint64(binary.LittleEndian.Uint32(data[loadStart:]))
		if span == 0 || span > loadLen-4 {
			return fmt.Errorf("sim: invalid source equipment prefix")
		}
		sourceEquipmentStart = loadStart + 4
		sourceEquipmentEnd = sourceEquipmentStart + int(span)
		actorsStart = sourceEquipmentEnd
	}
	if err := decodeActorLoads(data[actorsStart:loadEnd], ents); err != nil {
		return err
	}
	if err := sourceBindingsFault(ents); err != nil {
		return err
	}
	for i, e := range ents {
		if err := e.SourceBinding.Validate(e); err != nil {
			return err
		}
		// A source zero is retained data; its marker arrives in the bounded
		// actor section, after the ordinary entity records.
		if err := reachFault(e); err != nil {
			return fmt.Errorf("sim: entity record %d: %w", i, err)
		}
	}
	weightEnd := loadStart - 4
	if weightEnd < records+int(span) {
		return fmt.Errorf("sim: truncated instance-weight span")
	}
	weightLen := uint64(binary.LittleEndian.Uint32(data[weightEnd:]))
	if weightLen > uint64(weightEnd-records-int(span)) || weightLen%6 != 0 {
		return fmt.Errorf("sim: invalid instance-weight span")
	}
	weightStart := weightEnd - int(weightLen)
	deadEnd := weightStart - 4
	if deadEnd < records+int(span) {
		return fmt.Errorf("sim: missing original dead span")
	}
	deadLen := uint64(binary.LittleEndian.Uint32(data[deadEnd:]))
	if deadLen > uint64(deadEnd-records-int(span)) || deadLen%originalDeadRecordLen != 0 {
		return fmt.Errorf("sim: invalid original dead span")
	}
	deadStart := deadEnd - int(deadLen)
	tail := data[records+int(span) : deadStart]
	routes, used, err := decodeRoutes(tail, b, ents)
	if err != nil {
		return err
	}
	groups, gused, err := decodeGroups(tail[used:])
	if err != nil {
		return err
	}
	if err := applyGroupRoam(groups, groupRoam); err != nil {
		return err
	}
	if err := pickupCompletionGroupsFault(ents, groups, saved); err != nil {
		return err
	}
	sacks, sused, err := decodeSacks(b, tail[used+gused:])
	if err != nil {
		return err
	}
	carried, cused, err := decodeCarried(tail[used+gused+sused:], ents)
	if err != nil {
		return err
	}
	equipment, eused, err := decodeEquipment(tail[used+gused+sused+cused:], ents)
	if err != nil {
		return err
	}
	tused, err := decodeTreasure(tail[used+gused+sused+cused+eused:], ents)
	if err != nil {
		return err
	}
	purses, pused, err := decodePurses(tail[used+gused+sused+cused+eused+tused:])
	if err != nil {
		return err
	}
	spells, spused, err := decodeSpells(tail[used+gused+sused+cused+eused+tused+pused:])
	if err != nil {
		return err
	}
	itemWeights, iwused, err := decodeItemWeights(tail[used+gused+sused+cused+eused+tused+pused+spused:])
	if err != nil {
		return err
	}
	casts, bookCasts, effects, attached, caused, err := decodeCasting(tail[used+gused+sused+cused+eused+tused+pused+spused+iwused:])
	if err != nil {
		return err
	}
	for i, cast := range bookCasts {
		ci := indexOfEntity(ents, cast.Caster)
		if ci < 0 {
			return fmt.Errorf("sim: book cast %d names absent caster %d", i, cast.Caster)
		}
		if !ents[ci].Alive() {
			return fmt.Errorf("sim: book cast %d names caster %d at %d/%d, which is not alive",
				i, cast.Caster, ents[ci].HP, ents[ci].MaxHP)
		}
	}
	formations, cellTails, ssused, err := decodeScriptState(
		tail[used+gused+sused+cused+eused+tused+pused+spused+iwused+caused:])
	if err != nil {
		return err
	}
	structures, stused, err := decodeStructures(
		tail[used+gused+sused+cused+eused+tused+pused+spused+iwused+caused+ssused:], hasStructures)
	if err != nil {
		return err
	}
	for i, e := range ents {
		if e.HasAttackTarget && e.AttackTargetKind == AttackTargetStructure && indexOfStructure(structures, StructureID(e.AttackTarget)) < 0 {
			return fmt.Errorf("sim: entity record %d: attack names absent structure %d", i, e.AttackTarget)
		}
		if e.AttackTargetKind == AttackTargetStructure && e.AttackPhase == AttackCasting {
			return fmt.Errorf("sim: entity record %d: structure attack cannot hold a weapon cast", i)
		}
	}
	itemUsed, err := decodeItemState(
		tail[used+gused+sused+cused+eused+tused+pused+spused+iwused+caused+ssused+stused:],
		sacks, carried, equipment, ents)
	if err != nil {
		return err
	}
	scrolls, scrollUsed, err := decodeScrolls(tail[used+gused+sused+cused+eused+tused+pused+spused+iwused+caused+ssused+stused+itemUsed:], ents, b, spells)
	if err != nil {
		return err
	}
	st, err := decodeScript(
		tail[used+gused+sused+cused+eused+tused+pused+spused+iwused+caused+ssused+stused+itemUsed+scrollUsed:])
	if err != nil {
		return err
	}
	dead, err := decodeOriginalDead(data[deadStart:deadEnd], b, ents, carried, equipment)
	if err != nil {
		return err
	}
	weighted := World{entities: ents, sacks: sacks, carried: carried, equipment: equipment, scrollCasts: scrolls, itemWeights: itemWeights}
	if err := weighted.decodeSourceEquipment(data[sourceEquipmentStart:sourceEquipmentEnd]); err != nil {
		return err
	}
	if err := weighted.applySavedObjects(objects); err != nil {
		return err
	}
	if err := weighted.decodeInstanceWeights(data[weightStart:weightEnd]); err != nil {
		return err
	}

	if err := savedGroupsFault(saved, ents, dead); err != nil {
		return err
	}
	if hasStructures {
		if err := validateSavedStructures(structures, savedStructures, savedCells); err != nil {
			return err
		}
		for i := range structures {
			structures[i].Blocking = savedStructures[i].Blocking
		}
	}
	motionCheck := World{entities: ents, savedMotion: motion, bounds: b, grid: grid, savedCellPlanes: planes, cellTails: cellTails}
	if err := motionCheck.savedMotionFault(); err != nil {
		return err
	}
	if err := motionCheck.savedCellPlaneStateFault(); err != nil {
		return err
	}
	candidate := World{
		entityIDFloor:       entityIDFloor,
		savedWorldEffects:   worldEffects,
		structureUses:       structureUses,
		scorchedCells:       scorched,
		savedObjects:        weighted.savedObjects,
		savedCellPlanes:     planes,
		savedMotion:         motion,
		hasSavedStructures:  hasStructures,
		savedStructures:     savedStructures,
		savedStructureCells: savedCells,
		savedGroups:         saved,
		hasSessionClock:     hasClock,
		fullTick:            fullTick,
		tick:                binary.LittleEndian.Uint64(data[1:9]),
		rng:                 rng{state: binary.LittleEndian.Uint64(data[9:17])},
		bounds:              b,
		mode:                mode,
		grid:                grid,
		cost:                cost,
		height:              height,
		relations:           rel,
		entities:            ents,
		routes:              routes,
		groups:              groups,
		sacks:               sacks,
		spells:              spells,
		itemWeights:         itemWeights,
		// THE GHOST TEMPLATE IS THE ONE FIELD LEFT THAT THE FORM DOES NOT
		// CARRY; it is carried ACROSS the decode instead. It is install-
		// derived input, like the spell table, but resolved by the path that
		// rebuilds the world rather than stored in it. pkg/game's resume
		// unmarshals INTO the world StartMissionFrom already built
		// (resume.go), so the template the definition table gave that world
		// is the one a resumed mission raises from. A receiver that never had
		// one — a decode into a fresh World, which is what the read-only save
		// instruments do — keeps the zero template and refuses the raise at
		// admission. The composite literal's right-hand side is evaluated
		// before the assignment, so this reads the receiver's own value and
		// not the one being built.
		//
		// rawSessionHead, rawSessionMid, savedCellRecords, savedSpellEffects,
		// savedProjectiles and savedDiaries used to be carried the same way, with
		// no rebuild rule of their own.
		ghost:             w.ghost,
		sourceDerive:      w.sourceDerive,
		burst:             burstState{phases: w.burst.phases},
		rules:             w.rules,
		diary:             w.diary,
		rawSessionHead:    carriedResume.head,
		rawSessionMid:     carriedResume.mid,
		savedCellRecords:  carriedResume.cellRecords,
		savedSpellEffects: carriedResume.spellEffects,
		savedProjectiles:  carriedResume.projectiles,
		savedDiaries:      carriedResume.diaries,
		carried:           carried,
		equipment:         equipment,
		purses:            purses,
		script:            st.script,
		registers:         st.registers,
		latches:           st.latches,
		won:               st.won,
		lost:              st.lost,
		outcome:           st.outcome,
		casts:             casts,
		bookCasts:         bookCasts,
		scrollCasts:       scrolls,
		effects:           effects,
		attached:          attached,

		formations:   formations,
		cellTails:    cellTails,
		structures:   structures,
		originalDead: dead,
	}
	candidate.rebuildStructureSlots()
	if err := candidate.pendingAttackFault(); err != nil {
		return err
	}
	for _, meta := range structureMetadata {
		i := indexOfStructure(candidate.structures, meta.ID)
		if i < 0 {
			return fmt.Errorf("sim: structure-use metadata has absent structure %d", meta.ID)
		}
		candidate.structures[i].Kind = meta.Kind
		candidate.structures[i].UseAmount = meta.Amount
	}
	for _, u := range structureUses {
		i, si := indexOfEntity(ents, u.Entity), indexOfStructure(candidate.structures, u.Structure)
		if i < 0 || si < 0 || !ents[i].Alive() || ents[i].OffMap || !candidate.structures[si].Usable() {
			return fmt.Errorf("sim: invalid structure-use actor/target %d/%d", u.Entity, u.Structure)
		}
	}
	for _, key := range scorched {
		x, y := keyCell(key)
		if x >= b.Width || y >= b.Height {
			return fmt.Errorf("sim: scorched cell %04x is outside world bounds", key)
		}
	}
	if err := candidate.applyAttackNotices(notices); err != nil {
		return err
	}
	candidate.normaliseTargetReferences()
	if err := candidate.validateSavedObjects(); err != nil {
		return err
	}
	if err := candidate.savedWorldEffectsFault(); err != nil {
		return err
	}
	if err := candidate.areaOwnershipFault(); err != nil {
		return err
	}
	if err := candidate.applySpellDeliveryState(deliveryState); err != nil {
		return err
	}
	*w = candidate
	// The registers are taken from the form and the build-time constants are NOT
	// re-applied over them, which is the order a mission is resumed in: compile
	// the script, then lay the stored volatile half on top. Presetting here would
	// undo every variable the mission had set.
	return nil
}

// decodeRoutes reads one route per entity out of the form's last section and
// checks each against the world the rest of the form describes: the bounds and
// the entity it belongs to.
//
// It is checked against the DECODED world rather than against the receiver,
// which is what makes a form self-contained — a route is legal or not by what
// its own form says, never by what the world it is being read into happened to
// hold. The four refusals are the states a tick cannot produce, so a form
// carrying one is not a world this package could have written. Terrain is not
// among them: a route may cross a cell closed after the route was stored.
//
// It returns how many bytes it consumed and the SCRIPT SECTION consumes the
// rest, exactly. That is still the other half of the declared entity count being
// checked rather than trusted — a count one too small leaves this reader
// starting inside the last record, where it reads a cell count out of an id and
// either runs past the end or leaves a remainder the script section then
// refuses.
func decodeRoutes(data []byte, b Bounds, ents []Entity) ([][]cell, int, error) {
	routes := make([][]cell, len(ents))
	off := 0
	for i := range ents {
		if off+routeCountLen > len(data) {
			return nil, 0, fmt.Errorf("sim: byte form truncated: entity %d's route count needs %d byte(s), %d left",
				i, routeCountLen, len(data)-off)
		}
		count := binary.LittleEndian.Uint32(data[off : off+routeCountLen])
		off += routeCountLen
		// Multiplied in int64 and compared against what is left before a single
		// cell is allocated, so a declared count cannot ask for memory the form
		// does not carry the bytes for.
		if need := int64(count) * cellLen; need > int64(len(data)-off) {
			return nil, 0, fmt.Errorf("sim: byte form truncated: entity %d's route declares %d cell(s), "+
				"which is %d byte(s), and %d are left", i, count, need, len(data)-off)
		}
		if count != 0 && !ents[i].HasTarget {
			return nil, 0, fmt.Errorf("sim: entity record %d: a route of %d cell(s) on an entity with no target",
				i, count)
		}

		r := make([]cell, count)
		for k := range r {
			c := cell{
				x: int32(binary.LittleEndian.Uint32(data[off : off+4])),
				y: int32(binary.LittleEndian.Uint32(data[off+4 : off+8])),
			}
			off += cellLen
			_, in := cellIndexIn(b, c.x, c.y)
			if !in {
				return nil, 0, fmt.Errorf("sim: entity record %d: route cell %d is (%d,%d), outside %dx%d",
					i, k, c.x, c.y, b.Width, b.Height)
			}
			// A route cell is NOT tested against terrain: a wall that lands
			// across a stored route leaves the route standing, and the mover
			// meets the closed cell on its way, so a form holding one is a state
			// a tick produces.
			if k > 0 && !neighbours(r[k-1], c) {
				return nil, 0, fmt.Errorf("sim: entity record %d: route cells %d and %d are (%d,%d) and (%d,%d), "+
					"which are not neighbours", i, k-1, k, r[k-1].x, r[k-1].y, c.x, c.y)
			}
			r[k] = c
		}
		if count != 0 {
			if last := r[count-1]; last.x != ents[i].TargetX || last.y != ents[i].TargetY {
				return nil, 0, fmt.Errorf("sim: entity record %d: its route ends at (%d,%d) and its target is (%d,%d)",
					i, last.x, last.y, ents[i].TargetX, ents[i].TargetY)
			}
		}
		routes[i] = r
	}
	return routes, off, nil
}

// cellIndexIn is cellIndex over bounds that are not a world's yet: the decode
// checks a route before the world it belongs to exists. It is the same
// arithmetic and the same row-major order, taken from the bounds the form
// declares.
func cellIndexIn(b Bounds, x, y int32) (int, bool) {
	if x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return 0, false
	}
	return int(y)*int(b.Width) + int(x), true
}

// neighbours reports whether b is one of a's eight neighbours: a Chebyshev step
// of exactly one. Equal cells are not neighbours, so a route that stands still
// is refused by the same test that refuses one that jumps.
func neighbours(a, b cell) bool {
	dx, dy := a.x-b.x, a.y-b.y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx <= 1 && dy <= 1 && (dx != 0 || dy != 0)
}

// validGroupOrder reports whether o is one of the seven orders a decode may
// carry, including0 and Roam, held to one test by the decoder and by
// whatever else in this package needs to ask.
//
// Order 0 WAS refused here: it was what a group carried before anything
// installed one, reachable only through groupState's own missing-record
// answer and never through a byte a form named, so a form naming it could
// only be lying about a group this package never built that way.
func validGroupOrder(o uint8) bool {
	return o == 0 || o == orderGuard || o == orderSwarm || o == orderStandGround || o == orderMove || o == orderSwarm2 || o == orderRoam
}

// decodeGroups reads the group section out of the form's tail: a count, then
// that many (owner, group, base, order, commandedX, commandedY) records, and
// returns how many bytes it consumed so the script section can be found past
// them exactly as it is past the routes.
//
// THE COMMANDED CELL IS CARRIED WHOLE: no coordinate is refused, folded or
// clamped, on the same rule the entity's own X and Y already take — every
// int32 pair is a state a group command can leave, so refusing one here
// would make a world this package can produce a world it cannot read back.
func decodeGroups(data []byte) ([]groupAI, int, error) {
	if len(data) < groupCountLen {
		return nil, 0, fmt.Errorf("sim: byte form truncated: the group section's count needs %d byte(s), %d left",
			groupCountLen, len(data))
	}
	declared := binary.LittleEndian.Uint32(data[:groupCountLen])
	span := int64(declared) * groupRecordLen
	if avail := int64(len(data) - groupCountLen); span > avail {
		return nil, 0, fmt.Errorf("sim: byte form declares %d group record(s), whose records are %d byte(s), "+
			"and carries %d byte(s) after the count", declared, span, avail)
	}

	groups := make([]groupAI, declared)
	off := groupCountLen
	for i := range groups {
		o := off + groupRecordLen*i
		g := groupAI{
			owner:      binary.LittleEndian.Uint32(data[o : o+4]),
			group:      binary.LittleEndian.Uint32(data[o+4 : o+8]),
			base:       data[o+8],
			order:      data[o+9],
			commandedX: int32(binary.LittleEndian.Uint32(data[o+10 : o+14])),
			commandedY: int32(binary.LittleEndian.Uint32(data[o+14 : o+18])),
		}
		if !validGroupOrder(g.order) {
			return nil, 0, fmt.Errorf("sim: group record %d is (%d,%d): order %d is not one of the six this build writes",
				i, g.owner, g.group, g.order)
		}
		if i > 0 && !(groups[i-1].owner < g.owner ||
			(groups[i-1].owner == g.owner && groups[i-1].group < g.group)) {
			return nil, 0, fmt.Errorf("sim: group record %d is (%d,%d), which does not ascend past (%d,%d)",
				i, g.owner, g.group, groups[i-1].owner, groups[i-1].group)
		}
		groups[i] = g
	}
	used := off + groupRecordLen*int(declared)
	return groups, used, nil
}

// decodeSacks reads the sack section out of the form's tail: a count, then
// that many sacks — X, Y, gold, an item count and that many item codes — and
// returns how many bytes it consumed so the script section can be found past
// them exactly as it is past the groups (0103).
//
// The declared count is bounded against the buffer by the narrowest a sack
// can be — its sackHeaderLen head alone, naming no item — before a single
// sack is allocated, on decodeGroups' own ground: a declared count cannot
// ask for memory the form does not carry the bytes for. Each sack's own
// item count is bounded the same way once its head is read.
func decodeSacks(b Bounds, data []byte) ([]Sack, int, error) {
	if len(data) < sackCountLen {
		return nil, 0, fmt.Errorf("sim: byte form truncated: the sack section's count needs %d byte(s), %d left",
			sackCountLen, len(data))
	}
	declared := binary.LittleEndian.Uint32(data[:sackCountLen])
	if span := int64(declared) * sackHeaderLen; span > int64(len(data)-sackCountLen) {
		return nil, 0, fmt.Errorf("sim: byte form declares %d sack(s), whose headers alone are %d byte(s), "+
			"and carries %d byte(s) after the count", declared, span, len(data)-sackCountLen)
	}

	sacks := make([]Sack, declared)
	off := sackCountLen
	for i := range sacks {
		// The upfront span check above bounds ONLY the allocation, at every
		// sack's cheapest possible width — it says nothing about where off
		// stands once earlier sacks have each spent more than that on their
		// own items, which is why this reader still checks before every
		// head it reads, on decodeRoutes' own per-unit rule.
		if off+sackHeaderLen > len(data) {
			return nil, 0, fmt.Errorf("sim: byte form truncated: sack %d's head needs %d byte(s), %d left",
				i, sackHeaderLen, len(data)-off)
		}
		s := Sack{
			X:    int32(binary.LittleEndian.Uint32(data[off : off+4])),
			Y:    int32(binary.LittleEndian.Uint32(data[off+4 : off+8])),
			Gold: binary.LittleEndian.Uint32(data[off+8 : off+12]),
		}
		items := binary.LittleEndian.Uint32(data[off+12 : off+16])
		off += sackHeaderLen
		if need := int64(items) * 2; need > int64(len(data)-off) {
			return nil, 0, fmt.Errorf("sim: byte form truncated: sack %d declares %d item(s), which is %d byte(s), and %d are left", i, items, need, len(data)-off)
		}
		s.Items = make([]uint16, items)
		for k := range s.Items {
			s.Items[k] = binary.LittleEndian.Uint16(data[off : off+2])
			off += 2
		}
		s.ItemInstances = rawPlainItems(s.Items)
		if err := sackFault(b, s); err != nil {
			return nil, 0, fmt.Errorf("sim: sack %d: %w", i, err)
		}
		if i > 0 && !(sacks[i-1].Y < s.Y || (sacks[i-1].Y == s.Y && sacks[i-1].X < s.X)) {
			return nil, 0, fmt.Errorf("sim: sack %d is (%d,%d), which does not ascend past (%d,%d)",
				i, s.X, s.Y, sacks[i-1].X, sacks[i-1].Y)
		}
		sacks[i] = s
	}
	return sacks, off, nil
}

// decodeCarried reads the carry section out of the form's tail: one record
// per entity, in entity order, each a uint32 code count then that many
// little-endian uint16 codes — the route section's own rule, so it
// declares no count of its own that could disagree with the entity count the
// header already declares, and it is bounds-checked before every read
// exactly as decodeRoutes checks before every route: the upfront span check
// a declared count gets there is repeated here for the same reason, rather
// than trusted once for the whole section.
//
// It reads EXACTLY len(ents) records — the count decodeRoutes' own caller
// already parsed and checked — so a section short one record is a length
// error at the record decodeCarried cannot start, and one long by any
// amount is left for the SCRIPT SECTION's own exact-consumption check to
// catch.
//
// Preserve the flat order until decodeItemState compares this projection
// against the complete instances. Equal codes can describe different prices,
// effects, kinds or weights. Folding their plain-code placeholders here can
// reorder A,B,A into A,A,B and reject an otherwise valid native SAVE.
// The later typed section restores stack quantities, and decodeInstanceWeights
// validates canonical stacks after their full identity has been decoded.
func decodeCarried(data []byte, ents []Entity) ([][]ItemStack, int, error) {
	carried := make([][]ItemStack, len(ents))
	off := 0
	for i := range ents {
		if off+carryCountLen > len(data) {
			return nil, 0, fmt.Errorf("sim: byte form truncated: entity %d's carried code count needs %d byte(s), %d left",
				i, carryCountLen, len(data)-off)
		}
		count := binary.LittleEndian.Uint32(data[off : off+carryCountLen])
		off += carryCountLen
		if need := int64(count) * 2; need > int64(len(data)-off) {
			return nil, 0, fmt.Errorf("sim: byte form truncated: entity %d's carried section declares %d code(s), which is %d byte(s), and %d are left", i, count, need, len(data)-off)
		}
		codes := make([]uint16, count)
		for k := range codes {
			codes[k] = binary.LittleEndian.Uint16(data[off : off+2])
			off += 2
			if codes[k] == 0 {
				return nil, 0, fmt.Errorf("sim: entity record %d: carried item %d has code zero, which is not an item", i, k)
			}
		}
		carried[i] = appendUnits(nil, codes)
	}
	return carried, off, nil
}

// decodeEquipment reads the equipment section out of the form's tail: one
// fixed-width record per entity, in entity order, EquipSlots little-endian
// uint16 codes each, in slot order — decodeCarried's own placement rule,
// but decodePurses' own shape for the check: there is no count anywhere in
// the section, per-record or overall, because every one of the twelve slots
// is written whether it is empty or not, so it is bounds-checked once, at
// its own fixed width, before a single code is read — the purse section's
// own upfront span check, restated for a width that depends on len(ents)
// rather than on relationSlots.
//
// ZERO IS NOT REFUSED, and that is the one respect in which this reader is
// simpler than decodeCarried rather than merely different from it: a
// carried code of zero is refused because a container can never
// legitimately hold one (carryFault), but an item code's own class field
// never resolves to a real item at row 0 either, so an equipment slot can
// never legitimately hold anything BUT zero at rest. A decoded zero is
// therefore exactly the claim this package can have written, on
// carryFault's own doc stated the other way round, and there is no fault
// function here for the same reason reachFault and regenFault exist for
// their own fields: this one would have nothing to refuse.
func decodeEquipment(data []byte, ents []Entity) ([][EquipSlots]ItemInstance, int, error) {
	need := int64(len(ents)) * int64(equipRecordLen)
	if int64(len(data)) < need {
		return nil, 0, fmt.Errorf("sim: byte form truncated: the equipment section needs %d byte(s), %d left",
			need, len(data))
	}
	equipment := make([][EquipSlots]ItemInstance, len(ents))
	off := 0
	for i := range equipment {
		for k := 0; k < EquipSlots; k++ {
			equipment[i][k] = PlainItem(binary.LittleEndian.Uint16(data[off : off+2]))
			off += 2
		}
	}
	return equipment, off, nil
}

// decodeTreasure reads version 44's fixed-width death-gold record for every
// entity and writes the four values onto the records already decoded above.
func decodeTreasure(data []byte, ents []Entity) (int, error) {
	need := int64(len(ents)) * treasureRecordLen
	if int64(len(data)) < need {
		return 0, fmt.Errorf("sim: byte form truncated: the death-gold section needs %d byte(s), %d left",
			need, len(data))
	}
	for i := range ents {
		o := i * treasureRecordLen
		ents[i].TypeID = int32(binary.LittleEndian.Uint32(data[o : o+4]))
		ents[i].GoldChance = int32(binary.LittleEndian.Uint32(data[o+4 : o+8]))
		ents[i].TreasureMin = int32(binary.LittleEndian.Uint32(data[o+8 : o+12]))
		ents[i].TreasureMax = int32(binary.LittleEndian.Uint32(data[o+12 : o+16]))
	}
	return int(need), nil
}

// decodePurses reads the purse section out of the form's tail: a fixed
// relationSlots little-endian uint32s, one per roster slot in slot order,
// with no length of its own — the relation block's own rule, restated for
// a section that does not close the form. It is bounds-checked once, at its
// own fixed width, before a single slot is read: a block that declares no
// count of its own still has to be checked against what the buffer actually
// holds, on the same ground every other section's upfront span check is made
// from.
func decodePurses(data []byte) ([relationSlots]uint32, int, error) {
	var purses [relationSlots]uint32
	if len(data) < purseLen {
		return purses, 0, fmt.Errorf("sim: byte form truncated: the purse section needs %d byte(s), %d left",
			purseLen, len(data))
	}
	for i := range purses {
		purses[i] = binary.LittleEndian.Uint32(data[4*i : 4*i+4])
	}
	return purses, purseLen, nil
}

// decodeSpells reads the SPELL TABLE out of the form's tail: a uint16 count,
// then that many 17-byte records — id, mana cost, damageMin, damageMax,
// school, maxRange, flags — right after the purse section and before the
// script section (version 36). It returns how many bytes it consumed so the
// script section can be found past it exactly as it is past every other
// counted section.
//
// The declared count is bounded against the buffer at spellRecordLen per
// record, before a single record is allocated, on decodeGroups' own ground:
// a declared count cannot ask for memory the form does not carry the bytes
// for.
//
// EACH RECORD IS DECODED IN THE FORM'S OWN ORDER, never sorted, on the
// encoder's own rule: the table is a caller's book read out in the order it
// was built, and reordering it here would take that order away a second time
// — once from a decode that never restores it, on top of the encode that
// never scrambled it.
//
// THE FLAGS BYTE IS CHECKED FIRST, against the two bits this build defines,
// for the target-presence byte's own reason: a byte read with an undefined
// bit folded away would map two byte forms onto one world. What is left —
// an id of 0, a repeated id, a negative mana cost, a negative damage column
// — is exactly normaliseSpells' own four refusals, so decoding calls it
// rather than restating it: the same check the constructor makes is called
// here to refuse instead, on regenFault's and reachFault's own precedent
// for every other field this decoder shares a rule with its constructor
// over.
func decodeSpells(data []byte) ([]SpellRule, int, error) {
	if len(data) < spellCountLen {
		return nil, 0, fmt.Errorf("sim: byte form truncated: the spell table's count needs %d byte(s), %d left",
			spellCountLen, len(data))
	}
	declared := binary.LittleEndian.Uint16(data[:spellCountLen])
	span := int64(declared) * spellRecordLen
	if avail := int64(len(data) - spellCountLen); span > avail {
		return nil, 0, fmt.Errorf("sim: byte form declares %d spell(s), whose records are %d byte(s), "+
			"and carries %d byte(s) after the count", declared, span, avail)
	}

	spells := make([]SpellRule, declared)
	off := spellCountLen
	for i := range spells {
		o := off + spellRecordLen*i
		sp := SpellRule{
			ID:        binary.LittleEndian.Uint16(data[o : o+2]),
			ManaCost:  int32(binary.LittleEndian.Uint32(data[o+2 : o+6])),
			DamageMin: int32(binary.LittleEndian.Uint32(data[o+6 : o+10])),
			DamageMax: int32(binary.LittleEndian.Uint32(data[o+10 : o+14])),
			School:    data[o+14],
			MaxRange:  data[o+15],
		}
		flags := data[o+16]
		if flags&spellFlagRowSpecific != 0 && flags&(spellFlagRestorative|spellFlagDamaging) == 0 ||
			flags&(spellFlagAreaHitsLo|spellFlagAreaHitsHi) != 0 && flags&spellFlagArea == 0 && !raysFlag(flags, sp.ID) ||
			flags&(spellFlagAreaHitsLo|spellFlagAreaHitsHi) == spellFlagAreaHitsLo|spellFlagAreaHitsHi {
			return nil, 0, fmt.Errorf("sim: spell record %d: flags byte is %#02x, which sets an undefined bit",
				i, flags)
		}
		if flags&spellFlagArea != 0 {
			sp.AreaHits = AreaHits(flags >> 6)
		}
		if flags&spellFlagRowSpecific != 0 {
			sp.HealHostile = flags&spellFlagRestorative != 0
			sp.SelfCast = flags&spellFlagRestorative == 0
		}
		sp.TargetsUnit = flags&spellFlagTargetsUnit != 0
		sp.Damaging = flags&spellFlagDamaging != 0
		sp.Restorative = flags&spellFlagRestorative != 0
		sp.Area = flags&spellFlagArea != 0
		sp.Defensive = flags&spellFlagDefensive != 0
		sp.AreaDuration = int32(binary.LittleEndian.Uint32(data[o+17 : o+21]))
		sp.Distribution = data[o+21]
		sp.Radius = data[o+22]
		if raysFlag(flags, sp.ID) {
			if sp.Radius == 0 {
				return nil, 0, fmt.Errorf("sim: spell record %d: ray cap flag with a cap of 0", i)
			}
			sp.Rays, sp.Radius = sp.Radius, 0
		}
		sp.SpellDuration = int32(binary.LittleEndian.Uint32(data[o+23 : o+27]))
		sp.EffectKind = EffectKind(data[o+27])
		sp.EffectMode = EffectMode(data[o+28])
		sp.EffectMagnitude = int32(binary.LittleEndian.Uint32(data[o+29 : o+33]))
		sp.EffectDuration = binary.LittleEndian.Uint16(data[o+33 : o+35])
		sp.Complication = data[o+35]
		// Damaging and Restorative are the exact complement of one another
		// over the one row that carries damage columns and heals with them
		// (SpellRule's own doc, spell.go), so a form setting both describes
		// a row no loader can build.
		if sp.Damaging && sp.Restorative {
			return nil, 0, fmt.Errorf("sim: spell record %d: flags byte is %#02x, which is both damaging and restorative",
				i, flags)
		}
		spells[i] = sp
	}
	used := off + spellRecordLen*int(declared)

	norm, err := normaliseSpells(spells)
	if err != nil {
		return nil, 0, fmt.Errorf("sim: %w", err)
	}
	return norm, used, nil
}

// decodeItemWeights reads the item-weight section — the table every
// actor's load is derived against (version 56) — and answers it with how
// many bytes it consumed.
//
// It refuses THREE shapes, and each is a table this package cannot have
// written: a declared count whose records do not fit in what is left, a code
// of zero (never an item, carryFault's own rule), and a pair of records out of
// ascending code order. The order test also covers a duplicate, because equal
// codes are not ascending — so one comparison carries both the sort the
// encoder writes and the uniqueness normaliseItemWeights guarantees, and the
// decoded table is directly searchable by itemWeightOf without a second pass.
//
// A WEIGHT IS ACCEPTED WHOLE, including a negative one: the shipped Weapons
// collection carries a row whose price and weight columns are both -1, and the
// original's own field is a signed word (ITEM-STACK-003). There is no value to
// fold a weight onto.
func decodeItemWeights(data []byte) ([]ItemWeight, int, error) {
	if len(data) < itemWeightCountLen {
		return nil, 0, fmt.Errorf("sim: byte form truncated: the item-weight table's count needs %d byte(s), %d left",
			itemWeightCountLen, len(data))
	}
	declared := binary.LittleEndian.Uint16(data[:itemWeightCountLen])
	span := int64(declared) * itemWeightRecordLen
	if avail := int64(len(data) - itemWeightCountLen); span > avail {
		return nil, 0, fmt.Errorf("sim: byte form declares %d item weight(s), whose records are %d byte(s), "+
			"and carries %d byte(s) after the count", declared, span, avail)
	}
	if declared == 0 {
		return nil, itemWeightCountLen, nil
	}
	out := make([]ItemWeight, declared)
	off := itemWeightCountLen
	for i := range out {
		o := off + itemWeightRecordLen*i
		out[i] = ItemWeight{
			Code:   binary.LittleEndian.Uint16(data[o : o+2]),
			Weight: int32(binary.LittleEndian.Uint32(data[o+2 : o+6])),
		}
		if out[i].Code == 0 {
			return nil, 0, fmt.Errorf("sim: item-weight record %d names code 0, which is not an item", i)
		}
		if i > 0 && out[i].Code <= out[i-1].Code {
			return nil, 0, fmt.Errorf("sim: item-weight record %d names code %d, which is not above record %d's %d",
				i, out[i].Code, i-1, out[i-1].Code)
		}
	}
	return out, off + int(span), nil
}
