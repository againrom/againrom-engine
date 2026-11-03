package wav

import (
	"encoding/binary"
	"fmt"
)

// decodeADPCM implements Microsoft's Multimedia Standards Update, revision 3.0,
// Microsoft ADPCM section: block headers seed independent channel histories,
// then high-first signed nibbles correct the linear prediction. Integer division
// truncates toward zero. Only samples present in a partial last block are read.
func (p *PCM) decodeADPCM(format, fact []byte) error {
	if len(format) < 22 {
		return fmt.Errorf("wav: ADPCM fmt extension is truncated")
	}
	ext := int(binary.LittleEndian.Uint16(format[16:]))
	count := int(binary.LittleEndian.Uint16(format[20:]))
	if ext < 4 || ext > len(format)-18 || count == 0 || count > (ext-4)/4 {
		return fmt.Errorf("wav: ADPCM coefficient count %d does not fit extension of %d bytes", count, ext)
	}
	align := int(binary.LittleEndian.Uint16(format[12:]))
	samples := int(binary.LittleEndian.Uint16(format[18:]))
	header := 7 * p.Channels
	if align < header || samples < 2 || header+((samples-2)*p.Channels+1)/2 != align {
		return fmt.Errorf("wav: ADPCM block size %d does not fit %d samples and %d channel(s)", align, samples, p.Channels)
	}
	var factFrames uint32
	if fact != nil {
		if len(fact) < 4 {
			return fmt.Errorf("wav: ADPCM fact chunk is truncated")
		}
		factFrames = binary.LittleEndian.Uint32(fact)
	}
	data := p.Data
	out := make([]byte, 0, len(data))
	for pos := 0; pos < len(data); {
		block := data[pos : pos+min(align, len(data)-pos)]
		if len(block) < header {
			return fmt.Errorf("wav: ADPCM block at %d has %d bytes, want a header of %d", pos, len(block), header)
		}
		var channels [2]adpcmChannel
		for ch := 0; ch < p.Channels; ch++ {
			predictor := int(block[ch])
			if predictor >= count {
				return fmt.Errorf("wav: ADPCM block at %d channel %d predictor %d, only %d coefficients", pos, ch, predictor, count)
			}
			coef := format[22+4*predictor:]
			channels[ch] = adpcmChannel{
				coef1:   int64(int16(binary.LittleEndian.Uint16(coef))),
				coef2:   int64(int16(binary.LittleEndian.Uint16(coef[2:]))),
				delta:   int64(binary.LittleEndian.Uint16(block[p.Channels+2*ch:])),
				sample1: int64(int16(binary.LittleEndian.Uint16(block[3*p.Channels+2*ch:]))),
				sample2: int64(int16(binary.LittleEndian.Uint16(block[5*p.Channels+2*ch:]))),
			}
		}
		for ch := 0; ch < p.Channels; ch++ {
			out = binary.LittleEndian.AppendUint16(out, uint16(channels[ch].sample2))
		}
		for ch := 0; ch < p.Channels; ch++ {
			out = binary.LittleEndian.AppendUint16(out, uint16(channels[ch].sample1))
		}
		frames := min(samples, 2+(len(block)-header)*2/p.Channels)
		for i := 0; i < (frames-2)*p.Channels; i++ {
			nibble := block[header+i/2]
			if i%2 == 0 {
				nibble >>= 4
			}
			sample, err := channels[i%p.Channels].next(nibble & 15)
			if err != nil {
				return fmt.Errorf("wav: ADPCM block at %d: %w", pos, err)
			}
			out = binary.LittleEndian.AppendUint16(out, uint16(sample))
		}
		pos += len(block)
	}
	if fact != nil {
		frames := min(uint64(factFrames), uint64(len(out)/(2*p.Channels)))
		out = out[:int(frames)*2*p.Channels]
	}
	p.Data, p.BitsPerSample = out, 16
	return nil
}

type adpcmChannel struct {
	coef1, coef2, delta, sample1, sample2 int64
}

func (c *adpcmChannel) next(nibble byte) (int16, error) {
	if c.delta > (1<<63-1)/768 {
		return 0, fmt.Errorf("delta exceeds the arithmetic range")
	}
	code := int64(nibble)
	if code >= 8 {
		code -= 16
	}
	sample := (c.sample1*c.coef1+c.sample2*c.coef2)/256 + code*c.delta
	sample = max(int64(-32768), min(int64(32767), sample))
	c.sample2, c.sample1 = c.sample1, sample
	adaptation := [...]int64{230, 230, 230, 230, 307, 409, 512, 614, 768, 614, 512, 409, 307, 230, 230, 230}
	c.delta = max(int64(16), c.delta*adaptation[nibble]/256)
	return int16(sample), nil
}
