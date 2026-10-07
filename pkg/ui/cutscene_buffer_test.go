package ui

import (
	"bytes"
	"testing"
	"time"

	"againrom/pkg/audio"
)

func TestCutsceneDarwinBufferFillsDeviceCallbacks(t *testing.T) {
	const pcmFrameBytes = 4
	const callbackFrames = 1536
	sample := []byte{0x31, 0x12, 0x32, 0x12}
	pcm := bytes.Repeat(sample, audio.DeviceRate)
	q := newCutsceneAudioQueue(2)
	q.push(pcm)
	defer q.closeQueue()
	bufferFrames := int(cutsceneAudioBufferSize("darwin") * audio.DeviceRate / time.Second)
	capacity := bufferFrames * pcmFrameBytes
	var buffered []byte
	for callback := 1; callback <= 3; callback++ {
		refill := make([]byte, capacity-len(buffered))
		if n, err := q.Read(refill); err != nil || n != len(refill) {
			t.Fatalf("refill %d: n=%d err=%v", callback, n, err)
		}
		if !bytes.Equal(refill, bytes.Repeat(sample, len(refill)/pcmFrameBytes)) {
			t.Fatalf("refill %d exhausted the queued nonzero PCM", callback)
		}
		buffered = append(buffered, refill...)
		output := make([]byte, callbackFrames*pcmFrameBytes)
		n := copy(output, buffered)
		buffered = buffered[n:]
		if !bytes.Equal(output, bytes.Repeat(sample, callbackFrames)) {
			t.Fatalf("callback %d: movie buffer supplied %d/%d stereo frames; %d frames became silence with PCM still queued", callback, n/pcmFrameBytes, callbackFrames, callbackFrames-n/pcmFrameBytes)
		}
	}
	if q.consumed() >= int64(len(pcm)) {
		t.Fatal("the callback witness exhausted its queued PCM")
	}
}

func TestCutsceneOtherPlatformsKeepExistingBuffer(t *testing.T) {
	for _, goos := range []string{"windows", "linux"} {
		if got := cutsceneAudioBufferSize(goos); got != 40*time.Millisecond {
			t.Fatalf("%s: movie buffer=%v, want 40ms", goos, got)
		}
	}
}
