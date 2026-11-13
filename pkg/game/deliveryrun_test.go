package game

import (
	"io"
	"testing"
	"time"

	"againrom/internal/devicefake"
	"againrom/pkg/ui"
)

// deliveryRun is the one headless front end run for a delivery witness: the
// production delivery seam fed by controlled players, and a clock the run
// owns and the front end's town animation reads.
type deliveryRun struct {
	t       *testing.T
	front   *FrontEnd
	app     *ui.App
	now     time.Time
	players []*devicefake.Player
}

// newDeliveryRun installs the controlled player factory for the test when
// controlled. A clocked run's players consume their buffers against the run's
// clock, as a sound card would; otherwise the test drives them by hand.
func newDeliveryRun(t *testing.T, controlled, clocked bool) *deliveryRun {
	t.Helper()
	r := &deliveryRun{t: t, now: time.Unix(100, 0)}
	if controlled {
		t.Cleanup(ui.SetDeliveryPlayerFactory(func(reader io.ReadSeeker) (ui.DeliveryDevicePlayer, error) {
			p := &devicefake.Player{Reader: reader}
			if clocked {
				p.Clock, p.Gain = func() time.Time { return r.now }, 1
			}
			r.players = append(r.players, p)
			return p, nil
		}))
	}
	return r
}

// use makes the run's clock the front end's and binds the run to the front
// end and its app. A non-nil now resets the clock first.
func (r *deliveryRun) use(f *FrontEnd, app *ui.App, now *time.Time) {
	r.front, r.app = f, app
	if now != nil {
		r.now = *now
	}
	f.TownAnimationNow = func() time.Time { return r.now }
	r.t.Cleanup(app.StopAudio)
}
