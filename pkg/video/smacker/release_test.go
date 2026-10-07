package smacker_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"againrom/pkg/vfs"
	"againrom/pkg/video"
	"againrom/pkg/video/smacker"
)

// smkPayload names one representative sample: archive ("video4"/"video8")
// and member path within it. Together these cover every shape the shipped
// corpus contains: mono and stereo, VIDEO4 and VIDEO8, 15fps and the one
// 25fps movie (logos/buka), and the smallest and largest payloads (the EN
// root's logos/buka and intro/04 respectively; see docs/1179/story.md for
// the full corpus census this set was drawn from).
type smkPayload struct {
	archive, member string
}

var representativeSample = []smkPayload{
	{"video4", "m10/01.smk"},     // mono, 15fps, mid-size
	{"video4", "logos/buka.smk"}, // stereo, 25fps, smallest VIDEO4 payload
	{"video4", "intro/04.smk"},   // mono, largest VIDEO4 payload
	{"video8", "m10/01.smk"},     // stereo, VIDEO8
	{"video8", "m20/01.smk"},     // stereo, VIDEO8, smallest frame count
}

// TestReleaseSmackerDecoderAgainstOracles is this story's own gate-time
// verification (spec stage 2): every payload in both video archives decodes
// without error (a structural census over the corpus), and a representative
// sample additionally matches two independent oracles byte-for-byte —
// ffmpeg's own Smacker decoder (when present on PATH; this environment's own
// verification tool, never a shipped dependency) and the installed
// smackw32.dll through a disposable windows/386 helper process (Windows
// only, matching the existing native cutscene witness's own gating).
func TestReleaseSmackerDecoderAgainstOracles(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS not set")
	}

	ffmpegPath, ffmpegErr := exec.LookPath("ffmpeg")
	ffprobePath, ffprobeErr := exec.LookPath("ffprobe")
	haveFFmpeg := ffmpegErr == nil && ffprobeErr == nil
	if !haveFFmpeg {
		t.Log("ffmpeg/ffprobe not on PATH: structural census and the representative sample both skip the ffmpeg oracle half")
	}

	var helper, dllPath string
	haveNative := false
	if runtime.GOOS == "windows" {
		if dll, err := installedDLL(root); err == nil {
			h := filepath.Join(t.TempDir(), "cutscenehelper.exe")
			cmd := exec.Command("go", "build", "-trimpath", "-o", h, "./cmd/cutscenehelper")
			cmd.Dir = filepath.Join("..", "..", "..")
			for _, v := range os.Environ() {
				if !strings.HasPrefix(v, "GOARCH=") {
					cmd.Env = append(cmd.Env, v)
				}
			}
			cmd.Env = append(cmd.Env, "GOARCH=386")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("build 386 helper: %v\n%s", err, out)
			}
			helper, dllPath, haveNative = h, dll, true
		}
	}
	if !haveNative {
		t.Log("native windows/386 helper unavailable: skipping the installed-DLL oracle half of the representative sample")
	}

	// Every census payload and every sample is a parallel subtest holding one
	// slot; the group returns when all of them have finished.
	slots := make(chan struct{}, smackerWorkers)
	var census atomic.Int64
	t.Run("payloads", func(t *testing.T) {
		structuralCensus(t, root, ffmpegPath, ffprobePath, haveFFmpeg, slots, &census)
		if !haveFFmpeg && !haveNative {
			return
		}
		// Each oracle checks each sample in its own subtest.
		for _, sample := range representativeSample {
			for _, oracle := range []string{"ffmpeg", "native"} {
				if oracle == "ffmpeg" && !haveFFmpeg || oracle == "native" && !haveNative {
					continue
				}
				t.Run("sample-"+payloadName(sample.archive+"/"+sample.member)+"-"+oracle, func(t *testing.T) {
					t.Parallel()
					slots <- struct{}{}
					defer func() { <-slots }()
					media := readMember(t, root, sample)
					smkPath := filepath.Join(t.TempDir(), "sample.smk")
					if err := os.WriteFile(smkPath, media, 0600); err != nil {
						t.Fatal(err)
					}
					if oracle == "ffmpeg" {
						compareAgainstFFmpeg(t, sample, media, ffmpegPath, ffprobePath, smkPath)
					} else {
						compareAgainstNative(t, sample, media, helper, dllPath, smkPath)
					}
				})
			}
		}
	})
	t.Logf("structural census: %d payload(s) decoded without error across both archives", census.Load())
	if !haveFFmpeg && !haveNative {
		t.Skip("no oracle available on this machine to check the representative sample against")
	}
}

// smackerWorkers bounds the payloads checked at once: one sample's ffmpeg
// RGBA output and this decoder's own copy together pass 800 MB.
const smackerWorkers = 4

