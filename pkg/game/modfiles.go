package game

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io/fs"
	"slices"
	"strings"

	"againrom/pkg/audio"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/mod"
	"againrom/pkg/vfs"
)

// modFiles is the one boundary through which a front end's mods reach the
// files the game reads out of the install's archives. It is this front end's
// own container filesystem and the sound archive's: every file a mod serves
// at an install path is an entry on it, and every reader of the archives reads
// through it. The install files mods carry, the pictures of mod items, the
// body choice and the maps and texts of mod missions are its entries.
//
// Entries are decided in mod order: an entry for a path another mod serves
// goes to the mod that loads after the other, and the pair is an override;
// when neither mod loads after the other the entry is refused.
type modFiles struct {
	fs, sounds *vfs.FS
	order      mod.Precedence
	owners     map[string]modFileEntry
	overrides  []string
}

// modFileEntry is one file served at an install path. mod and file name the
// mod and the file in it that serve it; mod is empty for a file the mods'
// tables compose together, which no other entry shares.
type modFileEntry struct {
	mod, file, address string
	read               func() ([]byte, error)
}

// modFileLayer is the front end's boundary, made on first use: the archives
// become this front end's own copy, whose containers fork the shared ones,
// since the archives are shared by every front end over the install.
func (f *FrontEnd) modFileLayer() *modFiles {
	if f.Archives.mods != nil {
		return f.Archives.mods
	}
	fs := f.Archives.Containers.Fork()
	own := *f.Archives
	own.Containers = fs
	own.mods = &modFiles{fs: fs, owners: map[string]modFileEntry{}}
	f.Archives = &own
	return own.mods
}

// ModFileOverrides are the lines naming each install path a later mod serves
// over an earlier one.
func (f *FrontEnd) ModFileOverrides() []string {
	if f == nil || f.Archives == nil || f.Archives.mods == nil {
		return nil
	}
	return slices.Clone(f.Archives.mods.overrides)
}

// serve puts e on the boundary under the mod order.
func (b *modFiles) serve(e modFileEntry) error {
	key := mod.FoldAddress(e.address)
	if prior, ok := b.owners[key]; ok && prior.mod != "" && e.mod != "" && prior.mod != e.mod {
		switch {
		case b.order.After(e.mod, prior.mod):
			b.overrides = append(b.overrides, fmt.Sprintf("install file %s: mod %q (%s) overrides mod %q (%s)", e.address, e.mod, e.file, prior.mod, prior.file))
		case b.order.After(prior.mod, e.mod):
			b.overrides = append(b.overrides, fmt.Sprintf("install file %s: mod %q (%s) overrides mod %q (%s)", e.address, prior.mod, prior.file, e.mod, e.file))
			return nil
		default:
			return mod.Source{Mod: e.mod, File: e.file}.Errorf("install file %s is served by mod %q (%s) and by mod %q, and neither loads after the other: add load-after to one of them",
				e.address, prior.mod, prior.file, e.mod)
		}
	}
	b.owners[key] = e
	target := b.fs
	if b.sounds != nil && strings.HasPrefix(key, sfxPrefix) {
		target = b.sounds
	}
	target.Overlay(e.address, e.read)
	return nil
}

// serveBytes serves a copy of data at address for mod's file.
func (b *modFiles) serveBytes(modID, file, address string, data []byte) error {
	return b.serve(modFileEntry{mod: modID, file: file, address: address, read: func() ([]byte, error) { return slices.Clone(data), nil }})
}

// serveInstallFiles checks each install file the mods carry against the
// install of a and serves it: a picture as it stands, a sound as it stands
// and a text in the install's own bytes. A file the game cannot use is
// refused, naming the mod, the file and the reason.
func (b *modFiles) serveInstallFiles(a *Archives, files []mod.InstallFile) error {
	selector := LanguageSelector(a.Containers)
	code := InstallTextCode(a.Containers, a.Game().Edition())
	for _, file := range files {
		installed, err := b.installed(file.Address)
		if errors.Is(err, fs.ErrNotExist) {
			return file.Errorf("the install holds no file %s; a mod shadows a file the archives %s hold", file.Address, strings.Join(shadowedArchives(), ", "))
		}
		if err != nil {
			return file.Errorf("the install's file %s: %v", file.Address, err)
		}
		data := file.Bytes
		switch file.Kind {
		case mod.FilePicture:
			err = checkModPicture(installed, data)
		case mod.FileSound:
			if _, derr := audio.DecodeWAV(data, audio.DeviceRate); derr != nil {
				err = fmt.Errorf("the game cannot decode this sound: %v", derr)
			}
		case mod.FileText:
			if r, dropped := ModTextDropped(data, selector); dropped {
				err = fmt.Errorf("the install's alphabet has no %q (%U)", r, r)
			}
			data = code.FileBytes(EncodeInstallTextFile(data, selector))
		default:
			err = fmt.Errorf("the game takes no install file of kind %q", file.Kind)
		}
		if err != nil {
			return file.Errorf("%v", err)
		}
		if err := b.serveBytes(file.Mod, file.File, file.Address, data); err != nil {
			return err
		}
	}
	return nil
}

// installed is the install's own bytes at address, out of the archives or
// the sound archive.
func (b *modFiles) installed(address string) ([]byte, error) {
	src := b.fs
	if strings.HasPrefix(mod.FoldAddress(address), sfxPrefix) {
		src = b.sounds
	}
	if src == nil {
		return nil, fs.ErrNotExist
	}
	return src.ReadFile(address)
}

// shadowedArchives names the archives a mod's install file may shadow.
func shadowedArchives() []string {
	var out []string
	for _, name := range append(RequiredArchives(), SfxArchive) {
		out = append(out, strings.TrimSuffix(name, ".res"))
	}
	return out
}

// checkModPicture refuses a picture the game cannot draw in place of the
// install's: one the install's own decoder for that file does not read, or of
// another size, which the screens that draw the picture at fixed places
// cannot take.
func checkModPicture(installed, modded []byte) error {
	depth := func(b []byte) int {
		if len(b) < 30 {
			return 0
		}
		return int(binary.LittleEndian.Uint16(b[28:]))
	}
	decode := func(b []byte, bits int) (image.Rectangle, error) {
		switch bits {
		case bmp.BitsPerPixel:
			pic, err := bmp.DecodeRGBA(b)
			if err != nil {
				return image.Rectangle{}, err
			}
			return pic.Bounds(), nil
		case bmp.IndexedBitsPerPixel:
			pic, err := bmp.DecodePaletted(b)
			if err != nil {
				return image.Rectangle{}, err
			}
			return pic.Bounds(), nil
		}
		return image.Rectangle{}, fmt.Errorf("bmp: %d bits per pixel; the game reads %d or %d", bits, bmp.IndexedBitsPerPixel, bmp.BitsPerPixel)
	}
	bits := depth(installed)
	want, err := decode(installed, bits)
	if err != nil {
		return fmt.Errorf("the install's picture at this path is not one the game decodes (%v)", err)
	}
	got, err := decode(modded, bits)
	if err != nil {
		return fmt.Errorf("the game cannot decode this picture as the install's %d-bit picture: %v", bits, err)
	}
	if got.Size() != want.Size() {
		return fmt.Errorf("the picture is %dx%d; the install's is %dx%d, the size the game draws it at", got.Dx(), got.Dy(), want.Dx(), want.Dy())
	}
	return nil
}
