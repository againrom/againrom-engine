package audio

import (
	"encoding/binary"
	"testing"
)

// TestDecodeWAVMicrosoftADPCM supplies a mono block with predictor 0, delta 16,
// sample1 100, sample2 90 and codes 1,7: 90,100,116,228 at the device rate.
func TestDecodeWAVMicrosoftADPCM(t *testing.T) {
	b := []byte("RIFF\x00\x00\x00\x00WAVEfmt ")
	b = binary.LittleEndian.AppendUint32(b, 50)
	for _, v := range []uint16{2, 1, uint16(DeviceRate), 0, 0, 0, 8, 4, 32, 4, 7, 256, 0, 512, 65280, 0, 0, 192, 64, 240, 0, 460, 65328, 392, 65304} {
		b = binary.LittleEndian.AppendUint16(b, v)
	}
	b = append(b, "data"...)
	b = binary.LittleEndian.AppendUint32(b, 8)
	b = append(b, 0, 16, 0, 100, 0, 90, 0, 0x17)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)-8))
	s, err := DecodeWAV(b, DeviceRate)
	if err != nil {
		t.Fatal(err)
	}
	if s.Rate != DeviceRate || !equalPCM(s.PCM, []int16{90, 100, 116, 228}) {
		t.Fatalf("sample %+v", s)
	}
}
