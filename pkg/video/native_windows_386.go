package video

import (
	"debug/pe"
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

// exportBindings reads only PE metadata, not decoder instructions. Decoration
// is derived from the file and the published stdcall arity, never assumed to
// be an SDK source declaration. Duplicate identities and forwarders refuse.
func exportBindings(path string, arities map[string]int) (map[string]string, error) {
	f, err := pe.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h, ok := f.OptionalHeader.(*pe.OptionalHeader32)
	if !ok || f.Machine != pe.IMAGE_FILE_MACHINE_I386 {
		return nil, fmt.Errorf("video: decoder is not PE32/i386")
	}
	dir := h.DataDirectory[0]
	read := func(rva uint32, n uint32) ([]byte, error) {
		if n > 1<<20 {
			return nil, fmt.Errorf("video: export metadata exceeds bound")
		}
		for _, s := range f.Sections {
			if rva >= s.VirtualAddress && uint64(rva)+uint64(n) <= uint64(s.VirtualAddress)+uint64(s.Size) {
				b := make([]byte, int(n))
				_, err := s.ReadAt(b, int64(rva-s.VirtualAddress))
				return b, err
			}
		}
		return nil, fmt.Errorf("video: export RVA outside file-backed sections")
	}
	b, err := read(dir.VirtualAddress, 40)
	if err != nil {
		return nil, err
	}
	nfunc, nname := binary.LittleEndian.Uint32(b[20:]), binary.LittleEndian.Uint32(b[24:])
	if nfunc > 4096 || nname > 4096 {
		return nil, fmt.Errorf("video: export population exceeds bound")
	}
	funcs, err := read(binary.LittleEndian.Uint32(b[28:]), nfunc*4)
	if err != nil {
		return nil, err
	}
	names, err := read(binary.LittleEndian.Uint32(b[32:]), nname*4)
	if err != nil {
		return nil, err
	}
	ords, err := read(binary.LittleEndian.Uint32(b[36:]), nname*2)
	if err != nil {
		return nil, err
	}
	bindings := make(map[string]string)
	for i := uint32(0); i < nname; i++ {
		rva := binary.LittleEndian.Uint32(names[i*4:])
		var name []byte
		for j := uint32(0); j < 128; j++ {
			c, err := read(rva+j, 1)
			if err != nil {
				return nil, err
			}
			if c[0] == 0 {
				break
			}
			name = append(name, c[0])
		}
		if len(name) == 128 {
			return nil, fmt.Errorf("video: overlong export name")
		}
		for base, count := range arities {
			if string(name) != base && string(name) != "_"+base+"@"+strconv.Itoa(count*4) {
				continue
			}
			ord := uint32(binary.LittleEndian.Uint16(ords[i*2:]))
			if ord >= nfunc {
				return nil, fmt.Errorf("video: invalid export ordinal")
			}
			entry := binary.LittleEndian.Uint32(funcs[ord*4:])
			if entry == 0 || (uint64(entry) >= uint64(dir.VirtualAddress) && uint64(entry) < uint64(dir.VirtualAddress)+uint64(dir.Size)) || bindings[base] != "" {
				return nil, fmt.Errorf("video: ambiguous or forwarded export %s", base)
			}
			bindings[base] = string(name)
		}
	}
	if len(bindings) != len(arities) {
		return nil, fmt.Errorf("video: required decoder exports missing")
	}
	return bindings, nil
}

func decodeNative(dllPath, inputPath string, out io.Writer, packed565, paced bool) error {
	if !filepath.IsAbs(dllPath) || !filepath.IsAbs(inputPath) {
		return fmt.Errorf("video: absolute paths required")
	}
	f, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	meta, err := InspectNativeInput(f)
	if err != nil {
		return err
	}
	arities := map[string]int{"SmackOpen": 3, "SmackToBuffer": 7, "SmackDoFrame": 1, "SmackNextFrame": 1, "SmackClose": 1}
	if meta.AudioTrackBytes > 0 {
		arities["SmackSoundInTrack"] = 2
		arities["SmackGetTrackData"] = 3
	}
	bindings, err := exportBindings(dllPath, arities)
	if err != nil {
		return err
	}
	// Restrict dependency lookup to the lawful DLL's own directory and Windows
	// system libraries. The private input directory never participates.
	path, err := syscall.UTF16PtrFromString(dllPath)
	if err != nil {
		return err
	}
	loader := syscall.NewLazyDLL("kernel32.dll").NewProc("LoadLibraryExW")
	handle, _, loadErr := loader.Call(uintptr(unsafe.Pointer(path)), 0, 0x1100)
	if handle == 0 {
		return fmt.Errorf("video: load decoder: %w", loadErr)
	}
	dll := &syscall.DLL{Name: dllPath, Handle: syscall.Handle(handle)}
	defer dll.Release()
	procs := make(map[string]*syscall.Proc)
	for base, name := range bindings {
		p, err := dll.FindProc(name)
		if err != nil {
			return err
		}
		procs[base] = p
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// VIDEO-045/046: borrowed file handle at offset zero, automatic buffering,
	// DLL-allocated state. Reused-state/preload/audio selectors remain clear.
	state, _, _ := procs["SmackOpen"].Call(f.Fd(), 0x1000, ^uintptr(0))
	if state == 0 {
		return fmt.Errorf("video: decoder refused input")
	}
	defer procs["SmackClose"].Call(state)
	// Copy only the published state extent through a checked OS read. Native
	// addresses never become Go pointers or slices, even on a bad DLL return.
	var fields [0x618]byte
	readMemory := syscall.NewLazyDLL("kernel32.dll").NewProc("ReadProcessMemory")
	readState := func() error {
		var read uintptr
		ok, _, err := readMemory.Call(^uintptr(0), state, uintptr(unsafe.Pointer(&fields[0])), uintptr(len(fields)), uintptr(unsafe.Pointer(&read)))
		if ok == 0 || read != uintptr(len(fields)) {
			return fmt.Errorf("video: native state read: %v", err)
		}
		return nil
	}
	if err := readState(); err != nil {
		return err
	}
	if binary.LittleEndian.Uint32(fields[4:]) != meta.Width || binary.LittleEndian.Uint32(fields[8:]) != meta.Height || binary.LittleEndian.Uint32(fields[12:]) != meta.Frames {
		return fmt.Errorf("video: decoder/header dimensions disagree")
	}
	stride := (int(meta.Width) + 7) &^ 7
	flags := uintptr(0)
	if packed565 {
		stride *= 2
		flags = 0xc0000000
	}
	pixels := make([]byte, stride*int(meta.Height))
	var pin runtime.Pinner
	pin.Pin(&pixels[0])
	// Close the native descriptor before unpinning its borrowed destination.
	defer pin.Unpin()
	defer procs["SmackToBuffer"].Call(state, 0, 0, 0, 0, 0, 0)
	procs["SmackToBuffer"].Call(state, 0, 0, uintptr(stride), uintptr(meta.Height), uintptr(unsafe.Pointer(&pixels[0])), flags)
	// One reused destination for every frame's audio pull, sized to the
	// header's own AudioSize[0] bound (InspectNativeInput) and pinned once,
	// exactly like the pixel destination above.
	var audioBuf []byte
	if meta.AudioTrackBytes > 0 {
		audioBuf = make([]byte, meta.AudioTrackBytes)
		pin.Pin(&audioBuf[0])
	}
	if err := WriteHeader(out, meta.Info); err != nil {
		return err
	}
	frame := image.NewRGBA(image.Rect(0, 0, int(meta.Width), int(meta.Height)))
	for n := uint32(0); n < meta.Frames; n++ {
		procs["SmackDoFrame"].Call(state)
		if err := readState(); err != nil {
			return err
		}
		if binary.LittleEndian.Uint32(fields[0x374:]) != n {
			return fmt.Errorf("video: unexpected native frame position")
		}
		for y := 0; y < int(meta.Height); y++ {
			for x := 0; x < int(meta.Width); x++ {
				d := (y*int(meta.Width) + x) * 4
				if packed565 {
					v := binary.LittleEndian.Uint16(pixels[y*stride+x*2:])
					frame.Pix[d], frame.Pix[d+1], frame.Pix[d+2] = byte(v>>11)<<3, byte((v>>5)&63)<<2, byte(v&31)<<3
				} else {
					p := int(pixels[y*stride+x])*3 + 0x6c
					copy(frame.Pix[d:d+3], fields[p:p+3])
				}
				frame.Pix[d+3] = 255
			}
		}
		var audioChunk []byte
		if audioBuf != nil {
			if has, _, _ := procs["SmackSoundInTrack"].Call(state, audioTrackFlag); has != 0 {
				wrote, _, _ := procs["SmackGetTrackData"].Call(state, uintptr(unsafe.Pointer(&audioBuf[0])), audioTrackFlag)
				w := int32(wrote)
				if w < 0 || int(w) > len(audioBuf) {
					return fmt.Errorf("video: decoder returned an out-of-bounds audio length")
				}
				if w > 0 {
					audioChunk = audioBuf[:w]
				}
			}
		}
		if err := WriteFrame(out, meta.Info, frame, audioChunk); err != nil {
			return err
		}
		// Preserve the final frame's duration, too. Pipe backpressure never
		// creates catch-up bursts; the parent still owns cancellation/stalls.
		if paced {
			time.Sleep(meta.Interval)
		}
		if n+1 < meta.Frames {
			procs["SmackNextFrame"].Call(state)
		}
	}
	runtime.KeepAlive(pixels)
	runtime.KeepAlive(audioBuf)
	return nil
}
