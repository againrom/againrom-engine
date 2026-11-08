package ui

import "image"

type PanelComposition = panelComposition
type PanelArt = panelArt
type PanelButton = panelButton

type RoomPanel struct {
	Composition               PanelComposition
	Art                       PanelArt
	Buttons                   []PanelButton
	Values                    []TownSurfaceButton
	TwoLines                  bool
	PressedOnly               []bool
	Style                     *RoomButtonStyle
	TrackPointer, HoverInside bool
}

var TownRoomPanel = panelComposition{Body: TownUpperRegion, Seam: townUpperSeamRegion,
	MissingBox: TownUpperRegion, Bare: true, Ink: plaqueCommandInk}
var TownRoomShopPanel = func() panelComposition { p := shopPanel; p.Bare = true; return p }()
var TownRoomEntryPanel = panelComposition{FitLine: true, Ink: roomButtonInk}

func (p RoomPanel) resolve(pointer RoomPointer) buttonPanel {
	c := p.Composition
	c.Ink = p.Style.plaqueInk(c.Ink)
	p.Style.shift(&c)
	buttons := append([]panelButton(nil), p.Buttons...)
	if p.TwoLines {
		buttons = make([]panelButton, min(len(p.Values), len(c.Plaques)))
		for i, value := range p.Values[:len(buttons)] {
			buttons[i] = panelButton{Captions: []string{value.Label, value.Value}, Disabled: !value.Enabled}
		}
	}
	if p.TrackPointer {
		for i := range buttons {
			b := &buttons[i]
			b.Hover = pointer.Hover && pointer.At.In(c.Plaques[i])
			b.Pressed = pointer.Press.Kind == TownSurfaceControlButton && pointer.Press.Index == i
			if i < len(p.PressedOnly) && p.PressedOnly[i] {
				b.Pressed = b.Pressed && !b.Disabled
			}
			b.Inside = true
			if p.HoverInside {
				b.Inside = b.Hover
			}
			b.Disabled = p.Style.disabled(!b.Disabled)
		}
	}
	return (RoomPanel{Composition: c, Art: p.Art, Buttons: buttons}).build()
}

// WholeRoomPair keeps only complete off/on pairs.
func WholeRoomPair(pair [2]image.Image) [2]image.Image { return wholePair(pair) }
