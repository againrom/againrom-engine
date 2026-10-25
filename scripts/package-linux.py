#!/usr/bin/env python3
"""Build the Linux tar.gz and AppImage from native, stamped executables."""

import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import platform
import re
import shutil
import subprocess
import sys
import tarfile
import urllib.request


REPOSITORY = Path(__file__).resolve().parent.parent
PUBLIC_FILES = (
    "README.md", "LICENSE", "THIRD_PARTY_NOTICES.md",
    "LICENSES/Apache-2.0.txt", "LICENSES/BSD-3-Clause.txt",
    "LICENSES/LGPL-2.1.txt", "LICENSES/lz4-BSD-3-Clause.txt",
    "LICENSES/starlark-BSD-3-Clause.txt", "LICENSES/ebiten-NOTICE.md",
)
PROGRAMS = ("againrom", "starter", "mapedit")
COMMON_FILES = (*PUBLIC_FILES, *PROGRAMS, "BUILD-INFO.json")
APPIMAGE_FILES = ("AppRun", "againrom.desktop", "againrom.png", ".DirIcon", "LICENSES/AppImage-runtime.txt")
INSTALL_ARCHIVES = {"main.res", "graphics.res", "scenario.res", "world.res", "movies.res"}
APP_RUN = '''#!/bin/sh
set -eu
appdir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
if [ "${1:-}" = -version ]; then
    exec "$appdir/starter" "$@"
fi
case "${XDG_CONFIG_HOME:-}" in
    /*) config=$XDG_CONFIG_HOME ;;
    *) config=${HOME:?HOME or an absolute XDG_CONFIG_HOME is required}/.config ;;
esac
profile=$config/Againrom
export AGAINROM_PROFILE="$profile"
umask 077
mkdir -p "$profile/mods" "$profile/saves"
if [ ! -e "$profile/starter.ini" ]; then
    (set -C; printf '[mods]\\ndir = %s\\n\\n[options]\\nsaves = %s\\n' \\
        "$profile/mods" "$profile/saves" > "$profile/starter.ini")
fi
cd -- "$profile"
exec "$appdir/starter" -ini "$profile/starter.ini" "$@"
'''
DESKTOP = '''[Desktop Entry]
Type=Application
Name=Againrom
Exec=AppRun
Icon=againrom
Categories=Game;StrategyGame;
Terminal=false
'''


def run(arguments, cwd, environment=None):
    result = subprocess.run(arguments, cwd=cwd, env=environment, text=True,
                            capture_output=True, check=False, timeout=180)
    if result.returncode:
        raise ValueError(f"Command failed ({result.returncode}): {arguments[0]}: {result.stderr.strip()}")
    return result.stdout.strip()


def digest(path):
    with Path(path).open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def fresh_output(value):
    output = Path(os.path.abspath(value))
    if os.path.lexists(output):
        raise ValueError("Output already exists; choose a fresh directory.")
    if not output.parent.is_dir():
        raise ValueError("Output parent must exist.")
    for parent in (output.parent, *output.parent.parents):
        if parent.is_symlink():
            raise ValueError("Output cannot traverse a symbolic link.")
        names = {entry.name.casefold() for entry in parent.iterdir() if entry.is_file()}
        if INSTALL_ARCHIVES <= names:
            raise ValueError("Output cannot be inside a game install.")
    return output


def tool_pins():
    return json.loads((REPOSITORY / "scripts/linux-tools.json").read_text(encoding="utf-8"))


def verify_tool(path, pin):
    if path.is_symlink() or not path.is_file():
        raise ValueError(f"Missing regular tool: {path}")
    measured = digest(path)
    if not re.fullmatch(r"[0-9a-f]{64}", pin["sha256"]) or measured != pin["sha256"]:
        raise ValueError(f"Tool {path.name}: measured SHA256={measured}; pin={pin['sha256']!r}. Pin before use.")


def native_linux():
    if sys.platform != "linux" or platform.machine() != "x86_64":
        raise ValueError("A native Linux x86_64 host is required.")


def fetch_tools(output_directory):
    native_linux()
    output = fresh_output(output_directory)
    output.mkdir()
    for name, pin in tool_pins().items():
        target = output / name
        with urllib.request.urlopen(pin["url"], timeout=120) as source, target.open("xb") as destination:
            shutil.copyfileobj(source, destination)
        print(f"{name} {pin['version']} measured SHA256={digest(target)}", flush=True)
        verify_tool(target, pin)
        target.chmod(0o755)


def validate_identity(tag, revision, head, version):
    if not re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", tag) or tag != f"v{version}":
        raise ValueError("Tag must be vMAJOR.MINOR.PATCH and match VERSION.")
    if not re.fullmatch(r"[0-9a-f]{40}", revision) or revision != head:
        raise ValueError("Revision must be the exact checkout HEAD.")


