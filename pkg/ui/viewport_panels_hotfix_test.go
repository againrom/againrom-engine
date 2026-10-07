package ui

import (
	"image"
	"testing"
)

func bottomPanelViewportFixture(t *testing.T) *Viewer {
	t.Helper()
	v := newViewer(t, bigGrid())
	v.hudHidden[hudPanelPack], v.hudHidden[hudPanelBook] = false, false
	v.SetFont(panelFont())
	v.SetEntities([]MapEntity{{ID: 5, Life: LifeAlive}})
	v.sel = selection{5}
	v.SetInventorySubject(InventorySubject{ID: 5})
	v.SetSpellbook(5, nil) // a selected unit's empty book is still a book
	v.Layout(1920, 1080)
	v.syncMapViewport()
	return v
}

func TestBottomPanelsDefineTheLiveMapSurface(t *testing.T) {
	tests := []struct {
		name       string
		pack, book bool
		viewH      int
		packRect   image.Rectangle
		bookRect   image.Rectangle
	}{
		{name: "none", viewH: 768},
		{name: "pack only", pack: true, viewH: 678, packRect: image.Rect(0, 678, 1206, 768)},
		{name: "book only", book: true, viewH: 683, bookRect: image.Rect(0, 683, 1206, 768)},
		{name: "both", pack: true, book: true, viewH: 593,
			packRect: image.Rect(0, 678, 1206, 768), bookRect: image.Rect(0, 593, 1206, 678)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := bottomPanelViewportFixture(t)
			v.hudHidden[hudPanelPack] = !tc.pack
			v.hudHidden[hudPanelBook] = !tc.book
			v.syncMapViewport()

			if got := v.ViewportSize(); got != image.Pt(1206, tc.viewH) {
				t.Fatalf("viewport = %v, want (1206,%d)", got, tc.viewH)
			}
			pack, _, packOK := v.packBar()
			if packOK != tc.pack || (packOK && pack != tc.packRect) {
				t.Fatalf("pack = (%v,%v), want (%v,%v)", pack, packOK, tc.packRect, tc.pack)
			}
			book, _, bookOK := v.spellbookBar()
			if bookOK != tc.book || (bookOK && book != tc.bookRect) {
				t.Fatalf("book = (%v,%v), want (%v,%v)", book, bookOK, tc.bookRect, tc.book)
			}
			if !v.mapSurfaceCaptures(100, tc.viewH-1) {
				t.Fatal("last row above the panels is not game surface")
			}
			if v.mapSurfaceCaptures(100, tc.viewH) {
				t.Fatal("first row reserved for bottom UI is still game surface")
			}
		})
	}
}

func TestClosingBottomPanelsRevealsDownwardAndBouncesOnlyAtTheMapEdge(t *testing.T) {
	v := bottomPanelViewportFixture(t)
	if v.ViewportSize().Y != 593 {
		t.Fatalf("setup viewport height = %d, want 593", v.ViewportSize().Y)
	}

	v.cam.X, v.cam.Y = 1000, 1000
	v.cam.Clamp()
	before := v.cam.Y
	v.toggleHudPanel(hudPanelPack)
	v.toggleHudPanel(hudPanelBook)
	if v.ViewportSize().Y != 768 {
		t.Fatalf("closed viewport height = %d, want 768", v.ViewportSize().Y)
	}
	if v.cam.Y != before {
		t.Fatalf("closing away from the edge moved Y from %v to %v", before, v.cam.Y)
	}

	// Reopen both over the same large map and park at the bottom edge of the
	// short surface. Use an inset drawable edge, as production maps do. Each
	// close grows the surface; only the clamp moves the origin upward, exactly
	// enough to keep the new bottom on that edge rather than the full grid.
	v.toggleHudPanel(hudPanelPack)
	v.toggleHudPanel(hudPanelBook)
	drawableBottom := v.cam.WorldH() - 256
	v.cam.SetClampBounds(256, 256, v.cam.WorldW()-256, drawableBottom)
	v.cam.Pan(0, 1e9)
	if got, want := v.cam.Y, drawableBottom-593; got != want {
		t.Fatalf("short-surface bottom = %v, want %v", got, want)
	}
	v.toggleHudPanel(hudPanelPack)
	if got, want := v.cam.Y, drawableBottom-683; got != want {
		t.Fatalf("book-only bounce = %v, want %v", got, want)
	}
	v.toggleHudPanel(hudPanelBook)
	if got, want := v.cam.Y, drawableBottom-768; got != want {
		t.Fatalf("fully-open surface bounce = %v, want %v", got, want)
	}
}

// With nothing selected the open pack and book are drawn empty; the map
// surface and its scroll limit are the ones a selected hero gets.
func TestNoSelectionKeepsTheBottomPanelsViewport(t *testing.T) {
	v := bottomPanelViewportFixture(t)
	if got := v.ViewportSize().Y; got != 593 {
		t.Fatalf("selected viewport height = %d, want 593", got)
	}
	v.cam.SetClampBounds(256, 256, v.cam.WorldW()-256, v.cam.WorldH()-256)
	v.cam.Pan(0, 1e9)
	selectedY := v.cam.Y

	v.sel = selection{}
	v.ClearSpellbook()
	v.syncMapViewport()
	if got := v.ViewportSize().Y; got != 593 {
		t.Fatalf("no-selection viewport height = %d, want 593", got)
	}
	v.cam.Pan(0, 1e9)
	if v.cam.Y != selectedY {
		t.Fatalf("no-selection bottom scroll limit = %v, want %v", v.cam.Y, selectedY)
	}
	if v.mapSurfaceCaptures(100, 600) {
		t.Fatal("row under the empty panels is still game surface")
	}
}
