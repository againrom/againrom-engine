#!/usr/bin/env python3
"""Check Linux package contents and no-display execution; --unit works on every host."""

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import stat
import struct
import subprocess
import sys
import tarfile
from unittest import mock
import zlib

sys.dont_write_bytecode = True
SPEC = importlib.util.spec_from_file_location("linux_package", Path(__file__).with_name("package-linux.py"))
PACKAGE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PACKAGE)
EXPECTED = (
    "BUILD-INFO.json", "LICENSE", "LICENSES/Apache-2.0.txt", "LICENSES/BSD-3-Clause.txt",
    "LICENSES/LGPL-2.1.txt", "LICENSES/ebiten-NOTICE.md", "LICENSES/lz4-BSD-3-Clause.txt",
    "LICENSES/starlark-BSD-3-Clause.txt", "README.md", "THIRD_PARTY_NOTICES.md", "againrom", "mapedit", "starter",
)
IMAGE_EXTRA = (".DirIcon", "AppRun", "LICENSES/AppImage-runtime.txt", "againrom.desktop", "againrom.png")


def check(condition, message):
    if not condition:
        raise ValueError(message)


def refused(action):
    try:
        action()
    except ValueError:
        return
    raise ValueError("Invalid input was accepted.")


def inventory(directory, include_directories=False):
    return {path.relative_to(directory).as_posix() + ("/" if path.is_dir() else ""):
            (stat.S_IMODE(path.stat().st_mode), PACKAGE.digest(path) if path.is_file() else None)
            for path in directory.rglob("*") if path.is_file() or include_directories and path.is_dir()}


def check_unchanged(before, after, message):
    changes = {
        "added": sorted(after.keys() - before.keys()),
        "changed": sorted(name for name in before.keys() & after.keys() if before[name] != after[name]),
        "removed": sorted(before.keys() - after.keys()),
    }
    detail = "; ".join(f"{kind}: {', '.join(paths)}" for kind, paths in changes.items() if paths)
    check(not detail, f"{message} {detail}")


def check_png(data):
    check(data[:8] == b"\x89PNG\r\n\x1a\n", "Icon has no PNG signature.")
    offset, pixels, kinds = 8, b"", []
    while offset < len(data):
        size = struct.unpack(">I", data[offset:offset + 4])[0]
        kind, payload = data[offset + 4:offset + 8], data[offset + 8:offset + 8 + size]
        crc = struct.unpack(">I", data[offset + 8 + size:offset + 12 + size])[0]
        check(crc == zlib.crc32(kind + payload), "Icon chunk CRC differs.")
        kinds.append(kind)
        if kind == b"IHDR":
            check(struct.unpack(">IIBBBBB", payload) == (256, 256, 8, 6, 0, 0, 0), "Icon dimensions differ.")
        if kind == b"IDAT":
            pixels += payload
        offset += 12 + size
    check(kinds[0] == b"IHDR" and kinds[-1] == b"IEND" and b"IDAT" in kinds, "Icon chunks are incomplete.")
    check(len(zlib.decompress(pixels)) == 256 * (1 + 256 * 4), "Icon pixels are incomplete.")
    check(hashlib.sha256(data).hexdigest() == "f1dc0fb248da6c9d5813081ac61a5a31d4464290d78a08667c0cb3476910bb58", "Owner icon SHA256 differs.")


def check_tar(path, epoch):
    with tarfile.open(path, "r:gz") as archive:
        check(archive.getnames() == [f"againrom/{name}" for name in EXPECTED], "Tar members differ from the public allowlist.")
        for entry in archive.getmembers():
            name = entry.name.removeprefix("againrom/")
            mode = 0o755 if name in ("againrom", "starter", "mapedit") else 0o644
            check(entry.isfile() and entry.mode == mode, f"Tar type or executable bits differ: {entry.name}")
            check(entry.uid == entry.gid == 0 and entry.uname == entry.gname == "", "Tar leaks an owner identity.")
            check(entry.mtime == epoch, "Tar timestamps differ from the commit.")


