package game

import "github.com/hajimehoshi/ebiten/v2"

type presentationProbe struct {
	draw func() error
	done bool
	err  error
}

func (g *presentationProbe) Layout(_, _ int) (int, int) { return 1024, 768 }
func (g *presentationProbe) Update() error {
	if g.done {
		return ebiten.Termination
	}
	return nil
}
func (g *presentationProbe) Draw(_ *ebiten.Image) {
	if !g.done {
		g.err, g.done = g.draw(), true
	}
}

func runPresentationProbe(draw func() error) error {
	g := &presentationProbe{draw: draw}
	ebiten.SetWindowSize(64, 48)
	ebiten.SetWindowPosition(-32000, -32000)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{InitUnfocused: true, SkipTaskbar: true}); err != nil && err != ebiten.Termination {
		return err
	}
	return g.err
}
