package game

import (
	"reflect"
	"testing"

	"againrom/pkg/base"
)

func chatTexts(g *chatAdapter) (commands, modifiers []string) {
	for _, row := range g.commands {
		commands = append(commands, row.text)
		for _, m := range row.modifiers {
			modifiers = append(modifiers, m.text)
		}
	}
	return commands, modifiers
}

// TestChatCommandTablesKeepTheirOrder pins both games' tables: the first
// game's thirteen texts in its chain order, and the second game's sixteen
// command texts in R2-ENGINE-295's test order with its four modifiers.
func TestChatCommandTablesKeepTheirOrder(t *testing.T) {
	first, firstMods := chatTexts(&firstChat)
	if want := []string{"#create ", "#modify ", "#summon ", "#killall", "#kill all", "#kill cheaters", "#kill ", "#pickup all", "#show map", "#hide map", "#victory", "#event ", "#Chicken"}; !reflect.DeepEqual(first, want) {
		t.Fatalf("first table %q", first)
	}
	second, secondMods := chatTexts(&secondChat)
	if want := []string{"#kick ", "#locate ", "#set latency ", "#show latency", "#create ", "#modify ", "#summon ", "#killall", "#kill all", "#kill cheaters", "#kill ", "#pickup all", "#show map", "#hide map", "#victory", "#event "}; !reflect.DeepEqual(second, want) {
		t.Fatalf("second table %q", second)
	}
	mods := []string{"+god", "+spell ", "+spells", "+knowledge"}
	if !reflect.DeepEqual(firstMods, mods) || !reflect.DeepEqual(secondMods, mods) {
		t.Fatalf("modifiers %q / %q", firstMods, secondMods)
	}
	for _, row := range secondChat.commands {
		if (row.gate == chatHost) != (row.effect == chatNoFacility) || row.gate == chatUnlocked {
			t.Fatalf("second row %q gate %d effect %d", row.text, row.gate, row.effect)
		}
	}
}

// TestChatAdapterFollowsTheCampaignService: each game's campaign service
// names its adapter, and only the destinations campaign admits commands.
func TestChatAdapterFollowsTheCampaignService(t *testing.T) {
	first, second := campaignOf(base.GameROM1), campaignOf(base.GameROM2)
	if first.chat() != &firstChat || second.chat() != &secondChat {
		t.Fatal("campaign service chat adapters")
	}
	if second.chatCampaign(nil) || second.chatCampaign(&Town{}) || !second.chatCampaign(&Town{second: newSecondCampaign()}) {
		t.Fatal("second-game campaign mode")
	}
	if secondChat.launchLine != secondUnlock || !secondChat.unlockOnLoad || firstChat.unlockOnLoad || firstChat.launchLine != "#Chicken" {
		t.Fatal("launch lines")
	}
}
