package ui

import (
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

// deviceStream is the part of an Ebitengine player the devices drive.
type deviceStream interface {
	Play()
	Pause()
	PauseAndStopReading()
	IsPlaying() bool
	Position() time.Duration
	Seek(time.Duration) error
	Volume() float64
	SetVolume(float64)
	SetBufferSize(time.Duration)
}

// deviceOutput opens streams on the process-wide output: the Ebitengine
// context, or in a silent test process the reading device.
type deviceOutput interface {
	NewPlayer(io.Reader) (deviceStream, error)
	NewPlayerFromBytes([]byte) deviceStream
}

type ebitenOutput struct{ ctx *ebitenaudio.Context }

func (o ebitenOutput) NewPlayer(src io.Reader) (deviceStream, error) {
	p, err := o.ctx.NewPlayer(src)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (o ebitenOutput) NewPlayerFromBytes(src []byte) deviceStream {
	return o.ctx.NewPlayerFromBytes(src)
}

// devicePlayer keeps the engine gain for Volume and hands the device that gain
// times deviceOutputLevel. It starts at device gain 0, before any SetVolume.
type devicePlayer struct {
	deviceStream
	gain atomic.Uint64
}

func newDevicePlayer(p deviceStream) *devicePlayer {
	p.SetVolume(0)
	return &devicePlayer{deviceStream: p}
}

func (p *devicePlayer) SetVolume(gain float64) {
	p.gain.Store(math.Float64bits(gain))
	p.deviceStream.SetVolume(gain * deviceOutputLevel)
}

func (p *devicePlayer) Volume() float64 { return math.Float64frombits(p.gain.Load()) }

func (p *devicePlayer) DeviceGain() float64 { return p.deviceStream.Volume() }

// Close stops the player and its reading of the source, as the closing of an
// Ebitengine player does; the device player is released once unreachable.
func (p *devicePlayer) Close() error {
	p.deviceStream.PauseAndStopReading()
	return nil
}
