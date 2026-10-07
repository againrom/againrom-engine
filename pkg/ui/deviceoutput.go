package ui

import (
	"math"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"
)

// testVolumeEnv names the percent (0 to 100) at which a test process plays.
const testVolumeEnv = "AGAINROM_TEST_VOLUME"

// deviceOutputLevel scales only the gain handed to a real device player. The
// game plays at 1; a test process is silent unless testVolumeEnv asks otherwise.
var deviceOutputLevel = outputLevel(testing.Testing(), os.Getenv(testVolumeEnv))

func outputLevel(testProcess bool, percent string) float64 {
	if !testProcess {
		return 1
	}
	n, err := strconv.Atoi(strings.TrimSpace(percent))
	if err != nil || n < 0 || n > 100 {
		return 0
	}
	return float64(n) / 100
}

// devicePlayer keeps the engine gain for Volume and hands the device that gain
// times deviceOutputLevel. It starts at device gain 0, before any SetVolume.
type devicePlayer struct {
	*ebitenaudio.Player
	gain atomic.Uint64
}

func newDevicePlayer(p *ebitenaudio.Player) *devicePlayer {
	p.SetVolume(0)
	return &devicePlayer{Player: p}
}

func (p *devicePlayer) SetVolume(gain float64) {
	p.gain.Store(math.Float64bits(gain))
	p.Player.SetVolume(gain * deviceOutputLevel)
}

func (p *devicePlayer) Volume() float64 { return math.Float64frombits(p.gain.Load()) }

func (p *devicePlayer) DeviceGain() float64 { return p.Player.Volume() }
