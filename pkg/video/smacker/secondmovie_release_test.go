package smacker_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"againrom/pkg/vfs"
)

func TestReleaseSecondGameFirstMovieAgainstOracles(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS not set")
	}
	archive := filepath.Join(root, "video.res")
	if _, err := os.Stat(archive); os.IsNotExist(err) {
		t.Skip("the root has no second-game video archive")
	}
	fs, err := vfs.OpenFileBacked([]string{archive}, nil)
	if err != nil {
		t.Fatal(err)
	}
	media, err := fs.ReadFile("video/teleport/01.smk")
	if err != nil {
		t.Fatal(err)
	}
	scratch := filepath.Join(t.TempDir(), "first.smk")
	if err := os.WriteFile(scratch, media, 0600); err != nil {
		t.Fatal(err)
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Fatal("first movie witness needs the external FFmpeg oracle", err)
	}
	compareAgainstFFmpeg(t, smkPayload{"video", "teleport/01.smk"}, media, ffmpeg, "", scratch)
	if runtime.GOOS != "windows" {
		t.Skip("installed library oracle requires Windows")
	}
	helper := os.Getenv("AGAINROM_CUTSCENE_HELPER")
	if helper == "" {
		t.Fatal("first movie witness needs the separately stamped 386 library helper")
	}
	dll, err := installedDLL(root)
	if err != nil {
		t.Fatal(err)
	}
	compareAgainstNative(t, smkPayload{"video", "teleport/01.smk"}, media, helper, dll, scratch)
}
