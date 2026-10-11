package game

import (
	"errors"
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/town"
)

// roomPage answers the composer scene of a room the profile's room
// description gives a scene, building it on first use.
func (t *townScreen) roomPage(room string) *town.Scene {
	if p := t.pages[room]; p != nil {
		return p
	}
	if t.pages == nil {
		t.pages = map[string]*town.Scene{}
	}
	p := town.NewScene(t.roomDescription(), room, townSceneHost{t: t, room: room}, t.townProcess)
	t.pages[room] = p
	return p
}

func (t *townScreen) tavernPage() *town.Scene { return t.roomPage("tavern") }

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
		art, err := town.LoadArt(r.Scene, townArtLoader{src})
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

// roomEvents are the events the ROM1 campaign raises on its room pages: a
// shop rack picked by the player, the merchant's reactions to a trade, a
// school lesson and a change of the shown member's class.
var roomEvents = []string{"rack", "merchant-yes", "merchant-no", "train", "class-change"}

// roomValues answers the values the description names.
var roomValues = map[string]func(t *townScreen) int{
	"shop-chosen":  func(t *townScreen) int { return t.shopChosen },
	"member-class": (*townScreen).schoolMemberClass,
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
	if !t.inTavernInterior() && t.audioRoom == roomTavern {
		t.destroyRoomAudio()
	}
}

// TavernInteriorActive is an explicit Againrom pause policy for lifecycle
// arms the accepted research leaves open. A resume rebases all page clocks,
// so time spent unfocused, in a menu or outside the room cannot catch up.
func (t *townScreen) TavernInteriorActive(active bool) {
	if t == nil || t.sess == nil {
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
	t.shopPage().Advance()
}
