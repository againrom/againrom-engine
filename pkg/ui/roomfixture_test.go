package ui

import "image"

func townSurfacePanel(v TownSurfaceView, _, _, _ bool) buttonPanel {
	return firstTavernPage(v).Page.Panel.resolve(RoomPointer{At: v.Hover, Hover: v.HasHover, Press: v.Press, Cell: v.HoverCell})
}

func finishTestRoomPage(p *RoomPage, v TownRoomView) {
	p.ViewInputs = true
	p.BeforePane, p.AfterPane = nil, nil
	p.Scene = v.Scene
	p.Font, p.Cells = v.Font, v.Cells
	school := v.Kind == TownSurfaceSchool && v.SchoolArt != nil
	tavern := v.Kind == TownSurfaceTavern && v.TavernArt != nil
	command := tavern && len(v.Buttons) == 4 && v.TavernArt.CommandUpper != nil
	p.Layers = []RoomLayer{{Kind: RoomLayerFill, Ink: TownRoomFill}}
	switch {
	case school && v.SchoolArt.Background != nil:
		p.Layers = append(p.Layers, RoomLayer{Kind: RoomLayerPicture, Rect: TownContentRegion, Picture: v.SchoolArt.Background})
	case tavern && v.TavernArt.Center != nil:
		p.Layers = append(p.Layers, RoomLayer{Kind: RoomLayerBackdrop})
	default:
		p.Layers = append(p.Layers, RoomLayer{Kind: RoomLayerBox, Rect: TownContentRegion})
	}
	if school {
		p.Layers = append(p.Layers, RoomLayer{Kind: RoomLayerScene, Scene: "movies"})
	}
	p.Panel = testRoomPagePanel(*p, v, school, tavern, command)
	p.Layers = append(p.Layers, RoomLayer{Kind: RoomLayerPanel})
	if !school && !tavern {
		p.Layers = append(p.Layers, RoomLayer{Kind: RoomLayerTitle, Rect: TownRoomTitleRegion, Text: v.Title, Ink: TownRoomText})
	}
	if v.Kind == TownSurfaceTavern {
		p.HidePerson = !tavern
		p.Layers = append(p.Layers, RoomLayer{Kind: RoomLayerContent})
	} else if !school {
		p.Layers = append(p.Layers, RoomLayer{Kind: RoomLayerCells})
	}
	if school {
		p.School = &RoomSchool{Art: v.SchoolArt, Class: v.SchoolClass, ColumnFrame: v.SchoolColumnFrame,
			ColumnSet: v.SchoolColumnSet, IdleShine: v.SchoolIdleShine, IdleSlot: v.SchoolIdleSlot}
		p.Layers = append(p.Layers, RoomLayer{Kind: RoomLayerScene, Scene: "column"}, RoomLayer{Kind: RoomLayerSchool}, RoomLayer{Kind: RoomLayerScene, Scene: "diamond"})
	}
	p.Layers = append(p.Layers, RoomLayer{Kind: RoomLayerButtons})
	for i, b := range v.Buttons {
		if i < len(p.Buttons) {
			p.BeforePane = append(p.BeforePane, RoomHit{Rect: p.Buttons[i], Enabled: b.Enabled,
				Control: RoomControl{Surface: TownSurfaceControl{Kind: TownSurfaceControlButton, Index: i}}})
		}
	}
	if v.Kind == TownSurfaceTavern {
		for i, s := range p.Seats {
			p.AfterPane = append(p.AfterPane, RoomHit{Rect: s.Rect, Enabled: true,
				Control: RoomControl{Surface: TownSurfaceControl{Kind: TownSurfaceControlCell, Index: i}}})
		}
	} else {
		for i, cell := range v.Cells {
			p.AfterPane = append(p.AfterPane, RoomHit{Rect: p.Region("cells", i), Enabled: cell.Enabled,
				Control: RoomControl{Surface: TownSurfaceControl{Kind: TownSurfaceControlCell, Index: i}}})
		}
	}

}
func testRoomPagePanel(p RoomPage, v TownRoomView, school, tavern, command bool) RoomPanel {
	panel := RoomPanel{Composition: TownRoomPanel, Style: v.ButtonStyle, TrackPointer: true, TwoLines: true, Values: v.Buttons}
	if v.Kind == TownSurfaceSchool {
		panel.Composition.MissingBox = TownWideUpperRegion
	}
	switch {
	case school && v.SchoolArt.Upper != nil:
		panel.Art.Body, panel.Art.Seam = v.SchoolArt.Upper, v.SchoolArt.UpperSeam
	case command:
		panel.Composition = TownRoomShopPanel
		panel.Art.Body = v.TavernArt.CommandUpper
	case tavern && v.TavernArt.Upper != nil:
		panel.Art.Body, panel.Art.Seam = v.TavernArt.Upper, v.TavernArt.UpperSeam
	}
	for i := range v.Buttons {
		rect := image.Rectangle{}
		if i < len(p.Buttons) {
			rect = p.Buttons[i]
		}
		panel.Composition.Plaques = append(panel.Composition.Plaques, rect)
		var pair [2]image.Image
		pressedOnly := false
		switch {
		case school && i < len(v.SchoolArt.Buttons):
			pair = WholeRoomPair(v.SchoolArt.Buttons[i])
		case command && i < len(v.TavernArt.CommandButtons):
			pair[1], pressedOnly = v.TavernArt.CommandButtons[i], true
		case tavern:
			artIndex := i
			if len(v.Buttons) == 4 && i > 0 {
				artIndex = i - 1
			}
			if artIndex < len(v.TavernArt.Buttons) {
				pair = WholeRoomPair(v.TavernArt.Buttons[artIndex])
			}
		}
		panel.Art.Plaques = append(panel.Art.Plaques, pair)
		panel.PressedOnly = append(panel.PressedOnly, pressedOnly)
	}
	return panel
}
