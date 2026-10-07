package mapload

// Saved is the state an ORIGINAL-GAME save file recorded for one party member:
// where he stood and the pools he stood there with.
//
// IT IS NOT Carry AND IT IS NOT PartyMember's OWN FIELDS, and the difference is
// which arithmetic runs. Every other field of a member is an INPUT the mint
// folds — four statistics and a weapon become eight combat numbers, a step
// rate, a sight radius and two pool maxima. A Saved is the opposite: it is a
// number another program wrote, and the mint must put it in rather than derive
// it. A restored character's four statistics DO go on Hero and DO fold, because
// they are the same four numbers character generation produces; his health is
// not a statistic and no fold can produce the value his file holds.
//
// IT IS REACHED BY POINTER on Carry's own reason: the zero value has to mean
// "this member came from no save at all", and a saved character standing at
// cell (0,0) with no mana is a different state from a member the drop cell
// places. Nil is every party this tree builds that did not come out of an
// original save, and a nil Saved changes nothing at all — the mint takes exactly
// the arm it always took.
//
// WHAT IS NOT HERE IS NOT AN OMISSION. The file's own capacity word, its two
// Unknown statistic words, its spellbook and its journal are read by
// pkg/formats/sav and deliberately not carried into a world: capacity has no
// field in this tree, the two Unknown words are Unknown, and what a saved
// Spell record or a journal array MEANS is Unknown. A value applied here
// reaches the world hash, so an Unknown one applied here would be an unverified
// fact in hashed simulation state.
type Saved struct {
	// Cell is where the save left him. IT REPLACES THE DROP WALK ENTIRELY for
	// this member: the map's authorised start cell is where a fresh party
	// begins, and a resumed character does not begin.
	Cell Cell

	// HP and MaxHP are the health pair, and Mana and MaxMana the mana pair, as
	// the file holds them. Both are the file's values and not the fold's: a
	// resumed character wounded to a third of his health arrives wounded.
	HP, MaxHP     int32
	Mana, MaxMana int32

	// HealthRegenPeriod and ManaRegenPeriod are the two periods the file
	// holds. They are the base constructor's own 100 and 50 on every record
	// measured, which is the same pair data.UnitDefaults() answers, so
	// carrying them changes no world today — they are carried because the
	// file states them and a later save need not.
	HealthRegenPeriod, ManaRegenPeriod int32

	// MapUnitID is the map record's own identifier word this character was
	// placed from, and zero for a character the map never placed.
	//
	// IT IS HERE SO THE SCRIPT CAN STILL NAME HIM. A character the player took
	// into his own group keeps that id, and the resume WITHDRAWS the map's
	// record so one person does not become two entities. A map's type-7 script
	// names a placed unit by exactly this word, so without this the reference
	// would resolve to no entity at all — and an unresolved check writes no
	// register, leaving a zero that a proximity comparison satisfies.
	// ScriptUnits reads it and binds the id to the entity this member became.
	MapUnitID uint16
}