def unit(evidence):
    check(set(PACKAGE.COMMON_FILES) == set(EXPECTED), "Packaging public allowlist differs.")
    check(set(PACKAGE.APPIMAGE_FILES) == set(IMAGE_EXTRA), "AppImage allowlist differs.")
    for script in ("package-macos.py", "package-windows.ps1"):
        text = (PACKAGE.REPOSITORY / "scripts" / script).read_text(encoding="utf-8")
        for name in PACKAGE.PUBLIC_FILES:
            check(name in text, f"Linux notice is absent from {script}: {name}")
    revision, version = "a" * 40, "0.135.0"
    PACKAGE.validate_identity(f"v{version}", revision, revision, version)
    for tag in ("main", "v00.135.0", "v0.135.0-rc.1", "v999.999.999"):
        refused(lambda: PACKAGE.validate_identity(tag, revision, revision, version))
    for candidate in ("0" * 40, "bad"):
        refused(lambda: PACKAGE.validate_identity(f"v{version}", candidate, revision, version))
    refused(lambda: PACKAGE.fresh_output(evidence))
    refused(lambda: PACKAGE.fresh_output(evidence / "missing/output"))
    install = evidence / "install-fixture"
    (install / "child").mkdir(parents=True)
    for name in ("MAIN.RES", "Graphics.Res", "Scenario.res", "WORLD.RES", "movies.res"):
        (install / name).write_bytes(b"")
    refused(lambda: PACKAGE.fresh_output(install / "output"))
    refused(lambda: PACKAGE.fresh_output(install / "child/output"))
    binaries = evidence / "inputs"
    binaries.mkdir()
    for name in PACKAGE.PROGRAMS:
        (binaries / name).write_bytes(f"public fixture {name}".encode())
    for name in ("starter.ini", "saves/private.sav", "mods/private/mod.toml", "main.res", "credential.txt"):
        target = binaries / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(b"private sentinel")
    before = inventory(binaries)
    stage = evidence / "stage"
    PACKAGE.stage_public(binaries, stage, {"fixture": True})
    check_unchanged(before, inventory(binaries), "Staging wrote into input files.")
    check(sorted(inventory(stage)) == list(EXPECTED), "Private files entered staging.")
    tar, repeat = evidence / "fixture.tar.gz", evidence / "repeat.tar.gz"
    PACKAGE.write_tar(stage, tar, 123456)
    PACKAGE.write_tar(stage, repeat, 123456)
    check_tar(tar, 123456)
    check(tar.read_bytes() == repeat.read_bytes(), "Tar bytes are not reproducible.")
    refused_existing = False
    try:
        PACKAGE.write_tar(stage, tar, 123456)
    except FileExistsError:
        refused_existing = True
    check(refused_existing, "Existing tar was overwritten.")
    for name in PACKAGE.PUBLIC_FILES:
        check((stage / name).read_bytes() == (PACKAGE.REPOSITORY / name).read_bytes(), f"Notice changed: {name}")
    appdir = evidence / "fixture.AppDir"
    PACKAGE.stage_appimage(stage, appdir)
    check(sorted(inventory(appdir)) == sorted((*EXPECTED, *IMAGE_EXTRA)), "AppDir members differ.")
    check_png((appdir / "againrom.png").read_bytes())
    check((appdir / ".DirIcon").read_bytes() == (PACKAGE.REPOSITORY / "internal/appicon/againrom.png").read_bytes(), "AppImage icon differs from the owner input.")
    fake_tool = evidence / "fake-tool"
    fake_tool.write_bytes(b"fixture tool")
    good = {"sha256": hashlib.sha256(b"fixture tool").hexdigest()}
    PACKAGE.verify_tool(fake_tool, good)
    refused(lambda: PACKAGE.verify_tool(fake_tool, {"sha256": "0" * 64}))
    refused(lambda: PACKAGE.verify_tool(fake_tool, {"sha256": ""}))
    for pin in PACKAGE.tool_pins().values():
        check(len(bytes.fromhex(pin["sha256"])) == 32 and len(pin["revision"]) == 40, "A tool is not pinned.")
        check(pin["url"].startswith("https://github.com/AppImage/") and "/continuous/" not in pin["url"], "Tool URL is mutable.")
    elf = bytearray(512)
    elf[:6] = b"\x7fELF\x02\x01"
    struct.pack_into("<Q", elf, 40, 64)
    struct.pack_into("<HHH", elf, 58, 64, 3, 1)
    struct.pack_into("<QQ", elf, 64 + 64 + 24, 256, 13)
    struct.pack_into("<I", elf, 64 + 128, 1)
    struct.pack_into("<QQ", elf, 64 + 128 + 24, 300, 16)
    elf[256:269] = b"\0.digest_md5\0\0"
    patched = bytearray(elf)
    patched[300:316] = b"x" * 16
    check_embedded_runtime(patched, elf)
    patched[320] = 1
    refused(lambda: check_embedded_runtime(patched, elf))
    probe_home, tool_config = evidence / "version-home", evidence / "tool-config"
    probe_config = probe_home / "config"
    probe_config.mkdir(parents=True)
    tool_config.mkdir()
    environment = {"HOME": str(probe_home), "XDG_CONFIG_HOME": str(probe_config)}
    probe_binary = Path(sys.executable).resolve()
    profile_before = inventory(probe_home, include_directories=True)
    calls = []

    def version_command(command, cwd, environment=None):
        calls.append(command)
        if command[0] == "go":
            config = Path(environment["XDG_CONFIG_HOME"]) if environment is not None else tool_config
            counter = config / "go/telemetry/local/fixture.v1.count"
            counter.parent.mkdir(parents=True)
            counter.write_bytes(b"tool counter")
            return "fixture: go1.26.1\n\tbuild\tGOOS=linux\n\tbuild\tGOARCH=amd64\n\tbuild\tCGO_ENABLED=0\n\tbuild\t-trimpath=true"
        check(command == [str(probe_binary), "-version"] and environment == {
            "HOME": str(probe_home), "XDG_CONFIG_HOME": str(probe_config)}, "Program version environment differs.")
        return f"againrom {version} {revision[:12]}"

    with mock.patch.object(PACKAGE, "run", side_effect=version_command):
        check(PACKAGE.binary_identity(probe_binary, "againrom", version, revision, environment) == "go1.26.1", "Toolchain identity differs.")
    check_unchanged(profile_before, inventory(probe_home, include_directories=True), "Go metadata inspection contaminated the program profile.")
    check((tool_config / "go/telemetry/local/fixture.v1.count").is_file(), "Metadata tool did not use its own configuration.")
    check(len(calls) == 2 and calls[1] == ["go", "version", "-m", str(probe_binary)], "Version or Go metadata command was lost.")
    directory_before = inventory(probe_home, include_directories=True)
    (probe_home / "unexpected-empty-folder").mkdir()
    try:
        check_unchanged(directory_before, inventory(probe_home, include_directories=True), "Version wrote profile.")
    except ValueError as error:
        check("added: unexpected-empty-folder/" in str(error), "Empty-directory diagnostic lost its path.")
    else:
        raise ValueError("A version command's empty directory write was accepted.")
    try:
        check_unchanged({"settings.ini": (0o644, "old"), "private.sav": (0o644, "save")},
                        {"settings.ini": (0o644, "new"), "new.dat": (0o644, "data")}, "Version wrote profile.")
    except ValueError as error:
        check(all(text in str(error) for text in ("added: new.dat", "changed: settings.ini", "removed: private.sav")), "Profile diagnostic lost changed paths.")
    else:
        raise ValueError("A version command's file write was accepted.")
    print("linux-package unit: allowlists, public notice bytes, private sentinels, identity refusals, output fences, "
          "tar metadata and reproducibility, PNG structure, tool checksums, missing-pin refusal, metadata isolation "
          "and profile-write diagnostics PASS")


