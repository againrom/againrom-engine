package sim

// ActorDefinition is what one source states about a new actor: a resolved
// Units or Humans row, a generated party member, the raise template with its
// corpse stores, an authored scenario unit or a saved actor's row policy. The
// caller resolves the source, difficulty included; NewActor copies every field.
type ActorDefinition struct {
	Class, TypeID int32
	Humanoid      bool
	Domain        Domain

	HP, MaxHP                            int32
	Mana, MaxMana                        int32
	HealthRegenPeriod, ManaRegenPeriod   int32
	HealthRegeneration, ManaRegeneration int32

	Speed, RotationSpeed, Capacity int32
	SpeedModifier                  int32
	ScanRange, SeeInvisible        uint8
	Reach, TokenSize               uint8
	DyingTime, Withdraw, Wimpy     int32

	ToHit, Defence, Absorption int32
	DamageBase, DamageSpread   int32
	SecondBase, SecondSpread   uint8
	SecondaryDamage            SecondaryDamage
	AlwaysHits                 bool
	AttackCharge, AttackRelax  int32
	Protection                 [5]int32
	Resistance                 [5]uint8

	WeaponSpell       uint16
	WeaponSpellLevel  int32
	WeaponSpellSource WeaponSpellSource
	KnownSpells       uint32
	Book              Spellbook
	CreatureSpells    [CreatureSpellSlots]CreatureSpell
	AutoSpell         uint16

	XPValue, Reaction, Mind, Spirit      int32
	XPSlot                               uint8
	GainsXP                              bool
	GoldChance, TreasureMin, TreasureMax int32
	Skill, SkillXP                       [skillSlots]int32
	SuppressCorpseLoot                   bool

	NativeBasis    NativeActorBasis
	NativeClass    NativeClass
	NativeTraining NativeTraining
}

// ActorPlacement is where and for whom a new actor stands: its id, cell and
// facing, its roster slot and group, and the map unit id it answers to.
type ActorPlacement struct {
	ID        EntityID
	X, Y      int32
	Facing    uint8
	Owner     uint32
	Group     uint32
	MapUnitID uint16
}

// NewActor is the one constructor of an actor Entity. Every origin (map
// placement, party, siege hire, Control Spirit raise, scenario fixture and
// SAV seed) calls it; internal/archtest holds every other production Entity
// literal to a falling baseline. A facing starts with no turn in progress.
func NewActor(d ActorDefinition, at ActorPlacement) Entity {
	return Entity{
		ID: at.ID, X: at.X, Y: at.Y, Facing: at.Facing, DesiredFacing: at.Facing,
		Owner: at.Owner, Group: at.Group, MapUnitID: at.MapUnitID,

		Class: d.Class, TypeID: d.TypeID, Humanoid: d.Humanoid, Domain: d.Domain,
		HP: d.HP, MaxHP: d.MaxHP, Mana: d.Mana, MaxMana: d.MaxMana,
		HealthRegenPeriod: d.HealthRegenPeriod, ManaRegenPeriod: d.ManaRegenPeriod,
		HealthRegeneration: d.HealthRegeneration, ManaRegeneration: d.ManaRegeneration,
		Speed: d.Speed, SpeedModifier: d.SpeedModifier, RotationSpeed: d.RotationSpeed, Capacity: d.Capacity,
		ScanRange: d.ScanRange, SeeInvisible: d.SeeInvisible, Reach: d.Reach, TokenSize: d.TokenSize,
		DyingTime: d.DyingTime, Withdraw: d.Withdraw, Wimpy: d.Wimpy,
		ToHit: d.ToHit, Defence: d.Defence, Absorption: d.Absorption,
		DamageBase: d.DamageBase, DamageSpread: d.DamageSpread,
		SecondBase: d.SecondBase, SecondSpread: d.SecondSpread, SecondaryDamage: d.SecondaryDamage,
		AlwaysHits: d.AlwaysHits, AttackCharge: d.AttackCharge, AttackRelax: d.AttackRelax,
		Protection: d.Protection, Resistance: d.Resistance,
		WeaponSpell: d.WeaponSpell, WeaponSpellLevel: d.WeaponSpellLevel, WeaponSpellSource: d.WeaponSpellSource,
		KnownSpells: d.KnownSpells, Book: d.Book, CreatureSpells: d.CreatureSpells, AutoSpell: d.AutoSpell,
		XPValue: d.XPValue, Reaction: d.Reaction, Mind: d.Mind, Spirit: d.Spirit,
		XPSlot: d.XPSlot, GainsXP: d.GainsXP,
		GoldChance: d.GoldChance, TreasureMin: d.TreasureMin, TreasureMax: d.TreasureMax,
		Skill: d.Skill, SkillXP: d.SkillXP, SuppressCorpseLoot: d.SuppressCorpseLoot,
		NativeBasis: d.NativeBasis, NativeClass: d.NativeClass, NativeTraining: d.NativeTraining,
	}
}

// standAtPost is the state every actor enters a world in: guard at its own
// cell, with no retreat pending. The world constructor applies it to every
// initial actor and the Control Spirit raise to the actor it appends.
func (e *Entity) standAtPost() {
	e.ActorState = actorStateGuard
	e.Retreat = RetreatContinuation{}
	e.PostX, e.PostY = e.X, e.Y
}