// payloadName turns an archive address such as "video4/m10/01.smk" into a
// one-level subtest name, "video4-m10-01".
func payloadName(address string) string {
	return strings.NewReplacer("/", "-", ".smk", "").Replace(address)
}

// dimensionRE pulls the pal8 WxH pair out of one ffprobe stderr banner line,
// e.g. "Stream #0:0: Video: smackvid, pal8, 320x240, ...".
var dimensionRE = regexp.MustCompile(`pal8, (\d+)x(\d+)`)

// structuralCensus decodes every payload in both archives with video and
// every present audio track enabled, requiring a clean decode across the
// corpus (spec stage 2's "structural comparison ... over the whole corpus").
// When ffmpeg/ffprobe are available it additionally cross-checks each
// payload's dimensions, frame count and (single-track case) total audio byte
// count against them; without an oracle on PATH it falls back to the
// self-consistency check alone (decoded frame count vs. header count).
// Each payload is a parallel subtest; count gains one per payload that read
// and opened.
func structuralCensus(t *testing.T, root, ffmpegPath, ffprobePath string, haveFFmpeg bool, slots chan struct{}, count *atomic.Int64) {
	t.Helper()
	for _, archive := range []string{"video4", "video8"} {
		fsys, path := openArchive(t, root, archive)
		if fsys == nil {
			continue
		}
		// The archive reader is not documented as safe for concurrent reads.
		var reading sync.Mutex
		for _, e := range fsys.Entries() {
			if !strings.HasSuffix(e.Address, ".smk") {
				continue
			}
			address := e.Address
			t.Run("census-"+payloadName(address), func(t *testing.T) {
				t.Parallel()
				slots <- struct{}{}
				defer func() { <-slots }()
				reading.Lock()
				media, err := fsys.ReadFile(address)
				reading.Unlock()
				if err != nil {
					t.Errorf("%s: %s: read: %v", path, address, err)
					return
				}
				dec, err := smacker.Open(media)
				if err != nil {
					t.Errorf("%s: %s: Open: %v", path, address, err)
					return
				}
				dec.EnableVideo(true)
				var audioTracks []int
				for i := 0; i < 7; i++ {
					if _, _, _, exists := dec.AudioTrack(i); exists {
						dec.EnableAudio(i, true)
						audioTracks = append(audioTracks, i)
					}
				}
				frames := 0
				audioBytes := make(map[int]int)
				for {
					f, err := dec.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Errorf("%s: %s: frame %d: %v", path, address, frames, err)
						break
					}
					frames++
					for _, tr := range audioTracks {
						audioBytes[tr] += len(f.Audio[tr])
					}
				}
				if frames != dec.Frames() {
					t.Errorf("%s: %s: decoded %d frames, header declared %d", path, address, frames, dec.Frames())
				}

				if haveFFmpeg {
					structuralCrossCheck(t, ffmpegPath, ffprobePath, path, address, media, dec, frames, audioTracks, audioBytes)
				}
				count.Add(1)
			})
		}
	}
}

// structuralCrossCheck compares one already-decoded payload's dimensions,
// frame count and (single-track) audio byte total against ffprobe/ffmpeg,
// the same three checks proven interactively over the whole 66-payload
// corpus before this test was written.
func structuralCrossCheck(t *testing.T, ffmpegPath, ffprobePath, archivePath, member string, media []byte, dec *smacker.Decoder, frames int, audioTracks []int, audioBytes map[int]int) {
	t.Helper()
	smkPath := filepath.Join(t.TempDir(), "census.smk")
	if err := os.WriteFile(smkPath, media, 0600); err != nil {
		t.Fatalf("%s: %s: write temp: %v", archivePath, member, err)
	}

	probe := exec.Command(ffprobePath, "-hide_banner", smkPath)
	hideProcessWindow(probe)
	out, _ := probe.CombinedOutput()
	m := dimensionRE.FindSubmatch(out)
	if m == nil {
		t.Errorf("%s: %s: could not parse ffprobe dimensions from: %s", archivePath, member, out)
		return
	}
	refW, refH := 0, 0
	fmt.Sscanf(string(m[1]), "%d", &refW)
	fmt.Sscanf(string(m[2]), "%d", &refH)
	if refW != dec.Width() || refH != dec.Height() {
		t.Errorf("%s: %s: dimensions ours=%dx%d ffprobe=%dx%d", archivePath, member, dec.Width(), dec.Height(), refW, refH)
		return
	}

	ref, finish := startFFmpeg(t, ffmpegPath, smkPath, "-f", "rawvideo", "-pix_fmt", "rgba")
	rgba, err := io.Copy(io.Discard, ref)
	finish()
	perFrame := int64(refW * refH * 4)
	if err != nil || perFrame == 0 || rgba%perFrame != 0 || rgba/perFrame != int64(frames) {
		t.Errorf("%s: %s: frame count ours=%d ffmpeg=%d (rgba bytes=%d, %v)", archivePath, member, frames, rgba/max(perFrame, 1), rgba, err)
		return
	}

	if len(audioTracks) == 1 {
		pcm := runFFmpeg(t, ffmpegPath, smkPath, "-vn", "-f", "s16le", "-acodec", "pcm_s16le")
		if len(pcm) != audioBytes[audioTracks[0]] {
			t.Errorf("%s: %s: audio bytes ours=%d ffmpeg=%d", archivePath, member, audioBytes[audioTracks[0]], len(pcm))
		}
	}
}