def check_runtime(runtime, manifest, environment):
    before = inventory(runtime, include_directories=True)
    for name in PACKAGE.PUBLIC_FILES:
        check((runtime / name).read_bytes() == (PACKAGE.REPOSITORY / name).read_bytes(), f"Notice bytes differ: {name}")
    for name in PACKAGE.PROGRAMS:
        profile_before = inventory(Path(environment["HOME"]), include_directories=True)
        version = manifest["starter_version"] if name == "starter" else manifest["version"]
        check(stat.S_IMODE((runtime / name).stat().st_mode) == 0o755, f"Not executable: {name}")
        check(PACKAGE.binary_identity(runtime / name, name, version, manifest["revision"], environment) == manifest["go"], "Go version differs.")
        check_unchanged(profile_before, inventory(Path(environment["HOME"]), include_directories=True),
                        f"{runtime.name}/{name} version inspection wrote a user profile.")
        output = PACKAGE.run([str(runtime / name), "-version"], runtime, environment)
        check(output == f"{name} {version} {manifest['revision'][:12]}", "No-display version differs.")
        check_unchanged(profile_before, inventory(Path(environment["HOME"]), include_directories=True),
                        f"{runtime.name}/{name} -version wrote a user profile.")
    check_unchanged(before, inventory(runtime, include_directories=True), "Version inspection wrote into runtime.")


