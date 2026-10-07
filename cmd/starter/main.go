// Command starter is the launcher for againrom: it chooses a base game, the
// mods to load and the game's parameters, keeps them in an ini file, and starts
// againrom.exe with the resulting command line. It reads game folders only to
// look at them and writes only its own ini file.
//
//	starter [-ini <path>] [-screenshot <png>] [-version]
//
// The ini file defaults to starter.ini in the folder of starter.exe. The game
// program defaults to againrom.exe in that folder, and the mods directory to
// its mods subfolder; the ini can name others.
//
// -screenshot draws the window once into a PNG and exits, without opening a
// window.
//
// # Ini format
//
//	[starter]
//	againrom =
//	close-on-play = false
//	last-base = en
//
//	[bases]
//	en = C:\Games\againrom\en
//
//	[mods]
//	dir =
//	enabled = alpha,beta
//
//	[options]
//	sound = default
//	volume =
//	movies = true
//	video = normal
//	markers = false
//	saves =
//	mission =
//	picker = false
//	skill =
//	extra =
//
// againrom names the game program (empty: againrom.exe beside the starter),
// last-base the key of the selected base, dir the mods folder (empty: mods
// beside the starter) and enabled the mod ids in load order. sound is default,
// on or off; volume is empty (the game's saved preference) or 0-100; video is
// normal, 4x or 8x; picker opens the map picker for New Game; extra is a free
// argument line split on blanks, double quotes grouping.
//
// Values are the text after the first '=' without surrounding blanks; a
// comment is a whole line starting with ';' or '#'. Keys the starter does not
// know and comment lines survive a save.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"againrom/pkg/game"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func realDeps() deps {
	return deps{
		run: func(exe string, args []string) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, exe, args...)
			cmd.Dir = filepath.Dir(exe)
			hideConsole(cmd)
			var out bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &out
			err := cmd.Run()
			return out.String(), err
		},
		start: func(exe string, args []string) error {
			cmd := exec.Command(exe, args...)
			cmd.Dir = filepath.Dir(exe)
			hideConsole(cmd)
			if err := cmd.Start(); err != nil {
				return err
			}
			go cmd.Wait()
			return nil
		},
		paste:   clipboardText,
		inspect: game.InspectInstall,
		async:   func(f func()) { go f() },
	}
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("starter", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	iniPath := fs.String("ini", "", "settings file (default: starter.ini beside this program)")
	shot := fs.String("screenshot", "", "draw the window into this PNG file and exit, opening no window")
	version := fs.Bool("version", false, "print the program name, version and source revision, and exit")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "starter: %v\nusage: starter [-ini <path>] [-screenshot <png>] [-version]\n", err)
		return 2
	}
	if *version {
		fmt.Fprintln(stdout, versionLine())
		return 0
	}
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(stderr, "starter:", err)
		return 1
	}
	if *iniPath == "" {
		*iniPath = defaultINIPath(filepath.Dir(self))
	}
	d := realDeps()
	if *shot != "" {
		d.async = func(f func()) { f() }
	}
	a := newApp(*iniPath, self, runtime.GOOS, programVersion(), d)
	if *shot != "" {
		a.poll()
		return writeScreenshot(a, *shot, stderr)
	}
	if err := runWindow(a); err != nil {
		fmt.Fprintln(stderr, "starter:", err)
		return 1
	}
	return 0
}

func writeScreenshot(a *app, path string, stderr io.Writer) int {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(stderr, "starter:", err)
		return 1
	}
	defer f.Close()
	var img image.Image = a.render()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(stderr, "starter:", err)
		return 1
	}
	return 0
}