func compareAgainstFFmpeg(t *testing.T, sample smkPayload, media []byte, ffmpegPath, ffprobePath, smkPath string) {
	t.Helper()
	dec, err := smacker.Open(media)
	if err != nil {
		t.Fatalf("%s/%s: Open: %v", sample.archive, sample.member, err)
	}
	dec.EnableVideo(true)
	trackIdx := -1
	for i := 0; i < 7; i++ {
		if _, _, _, exists := dec.AudioTrack(i); exists {
			dec.EnableAudio(i, true)
			trackIdx = i
			break
		}
	}

	// ffmpeg's stream is read one frame at a time against ours, so neither
	// whole video is held in memory.
	ref, finish := startFFmpeg(t, ffmpegPath, smkPath, "-f", "rawvideo", "-pix_fmt", "rgba")
	var audio bytes.Buffer
	ours := make([]byte, dec.Width()*dec.Height()*4)
	theirs := make([]byte, len(ours))
	var oursLen, theirsLen int64
	differs := -1
	for frame := 0; ; frame++ {
		f, err := dec.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("%s/%s: Next: %v", sample.archive, sample.member, err)
		}
		for i, idx := range f.Indices {
			ours[4*i], ours[4*i+1], ours[4*i+2], ours[4*i+3] = f.Palette[idx][0], f.Palette[idx][1], f.Palette[idx][2], 255
		}
		n, _ := io.ReadFull(ref, theirs)
		oursLen += int64(len(ours))
		theirsLen += int64(n)
		if differs < 0 && (n != len(ours) || !bytes.Equal(ours, theirs)) {
			differs = frame
		}
		if trackIdx >= 0 {
			audio.Write(f.Audio[trackIdx])
		}
	}
	rest, _ := io.Copy(io.Discard, ref)
	theirsLen += rest
	finish()
	if differs >= 0 || oursLen != theirsLen {
		t.Errorf("%s/%s: video bytes differ from ffmpeg from frame %d: ours=%d ffmpeg=%d", sample.archive, sample.member, differs, oursLen, theirsLen)
	} else {
		t.Logf("%s/%s: video byte-exact vs ffmpeg (%d bytes)", sample.archive, sample.member, oursLen)
	}

	if trackIdx >= 0 {
		refPCM := runFFmpeg(t, ffmpegPath, smkPath, "-vn", "-f", "s16le", "-acodec", "pcm_s16le")
		if !bytes.Equal(refPCM, audio.Bytes()) {
			t.Errorf("%s/%s: audio bytes differ from ffmpeg: ours=%d ffmpeg=%d", sample.archive, sample.member, audio.Len(), len(refPCM))
		} else {
			t.Logf("%s/%s: audio byte-exact vs ffmpeg (%d bytes)", sample.archive, sample.member, audio.Len())
		}
	}
	_ = ffprobePath
}

// startFFmpeg starts ffmpeg writing to its standard output and returns that
// stream; finish waits for ffmpeg once the caller has read the stream to its
// end and fails the test on ffmpeg's error. A test that stops early kills it.
func startFFmpeg(t *testing.T, ffmpegPath, smkPath string, extra ...string) (io.Reader, func()) {
	t.Helper()
	args := append([]string{"-hide_banner", "-loglevel", "error", "-y", "-i", smkPath}, extra...)
	cmd := exec.Command(ffmpegPath, append(args, "-")...)
	hideProcessWindow(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("ffmpeg %v: %v", args, err)
	}
	done := false
	t.Cleanup(func() {
		if !done {
			cmd.Process.Kill()
			cmd.Wait()
		}
	})
	return out, func() {
		t.Helper()
		done = true
		if err := cmd.Wait(); err != nil {
			t.Fatalf("ffmpeg %v: %v: %s", args, err, stderr.Bytes())
		}
	}
}