def check_embedded_runtime(image, runtime):
    check(runtime[:6] == b"\x7fELF\x02\x01", "Runtime is not little-endian ELF64.")
    table = struct.unpack_from("<Q", runtime, 40)[0]
    size, count, names_index = struct.unpack_from("<HHH", runtime, 58)
    check(size == 64 and names_index < count and table + size * count <= len(runtime), "Invalid ELF section table.")
    names_offset, names_size = struct.unpack_from("<QQ", runtime, table + size * names_index + 24)
    names = runtime[names_offset:names_offset + names_size]
    for index in range(count):
        section = table + index * size
        name = struct.unpack_from("<I", runtime, section)[0]
        if names[name:].split(b"\0", 1)[0] == b".digest_md5":
            offset, length = struct.unpack_from("<QQ", runtime, section + 24)
            check(length >= 16 and offset + length <= len(runtime), "Invalid AppImage digest section.")
            check(image[:offset] == runtime[:offset] and image[offset + 16:len(runtime)] == runtime[offset + 16:],
                  "Embedded runtime differs outside its AppImage MD5 field.")
            return
    raise ValueError("Runtime lacks the AppImage MD5 section.")


def check_launcher(evidence, environment):
    appdir = evidence / "read-only AppDir ü"
    appdir.mkdir()
    launcher = appdir / "AppRun"
    launcher.write_text(PACKAGE.APP_RUN, encoding="utf-8", newline="\n")
    launcher.chmod(0o555)
    starter = appdir / "starter"
    starter.write_text('#!/bin/sh\nif [ "${1:-}" = -version ]; then printf "fixture version\\n"; exit; fi\npwd\nprintf "%s\\n" "$AGAINROM_PROFILE" "$@"\n',
                       encoding="utf-8", newline="\n")
    starter.chmod(0o555)
    appdir.chmod(0o555)
    before = inventory(appdir, include_directories=True)
    home_before = inventory(Path(environment["HOME"]), include_directories=True)
    check(PACKAGE.run([str(launcher), "-version"], evidence, environment) == "fixture version", "AppRun version route failed.")
    check_unchanged(home_before, inventory(Path(environment["HOME"]), include_directories=True), "AppRun version wrote a profile.")
    profile = Path(environment["XDG_CONFIG_HOME"]) / "Againrom"
    expected = [str(profile), str(profile), "-ini", str(profile / "starter.ini"), "argument with spaces ü"]
    check(PACKAGE.run([str(launcher), "argument with spaces ü"], evidence, environment).splitlines() == expected, "AppRun arguments or working directory differ.")
    ini = profile / "starter.ini"
    check(ini.read_text() == f"[mods]\ndir = {profile}/mods\n\n[options]\nsaves = {profile}/saves\n", "AppRun defaults do not select the user profile.")
    check((profile / "mods").is_dir() and (profile / "saves").is_dir(), "Profile folders are missing.")
    ini.write_text(ini.read_text() + "\n; retained player setting\n")
    retained = ini.read_bytes()
    PACKAGE.run([str(launcher)], evidence, environment)
    check(ini.read_bytes() == retained, "AppRun replaced existing settings.")
    check_unchanged(before, inventory(appdir, include_directories=True), "AppRun wrote inside its read-only directory.")