def binary_identity(binary, name, version, revision, environment=None):
    if binary.is_symlink() or not binary.is_file() or not binary.stat().st_mode & 0o111:
        raise ValueError(f"Missing regular executable: {binary}")
    if run([str(binary), "-version"], binary.parent, environment) != f"{name} {version} {revision[:12]}":
        raise ValueError(f"Unexpected version or source stamp: {binary}")
    # Go records tool telemetry; only the program inherits the profile under test.
    metadata = run(["go", "version", "-m", str(binary)], binary.parent).splitlines()
    header = re.search(r": (go\S+)$", metadata[0]) if metadata else None
    if header is None:
        raise ValueError(f"Missing Go identity: {binary}")
    for setting in ("GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "-trimpath=true"):
        if not any(line.strip() == f"build\t{setting}" for line in metadata):
            raise ValueError(f"Missing {setting}: {binary}")
    return header.group(1)


def stage_public(binaries, stage, manifest):
    (stage / "LICENSES").mkdir(parents=True)
    for name in PUBLIC_FILES:
        source = REPOSITORY / name
        if source.is_symlink() or not source.is_file():
            raise ValueError(f"Missing regular public file: {name}")
        shutil.copyfile(source, stage / name)
        (stage / name).chmod(0o644)
    for name in PROGRAMS:
        shutil.copyfile(binaries / name, stage / name)
        (stage / name).chmod(0o755)
    (stage / "BUILD-INFO.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8", newline="\n")
    (stage / "BUILD-INFO.json").chmod(0o644)


def write_tar(stage, target, epoch):
    with target.open("xb") as destination, gzip.GzipFile(filename="", mode="wb", fileobj=destination, mtime=0) as compressed:
        with tarfile.open(fileobj=compressed, mode="w", format=tarfile.USTAR_FORMAT) as archive:
            for name in sorted(COMMON_FILES):
                data = (stage / name).read_bytes()
                entry = tarfile.TarInfo(f"againrom/{name}")
                entry.size, entry.mtime = len(data), epoch
                entry.mode = 0o755 if name in PROGRAMS else 0o644
                archive.addfile(entry, io.BytesIO(data))


def stage_appimage(stage, appdir):
    shutil.copytree(stage, appdir)
    (appdir / "AppRun").write_text(APP_RUN, encoding="utf-8", newline="\n")
    (appdir / "AppRun").chmod(0o755)
    (appdir / "againrom.desktop").write_text(DESKTOP, encoding="utf-8", newline="\n")
    (appdir / "againrom.desktop").chmod(0o644)
    for name in ("againrom.png", ".DirIcon"):
        shutil.copyfile(REPOSITORY / "internal/appicon/againrom.png", appdir / name)
        (appdir / name).chmod(0o644)
    shutil.copyfile(REPOSITORY / "LICENSES/AppImage-runtime.txt", appdir / "LICENSES/AppImage-runtime.txt")
    (appdir / "LICENSES/AppImage-runtime.txt").chmod(0o644)


def package(arguments):
    native_linux()
    version = (REPOSITORY / "cmd/againrom/VERSION").read_text().strip()
    starter_version = (REPOSITORY / "cmd/starter/VERSION").read_text().strip()
    head = run(["git", "rev-parse", "HEAD"], REPOSITORY)
    validate_identity(arguments.tag, arguments.revision, head, version)
    if run(["git", "status", "--porcelain", "--untracked-files=no"], REPOSITORY):
        raise ValueError("Package source has tracked changes.")
    output = fresh_output(arguments.output_directory)
    binaries = Path(os.path.abspath(arguments.binaries_directory))
    tools = Path(os.path.abspath(arguments.tools_directory))
    pins = tool_pins()
    for name, pin in pins.items():
        verify_tool(tools / name, pin)
    versions = {"againrom": version, "starter": starter_version, "mapedit": version}
    go_versions = {binary_identity(binaries / name, name, value, head) for name, value in versions.items()}
    if len(go_versions) != 1:
        raise ValueError("Programs use different Go toolchains.")
    epoch = int(run(["git", "show", "-s", "--format=%ct", "HEAD"], REPOSITORY))
    manifest = {"version": version, "starter_version": starter_version, "revision": head,
                "tag": arguments.tag, "target": "linux/amd64", "go": next(iter(go_versions)),
                "repository": "https://github.com/againrom/againrom-engine", "linux_tools": pins}
    output.mkdir()
    stage = output / "againrom"
    stage_public(binaries, stage, manifest)
    tar = output / f"againrom-{version}-linux-x86_64.tar.gz"
    write_tar(stage, tar, epoch)
    appdir = output / "Againrom.AppDir"
    stage_appimage(stage, appdir)
    for entry in (appdir, *appdir.rglob("*")):
        os.utime(entry, (epoch, epoch))
    unpack = output / "build-tool"
    unpack.mkdir()
    run([str(tools / "appimagetool"), "--appimage-extract"], unpack)
    image = output / f"againrom-{version}-linux-x86_64.AppImage"
    environment = os.environ.copy()
    environment.update(ARCH="x86_64", SOURCE_DATE_EPOCH=str(epoch))
    run([str(unpack / "squashfs-root/AppRun"), "--runtime-file", str(tools / "runtime"),
         "--no-appstream", "--comp", "zstd", "--mksquashfs-opt=-processors", "--mksquashfs-opt=2",
         str(appdir), str(image)], output, environment)
    image.chmod(0o755)
    receipt = output / "LINUX-RELEASE.json"
    hashes = {tar.name: digest(tar), image.name: digest(image)}
    receipt.write_text(json.dumps({"manifest": manifest, "sha256": hashes}, indent=2) + "\n", encoding="utf-8", newline="\n")
    return {"tar": str(tar), "appimage": str(image), "manifest": str(stage / "BUILD-INFO.json"),
            "receipt": str(receipt), "sha256": hashes}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    fetch = commands.add_parser("fetch-tools")
    fetch.add_argument("--output-directory", required=True)
    build = commands.add_parser("package")
    for name in ("binaries-directory", "tools-directory", "output-directory", "revision", "tag"):
        build.add_argument("--" + name, required=True)
    arguments = parser.parse_args()
    try:
        if arguments.command == "fetch-tools":
            fetch_tools(arguments.output_directory)
        else:
            print(json.dumps(package(arguments), sort_keys=True))
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        print(f"linux-package: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
