package game

import (
	"errors"
	"fmt"
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/screenart"
	"againrom/pkg/town"
)

// roomPage answers the composer scene of a room the profile's room
// description gives a scene, building it on first use.
func (t *townScreen) roomPage(room string) *town.Scene {
	d := t.townDescription()
	if d == nil {
		return nil
	}
	return t.scene(d, room)
}

func (t *townScreen) tavernPage() *town.Scene { return t.roomPage("tavern") }

// TownClickSounds: the first game's tip Close and dialogue page presses
// request click01 and its tavern and school buttons request none (TOWN-547,
// TOWN-548, DIALOGUE-080); no second-game claim names these.
func (t *townScreen) TownClickSounds() bool { return t.hostRules().ClickSounds }

// roomSound requests what the room description's sound table binds to one
// named room event. Every town room interface sound goes through it.
func (t *townScreen) roomSound(room, on string) {
	if t == nil || t.sess == nil || t.townDescription() == nil {
		return
	}
	t.roomPage(room).Trigger(on)
}

func (t *townScreen) shopPage() *town.Scene { return t.roomPage("shop") }

// loadRoomSceneArt resolves a room scene's art as room description d names
// it: its pictures by entry name, and the problems of the entries that did
// not load joined into one error. A required entry that fails answers no
// pictures, and so does a nil description.
func loadRoomSceneArt(d *town.Description, room string, src terrain.EntrySource) (map[string][]image.Image, error) {
	if d == nil {
		return nil, nil
	}
	for i := range d.Rooms {
		r := &d.Rooms[i]
		if r.Name != room || r.Scene == nil {
			continue
		}
		art, err := town.LoadArt(r.Scene, screenArtOf(src))
		if err != nil {
			return nil, err
		}
		var problems []error
		for _, p := range art.Problems {
			problems = append(problems, errors.New(p))
		}
		return art.Frames, errors.Join(problems...)
	}
	return nil, nil
}

// roomPageArt resolves a room page's own art, outside its scene, as room
// description d names it. A nil description, or a room it does not name, is
// an error.
func roomPageArt(d *town.Description, room string, src terrain.EntrySource) (*screenart.Set, error) {
	if d != nil {
		for i := range d.Rooms {
			if r := &d.Rooms[i]; r.Name == room {
				return screenArtOf(src).Load(room, r.Art)
			}
		}
	}
	return nil, fmt.Errorf("room %s: no page art", room)
}

// roomEvents are the events the ROM1 campaign raises on its room pages: a
// shop rack picked by the player, the merchant's reactions to a trade, a
// school lesson and a change of the shown member's class.
var roomEvents = []string{"rack", "merchant-yes", "merchant-no", "train", "class-change"}

// roomValues answers the values the description names.
var roomValues = map[string]func(t *townScreen) int{
	"shop-chosen":  func(t *townScreen) int { return t.shopChosen },
	"member-class": (*townScreen).schoolMemberClass,
	"school-hover": func(t *townScreen) int { return t.schoolHover },
}

// roomScene is a page's composed centre as ui paints it.
type roomScene struct{ p *town.Scene }

func (s roomScene) Paint(dst *image.RGBA, group string) { s.p.Paint(dst, group) }

func (t *townScreen) inTavernInterior() bool {
	return t != nil && (t.room == roomTavern || t.room == roomTalk && t.dialogueBuilding == TownTavern)
}

// leaveTavernInterior pauses the tavern page and releases the tavern's room
// audio once the screen is elsewhere.
func (t *townScreen) leaveTavernInterior() {
	if t == nil {
		return
	}
	t.tavernPage().SetActive(false, t.inTavernInterior())
	if !t.inTavernInterior() && t.audioRoom == "tavern" {
		t.destroyRoomAudio()
	}
}

// TavernInteriorActive is an explicit Againrom pause policy for lifecycle
// arms the accepted research leaves open. A resume rebases all page clocks,
// so time spent unfocused, in a menu or outside the room cannot catch up.
func (t *townScreen) TavernInteriorActive(active bool) {
	if t == nil || t.sess == nil || t.townDescription() == nil || t.townDescription().Room("tavern") == nil {
		return
	}
	t.tavernPage().SetActive(active, t.inTavernInterior())
}

func (t *townScreen) inShopInterior() bool { return t != nil && t.AtTownShop() }

// ShopInteriorActive freezes the shop page's clocks while the shop is not the
// focused live room. The first activation keeps the entry stamps; later
// resumes rebase both clocks and cannot catch up time spent behind a menu,
// cutscene or focus loss.
func (t *townScreen) ShopInteriorActive(active bool) {
	if t != nil && t.hostRules().Pages == "entry" {
		if t.composed != nil && t.AtTownShop() {
			t.roomActive(active)
		}
		return
	}
	if t == nil || t.sess == nil {
		return
	}
	t.shopPage().SetActive(active, t.inShopInterior())
}

// AdvanceShopInteriorAnimation is one paint of the shop page.
func (t *townScreen) AdvanceShopInteriorAnimation() {
	if t == nil || !t.inShopInterior() {
		return
	}
	t.advanceShopSecondTip()
	if t.hostRules().Pages == "entry" {
		if t.composed != nil {
			t.advanceRoom()
		}
		return
	}
	t.shopPage().Advance()
}