def native(arguments, evidence):
    PACKAGE.native_linux()
    directory = Path(arguments.package_directory).absolute()
    repository = PACKAGE.REPOSITORY
    version = (repository / "cmd/againrom/VERSION").read_text().strip()
    revision = PACKAGE.run(["git", "rev-parse", "HEAD"], repository)
    epoch = int(PACKAGE.run(["git", "show", "-s", "--format=%ct", "HEAD"], repository))
    tar = directory / f"againrom-{version}-linux-x86_64.tar.gz"
    image = directory / f"againrom-{version}-linux-x86_64.AppImage"
    check_tar(tar, epoch)
    environment = os.environ.copy()
    environment.pop("DISPLAY", None)
    environment.pop("WAYLAND_DISPLAY", None)
    environment["HOME"] = str(evidence / "home ü")
    environment["XDG_CONFIG_HOME"] = str(Path(environment["HOME"]) / "config ü")
    Path(environment["XDG_CONFIG_HOME"]).mkdir(parents=True)
    config_before = inventory(Path(environment["HOME"]), include_directories=True)
    tar_root = evidence / "tar"
    with tarfile.open(tar, "r:gz") as archive:
        manifest = json.load(archive.extractfile("againrom/BUILD-INFO.json"))
        for entry in archive:
            target = tar_root / entry.name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(archive.extractfile(entry).read())
            target.chmod(entry.mode)
    PACKAGE.validate_identity(manifest["tag"], manifest["revision"], revision, version)
    check(manifest["target"] == "linux/amd64" and manifest["version"] == version and
          manifest["starter_version"] == (repository / "cmd/starter/VERSION").read_text().strip() and
          manifest["linux_tools"] == PACKAGE.tool_pins(), "Manifest identity differs.")
    check_runtime(tar_root / "againrom", manifest, environment)
    check(stat.S_IMODE(image.stat().st_mode) == 0o755, "AppImage is not executable.")
    extracted = evidence / "image"
    extracted.mkdir()
    PACKAGE.run([str(image), "--appimage-extract"], extracted, environment)
    check_unchanged(config_before, inventory(Path(environment["HOME"]), include_directories=True),
                    "AppImage --appimage-extract wrote a user profile.")
    runtime = extracted / "squashfs-root"
    check(sorted(inventory(runtime)) == sorted((*EXPECTED, *IMAGE_EXTRA)), "Extracted AppImage members differ.")
    check(sorted(path.relative_to(runtime).as_posix() for path in runtime.rglob("*") if path.is_dir()) == ["LICENSES"], "Unexpected AppImage directory.")
    check(not any(path.is_symlink() for path in runtime.rglob("*")), "Unexpected AppImage symlink.")
    check(json.loads((runtime / "BUILD-INFO.json").read_text()) == manifest, "AppImage manifest differs from tar.")
    check_runtime(runtime, manifest, environment)
    for name in IMAGE_EXTRA:
        mode = 0o755 if name == "AppRun" else 0o644
        check(stat.S_IMODE((runtime / name).stat().st_mode) == mode, f"AppImage permissions differ: {name}")
    check((runtime / "AppRun").read_text() == PACKAGE.APP_RUN, "AppRun bytes differ.")
    check((runtime / "LICENSES/AppImage-runtime.txt").read_bytes() == (repository / "LICENSES/AppImage-runtime.txt").read_bytes(), "Runtime notices differ.")
    check_png((runtime / "againrom.png").read_bytes())
    check((runtime / ".DirIcon").read_bytes() == (runtime / "againrom.png").read_bytes(), "AppImage icons differ.")
    tools = Path(arguments.tools_directory).absolute()
    PACKAGE.verify_tool(tools / "runtime", PACKAGE.tool_pins()["runtime"])
    runtime_bytes = (tools / "runtime").read_bytes()
    with image.open("rb") as source:
        check_embedded_runtime(source.read(len(runtime_bytes)), runtime_bytes)
    receipt = json.loads((directory / "LINUX-RELEASE.json").read_text())
    check(receipt == {"manifest": manifest, "sha256": {tar.name: PACKAGE.digest(tar), image.name: PACKAGE.digest(image)}}, "Publication receipt differs.")
    check(PACKAGE.run([str(runtime / "AppRun"), "-version"], runtime, environment) ==
          f"starter {manifest['starter_version']} {revision[:12]}", "AppImage entry version differs.")
    check_unchanged(config_before, inventory(Path(environment["HOME"]), include_directories=True),
                    "AppRun -version wrote a user profile.")
    check_launcher(evidence, environment)
    target = evidence / "symlink-target"
    target.mkdir()
    link = evidence / "symlink-parent"
    link.symlink_to(target, target_is_directory=True)
    refused(lambda: PACKAGE.fresh_output(link / "output"))
    dangling = evidence / "dangling"
    dangling.symlink_to(evidence / "absent", target_is_directory=True)
    refused(lambda: PACKAGE.fresh_output(dangling))
    inputs = evidence / "private-inputs"
    inputs.mkdir()
    for name in PACKAGE.PROGRAMS:
        shutil.copy2(Path(arguments.binaries_directory) / name, inputs / name)
    for name in ("starter.ini", "options.txt", "saves/private.sav", "mods/private/mod.toml", "main.res"):
        target = inputs / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(b"private sentinel")
    input_before = inventory(inputs)
    defaults = dict(binaries_directory=str(inputs), tools_directory=str(tools),
                    revision=revision, tag=manifest["tag"])
    repeat = PACKAGE.package(argparse.Namespace(**defaults, output_directory=str(evidence / "repeat")))
    check_unchanged(input_before, inventory(inputs), "Native package inspection wrote into inputs.")
    check(PACKAGE.digest(tar) == PACKAGE.digest(repeat["tar"]), "Repeated tar bytes differ.")
    check(PACKAGE.digest(image) == PACKAGE.digest(repeat["appimage"]), "Repeated AppImage bytes differ.")
    refused(lambda: PACKAGE.package(argparse.Namespace(**defaults, output_directory=str(evidence / "repeat"))))
    for index, (key, value) in enumerate((("tag", "main"), ("tag", "v999.999.999"), ("revision", "0" * 40))):
        invalid = dict(defaults)
        invalid[key] = value
        output = evidence / f"invalid-native-{index}"
        refused(lambda: PACKAGE.package(argparse.Namespace(**invalid, output_directory=str(output))))
        check(not output.exists(), "Invalid identity wrote native output.")
    missing = evidence / "without-editor"
    missing.mkdir()
    for name in ("againrom", "starter"):
        shutil.copy2(inputs / name, missing / name)
    invalid = dict(defaults, binaries_directory=str(missing))
    output = evidence / "missing-editor-package"
    refused(lambda: PACKAGE.package(argparse.Namespace(**invalid, output_directory=str(output))))
    check(not output.exists(), "Missing-editor refusal wrote native output.")
    print("linux-package native: 13 tar files, 18 AppImage files; notices, stamps, no-display versions, executable bits, "
          "embedded runtime, read-only launcher, user settings retention, symlink refusals and repeated artifact bytes PASS")
    print(json.dumps({tar.name: PACKAGE.digest(tar), image.name: PACKAGE.digest(image)}, sort_keys=True))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--unit", action="store_true")
    parser.add_argument("--evidence-directory", required=True)
    for name in ("package-directory", "tools-directory", "binaries-directory"):
        parser.add_argument("--" + name)
    arguments = parser.parse_args()
    try:
        evidence = PACKAGE.fresh_output(arguments.evidence_directory)
        evidence.mkdir()
        if arguments.unit:
            unit(evidence)
        else:
            check(all((arguments.package_directory, arguments.tools_directory, arguments.binaries_directory)), "Native checks require package, tools and binaries directories.")
            native(arguments, evidence)
    except (OSError, ValueError, subprocess.SubprocessError, tarfile.TarError) as error:
        print(f"linux-package-check: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