func runFFmpeg(t *testing.T, ffmpegPath, smkPath string, extra ...string) []byte {
	t.Helper()
	args := append([]string{"-hide_banner", "-loglevel", "error", "-y", "-i", smkPath}, extra...)
	args = append(args, "-")
	cmd := exec.Command(ffmpegPath, args...)
	hideProcessWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffmpeg %v: %v", args, err)
	}
	return out
}

func compareAgainstNative(t *testing.T, sample smkPayload, media []byte, helper, dll, smkPath string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, helper, "-dll", dll, "-input", smkPath, "-unpaced")
	hideProcessWindow(cmd)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	finished := false
	defer func() {
		pipe.Close()
		if !finished {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	info, err := video.ReadHeader(pipe)
	if err != nil {
		t.Fatalf("%s/%s: native header: %v", sample.archive, sample.member, err)
	}
	dec, err := smacker.Open(media)
	if err != nil {
		t.Fatalf("%s/%s: Open: %v", sample.archive, sample.member, err)
	}
	dec.EnableVideo(true)
	if int(info.Width) != dec.Width() || int(info.Height) != dec.Height() || int(info.Frames) != dec.Frames() {
		t.Fatalf("%s/%s: dimension/frame mismatch native=%dx%d/%d ours=%dx%d/%d", sample.archive, sample.member, info.Width, info.Height, info.Frames, dec.Width(), dec.Height(), dec.Frames())
	}
	if info.AudioRate != 0 {
		dec.EnableAudio(0, true)
	}
	diffFrames := 0
	audioBytes, diffAudio := 0, 0
	for n := uint32(0); n < info.Frames; n++ {
		nativeFrame, nativeAudio, err := video.ReadFrame(pipe, info)
		if err != nil {
			t.Fatalf("%s/%s: native frame %d: %v", sample.archive, sample.member, n, err)
		}
		f, err := dec.Next()
		if err != nil {
			t.Fatalf("%s/%s: our frame %d: %v", sample.archive, sample.member, n, err)
		}
		ours := make([]byte, dec.Width()*dec.Height()*4)
		for i, idx := range f.Indices {
			ours[4*i], ours[4*i+1], ours[4*i+2], ours[4*i+3] = f.Palette[idx][0], f.Palette[idx][1], f.Palette[idx][2], 255
		}
		if !bytes.Equal(nativeFrame.Pix, ours) {
			diffFrames++
		}
		audioBytes += len(nativeAudio)
		if !bytes.Equal(nativeAudio, f.Audio[0]) {
			diffAudio++
		}
	}
	var extra [1]byte
	if n, err := pipe.Read(extra[:]); n != 0 || err != io.EOF {
		t.Fatalf("native stream after final frame: %d %v", n, err)
	}
	err = cmd.Wait()
	finished = true
	if err != nil {
		t.Fatal("native helper exit", err)
	}
	if diffFrames != 0 {
		t.Errorf("%s/%s: %d/%d frames differ from the installed DLL", sample.archive, sample.member, diffFrames, info.Frames)
	} else {
		t.Logf("%s/%s: video byte-exact vs installed DLL (%d frames)", sample.archive, sample.member, info.Frames)
	}
	if diffAudio != 0 {
		t.Errorf("%s/%s: %d audio chunks differ from installed DLL", sample.archive, sample.member, diffAudio)
	} else {
		t.Logf("%s/%s: audio byte-exact vs installed DLL (%d bytes)", sample.archive, sample.member, audioBytes)
	}
}

func openArchive(t *testing.T, root, archive string) (*vfs.FS, string) {
	t.Helper()
	path, err := installPath(root, "Allods", archive+".res")
	if err != nil {
		t.Fatalf("%s: %v", archive, err)
	}
	fsys, err := vfs.OpenFileBacked([]string{path}, nil)
	if err != nil {
		t.Fatalf("%s: open: %v", path, err)
	}
	return fsys, path
}

func readMember(t *testing.T, root string, sample smkPayload) []byte {
	t.Helper()
	fsys, path := openArchive(t, root, sample.archive)
	address := sample.archive + "/" + sample.member
	media, err := fsys.ReadFile(address)
	if err != nil {
		t.Fatalf("%s: %s: %v", path, address, err)
	}
	return media
}

// installPath folds each path component case-insensitively over root,
// matching pkg/game's own cutsceneInstallPath (kept independent here: this
// package has no import path to pkg/game, which itself imports pkg/video).
func installPath(root string, components ...string) (string, error) {
	path, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	for _, component := range components {
		entries, err := os.ReadDir(path)
		if err != nil {
			return "", err
		}
		found := false
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), component) {
				path = filepath.Join(path, entry.Name())
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("installed %s missing", component)
		}
	}
	return path, nil
}

func installedDLL(root string) (string, error) {
	return installPath(root, "smackw32.dll")
}
