package alm_test

import (
	"testing"

	"againrom/pkg/formats/alm"
)

// TestSchemaKeepsTheRecordLayout pins every schema constant to the value the
// readers and the editor spelled before they shared it, record by record.
func TestSchemaKeepsTheRecordLayout(t *testing.T) {
	records := map[string][]struct {
		name      string
		got, want int
	}{
		"tile word": {
			{"TileIndexMask", alm.TileIndexMask, 0x03ff}, {"TileImpassable", alm.TileImpassable, 0x2000},
			{"TileGroupBits", alm.TileGroupBits, 0x1fff},
		},
		"file header": {
			{"FileHeaderSize", alm.FileHeaderSize, 20}, {"FileMagicAt", alm.FileMagicAt, 0x00},
			{"FileHeaderLength", alm.FileHeaderLength, 0x04}, {"FileDataSize", alm.FileDataSize, 0x08},
			{"FileRecordCount", alm.FileRecordCount, 0x0c}, {"FileFormatVersion", alm.FileFormatVersion, 0x10},
		},
		"record header": {
			{"RecordHeaderSize", alm.RecordHeaderSize, 20}, {"RecordTag", alm.RecordTag, 0x00},
			{"RecordHeaderLength", alm.RecordHeaderLength, 0x04}, {"RecordPayloadSize", alm.RecordPayloadSize, 0x08},
			{"RecordTypeID", alm.RecordTypeID, 0x0c}, {"RecordWord10", alm.RecordWord10, 0x10},
			{"TypeIDCount", alm.TypeIDCount, 10},
		},
		"type ids": {
			{"TypeMeta", alm.TypeMeta, 0}, {"TypeTiles", alm.TypeTiles, 1}, {"TypeAltitude", alm.TypeAltitude, 2},
			{"TypeOverlay", alm.TypeOverlay, 3}, {"TypeObjects", alm.TypeObjects, 4}, {"TypeGroups", alm.TypeGroups, 5},
			{"TypeUnits", alm.TypeUnits, 6}, {"TypeScript", alm.TypeScript, 7}, {"TypeLoot", alm.TypeLoot, 8},
			{"TypeTileMarkers", alm.TypeTileMarkers, 9},
		},
		"type 0": {
			{"MetaSize", alm.MetaSize, 632}, {"MetaWidth", alm.MetaWidth, 0x00}, {"MetaHeight", alm.MetaHeight, 0x04},
			{"MetaAngle", alm.MetaAngle, 0x08}, {"MetaWord0C", alm.MetaWord0C, 0x0c}, {"MetaWord10", alm.MetaWord10, 0x10},
			{"MetaWord14", alm.MetaWord14, 0x14}, {"MetaBitmask", alm.MetaBitmask, 0x18},
			{"MetaGroupCount", alm.MetaGroupCount, 0x1c}, {"MetaObjectCount", alm.MetaObjectCount, 0x20},
			{"MetaUnitCount", alm.MetaUnitCount, 0x24}, {"MetaWord28", alm.MetaWord28, 0x28},
			{"MetaLootCount", alm.MetaLootCount, 0x2c}, {"MetaName", alm.MetaName, 0x30}, {"MetaNameLen", alm.MetaNameLen, 64},
			{"MetaWord70", alm.MetaWord70, 0x70}, {"MetaWord74", alm.MetaWord74, 0x74}, {"MetaDesc", alm.MetaDesc, 0x78},
			{"MetaDescLen", alm.MetaDescLen, 64}, {"MetaDescBlockLen", alm.MetaDescBlockLen, 512},
			{"MetaSlots", alm.MetaSlots, 0xb8}, {"MetaSlotsLen", alm.MetaSlotsLen, 448},
		},
		"type 4": {
			{"ObjectSize", alm.ObjectSize, 20}, {"ObjectExtSize", alm.ObjectExtSize, 8}, {"ObjectExtKind", alm.ObjectExtKind, 0x21},
			{"ObjectX", alm.ObjectX, 0x00}, {"ObjectY", alm.ObjectY, 0x04}, {"ObjectKind", alm.ObjectKind, 0x08},
			{"ObjectPlacement", alm.ObjectPlacement, 0x0c}, {"ObjectOwner", alm.ObjectOwner, 0x0e},
			{"ObjectIdentity", alm.ObjectIdentity, 0x12}, {"ObjectWidth", alm.ObjectWidth, 0x14},
			{"ObjectHeight", alm.ObjectHeight, 0x18},
		},
		"type 5": {
			{"GroupSize", alm.GroupSize, 76}, {"GroupColour", alm.GroupColour, 0x00}, {"GroupControl", alm.GroupControl, 0x04},
			{"GroupScalar", alm.GroupScalar, 0x08}, {"GroupName", alm.GroupName, 0x0c}, {"GroupNameLen", alm.GroupNameLen, 32},
			{"GroupRelations", alm.GroupRelations, 0x2c}, {"GroupRelationCount", alm.GroupRelationCount, 16},
		},
		"type 6": {
			{"UnitSize", alm.UnitSize, 70}, {"UnitX", alm.UnitX, 0x00}, {"UnitY", alm.UnitY, 0x04},
			{"UnitClass", alm.UnitClass, 0x08}, {"UnitSubID", alm.UnitSubID, 0x0a}, {"UnitFlags", alm.UnitFlags, 0x0c},
			{"UnitDefinition", alm.UnitDefinition, 0x10}, {"UnitOwner", alm.UnitOwner, 0x14}, {"UnitLoot", alm.UnitLoot, 0x18},
			{"UnitHP", alm.UnitHP, 0x20}, {"UnitMana", alm.UnitMana, 0x24}, {"UnitBody", alm.UnitBody, 0x2c},
			{"UnitMind", alm.UnitMind, 0x2d}, {"UnitSpirit", alm.UnitSpirit, 0x2e}, {"UnitReaction", alm.UnitReaction, 0x2f},
			{"UnitDefence", alm.UnitDefence, 0x31}, {"UnitAbsorption", alm.UnitAbsorption, 0x32},
			{"UnitSkills", alm.UnitSkills, 0x36}, {"UnitProtections", alm.UnitProtections, 0x3b},
			{"UnitID", alm.UnitID, 0x40}, {"UnitGroup", alm.UnitGroup, 0x42},
			{"UnitFlagNPC", alm.UnitFlagNPC, 0x01}, {"UnitFlagSex", alm.UnitFlagSex, 0x04},
			{"UnitFlagSkipGear", alm.UnitFlagSkipGear, 0x80}, {"UnitFaceMask", alm.UnitFaceMask, 0x7f},
		},
		"type 7": {
			{"CountWordSize", alm.CountWordSize, 4}, {"ScriptNodeSize", alm.ScriptNodeSize, 796},
			{"ScriptTriggerSize", alm.ScriptTriggerSize, 184}, {"ScriptLabelLen", alm.ScriptLabelLen, 64},
			{"ScriptNodeOpcode", alm.ScriptNodeOpcode, 0x40}, {"ScriptNodeID", alm.ScriptNodeID, 0x44},
			{"ScriptNodeValues", alm.ScriptNodeValues, 0x4c}, {"ScriptNodeTypes", alm.ScriptNodeTypes, 0x74},
			{"ScriptParams", alm.ScriptParams, 10}, {"ScriptTriggerPairs", alm.ScriptTriggerPairs, 0x80},
			{"ScriptTriggerActions", alm.ScriptTriggerActions, 0x98}, {"ScriptTriggerCompare", alm.ScriptTriggerCompare, 0xa8},
			{"ScriptTriggerOnce", alm.ScriptTriggerOnce, 0xb4}, {"ScriptPairs", alm.ScriptPairs, 3},
			{"ScriptActionRefs", alm.ScriptActionRefs, 4},
		},
		"type 8": {
			{"LootCount", alm.LootCount, 0x00}, {"LootOwner", alm.LootOwner, 0x04}, {"LootX", alm.LootX, 0x08},
			{"LootY", alm.LootY, 0x0c}, {"LootGold", alm.LootGold, 0x10}, {"LootHeadShort", alm.LootHeadShort, 16},
			{"LootHeadWide", alm.LootHeadWide, 20}, {"LootElementSize", alm.LootElementSize, 10},
			{"LootElementCode", alm.LootElementCode, 0x00}, {"LootElementField04", alm.LootElementField04, 0x04},
			{"LootElementMarker", alm.LootElementMarker, 0x06}, {"LootWideHeadVersion", alm.LootWideHeadVersion, 989},
		},
	}
	for record, rows := range records {
		for _, r := range rows {
			if r.got != r.want {
				t.Errorf("%s %s is %#x, want %#x", record, r.name, r.got, r.want)
			}
		}
	}
	if alm.ScriptTriggerPairs+8*alm.ScriptPairs != alm.ScriptTriggerActions ||
		alm.ScriptTriggerActions+4*alm.ScriptActionRefs != alm.ScriptTriggerCompare ||
		alm.ScriptNodeValues+4*alm.ScriptParams != alm.ScriptNodeTypes ||
		alm.UnitGroup+4 != alm.UnitSize || alm.GroupRelations+2*alm.GroupRelationCount != alm.GroupSize {
		t.Error("an array no longer ends where the next field begins")
	}
}
