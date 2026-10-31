package alm

// The ROM1 record layout the readers and the map editor address: record type
// ids, field offsets relative to their record (the file header's to the file),
// record and field sizes, and the bits of the tile word and the unit flags.
const (
	TileIndexMask  = 0x03ff // tile word bits 0-9: the tile index
	TileImpassable = 0x2000 // tile word bit 13: the cell is impassable

	FileHeaderSize    = 20
	FileMagicAt       = 0x00
	FileHeaderLength  = 0x04
	FileDataSize      = 0x08
	FileRecordCount   = 0x0c
	FileFormatVersion = 0x10

	RecordHeaderSize   = 20
	RecordTag          = 0x00
	RecordHeaderLength = 0x04
	RecordPayloadSize  = 0x08
	RecordTypeID       = 0x0c
	RecordWord10       = 0x10

	// TypeIDCount is the size of the typeId index space: an id at or above it
	// selects no record case (ALM-REQ-055).
	TypeIDCount = 10

	TypeMeta     = 0
	TypeTiles    = 1
	TypeAltitude = 2
	TypeOverlay  = 3
	TypeObjects  = 4
	TypeGroups   = 5
	TypeUnits    = 6
	TypeScript   = 7
	TypeLoot     = 8

	TypeTileMarkers = 9

	MetaSize         = 632
	MetaWidth        = 0x00
	MetaHeight       = 0x04
	MetaAngle        = 0x08
	MetaWord0C       = 0x0c
	MetaWord10       = 0x10
	MetaWord14       = 0x14
	MetaBitmask      = 0x18
	MetaGroupCount   = 0x1c
	MetaObjectCount  = 0x20
	MetaUnitCount    = 0x24
	MetaWord28       = 0x28
	MetaLootCount    = 0x2c
	MetaName         = 0x30
	MetaNameLen      = 64
	MetaWord70       = 0x70
	MetaWord74       = 0x74
	MetaDesc         = 0x78
	MetaDescLen      = 64
	MetaDescBlockLen = 512
	MetaSlots        = 0xb8
	MetaSlotsLen     = 448

	ObjectSize      = 20
	ObjectExtSize   = 8
	ObjectExtKind   = 0x21 // an object of this kind carries the extension
	ObjectX         = 0x00
	ObjectY         = 0x04
	ObjectKind      = 0x08
	ObjectPlacement = 0x0c
	ObjectOwner     = 0x0e
	ObjectIdentity  = 0x12
	ObjectWidth     = 0x14 // in the extension
	ObjectHeight    = 0x18 // in the extension

	GroupSize          = 76
	GroupColour        = 0x00
	GroupControl       = 0x04
	GroupScalar        = 0x08
	GroupName          = 0x0c
	GroupNameLen       = 32
	GroupRelations     = 0x2c
	GroupRelationCount = 16

	UnitSize        = 70
	UnitX           = 0x00
	UnitY           = 0x04
	UnitClass       = 0x08
	UnitSubID       = 0x0a
	UnitFlags       = 0x0c
	UnitDefinition  = 0x10
	UnitOwner       = 0x14
	UnitLoot        = 0x18 // 1-based loot record index, 0 for none
	UnitHP          = 0x20
	UnitMana        = 0x24
	UnitBody        = 0x2c
	UnitMind        = 0x2d
	UnitSpirit      = 0x2e
	UnitReaction    = 0x2f
	UnitDefence     = 0x31
	UnitAbsorption  = 0x32
	UnitSkills      = 0x36
	UnitProtections = 0x3b
	UnitID          = 0x40
	UnitGroup       = 0x42

	UnitFlagNPC      = 0x01
	UnitFlagSex      = 0x04
	UnitFlagSkipGear = 0x80
	UnitFaceMask     = 0x7f // in the low byte of the sub id

	CountWordSize = 4

	ScriptNodeSize       = 796
	ScriptTriggerSize    = 184
	ScriptLabelLen       = 64
	ScriptNodeOpcode     = 0x40
	ScriptNodeID         = 0x44
	ScriptNodeValues     = 0x4c
	ScriptNodeTypes      = 0x74
	ScriptParams         = 10
	ScriptTriggerPairs   = 0x80
	ScriptTriggerActions = 0x98
	ScriptTriggerCompare = 0xa8
	ScriptTriggerOnce    = 0xb4
	ScriptPairs          = 3
	ScriptActionRefs     = 4

	LootCount           = 0x00
	LootOwner           = 0x04
	LootX               = 0x08
	LootY               = 0x0c
	LootGold            = 0x10
	LootHeadShort       = 16
	LootHeadWide        = 20
	LootElementSize     = 10
	LootElementCode     = 0x00
	LootElementField04  = 0x04
	LootElementMarker   = 0x06
	LootWideHeadVersion = 989 // from this format version a loot head carries gold
)
