#!/usr/bin/env python3
"""Package the native macOS runtime and public notices into a fresh directory."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import shutil
import stat
import subprocess
import sys
import zipfile


PUBLIC_FILES = (
    "README.md", "LICENSE", "THIRD_PARTY_NOTICES.md",
    "LICENSES/Apache-2.0.txt", "LICENSES/BSD-3-Clause.txt",
    "LICENSES/LGPL-2.1.txt", "LICENSES/lz4-BSD-3-Clause.txt",
    "LICENSES/starlark-BSD-3-Clause.txt",
)
INSTALL_ARCHIVES = {"main.res", "graphics.res", "scenario.res", "world.res", "movies.res"}


def run(arguments, cwd):
    result = subprocess.run(arguments, cwd=cwd, text=True, capture_output=True, check=False)
    if result.returncode:
        raise ValueError(f"Command failed: {arguments[0]}: {result.stderr.strip()}")
    return result.stdout.strip()


def fresh_output(value):
    output = Path(os.path.abspath(value))
    if os.path.lexists(output):
        raise ValueError("Output directory already exists; preserve it and choose a fresh path.")
    if not output.parent.is_dir():
        raise ValueError("Output parent must be an existing directory.")
    for directory in (output.parent, *output.parent.parents):
        if directory.is_symlink():
            raise ValueError("Output cannot traverse a symbolic link.")
        names = {entry.name.casefold() for entry in directory.iterdir() if entry.is_file()}
        if INSTALL_ARCHIVES <= names:
            raise ValueError("Output cannot be inside a game install.")
    return output


def binary_identity(binary, name, version, revision, arch):
    if binary.is_symlink() or not binary.is_file():
        raise ValueError(f"Missing regular executable: {binary}")
    expected = f"{name} {version} {revision[:12]}"
    if run([str(binary), "-version"], binary.parent) != expected:
        raise ValueError(f"Unexpected source stamp or VERSION: {binary}")
    metadata = run(["go", "version", "-m", str(binary)], binary.parent).splitlines()
    header = re.search(r": (go\S+)$", metadata[0]) if metadata else None
    if header is None:
        raise ValueError(f"Cannot read Go build identity: {binary}")
    settings = {}
    for line in metadata[1:]:
        kind, _, value = line.strip().partition("\t")
        if kind == "build":
            key, separator, setting = value.partition("=")
            if separator:
                settings[key] = setting
    for key, value in {"GOOS": "darwin", "GOARCH": arch, "CGO_ENABLED": "1", "-trimpath": "true"}.items():
        if settings.get(key) != value:
            raise ValueError(f"Missing build setting {key}={value} in {binary}")
    return header.group(1)


def package(arguments):
    repository = Path(__file__).resolve().parent.parent
    version = (repository / "cmd/againrom/VERSION").read_text(encoding="utf-8").strip()
    starter_version = (repository / "cmd/starter/VERSION").read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", arguments.tag) or arguments.tag != f"v{version}":
        raise ValueError("Tag must be vMAJOR.MINOR.PATCH and match cmd/againrom/VERSION.")
    head = run(["git", "rev-parse", "HEAD"], repository)
    if not re.fullmatch(r"[0-9a-f]{40}", arguments.revision) or arguments.revision != head:
        raise ValueError("Revision must be the exact checkout HEAD.")
    if run(["git", "status", "--porcelain", "--untracked-files=no"], repository):
        raise ValueError("Package source must have no tracked changes.")
    output = fresh_output(arguments.output_directory)
    native_arch = {"x86_64": "amd64", "arm64": "arm64"}.get(platform.machine())
    if sys.platform != "darwin" or native_arch != arguments.arch:
        raise ValueError("Packaging requires a native macOS host of the selected architecture.")
    binaries = Path(os.path.abspath(arguments.binaries_directory))
    versions = {"againrom": version, "starter": starter_version}
    go_versions = {
        binary_identity(binaries / name, name, program_version, arguments.revision, arguments.arch)
        for name, program_version in versions.items()
    }
    if len(go_versions) != 1:
        raise ValueError("The executables use different Go toolchains.")
    for name in PUBLIC_FILES:
        source = repository / name
        if source.is_symlink() or not source.is_file():
            raise ValueError(f"Missing regular public file: {name}")
    manifest = {
        "version": version, "starter_version": starter_version,
        "revision": arguments.revision, "tag": arguments.tag,
        "target": f"darwin/{arguments.arch}", "go": next(iter(go_versions)),
        "repository": "https://github.com/againrom/againrom-engine",
    }
    output.mkdir()
    stage = output / "againrom"
    (stage / "LICENSES").mkdir(parents=True)
    for name in PUBLIC_FILES:
        shutil.copyfile(repository / name, stage / name)
        (stage / name).chmod(0o644)
    for name in versions:
        shutil.copyfile(binaries / name, stage / name)
        (stage / name).chmod(0o755)
    manifest_path = stage / "BUILD-INFO.json"
    manifest_path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    manifest_path.chmod(0o644)
    archive_name = f"againrom-{version}-macos-{arguments.arch}.zip"
    archive_path = output / archive_name
    entries = sorted((*PUBLIC_FILES, *versions, "BUILD-INFO.json"))
    with zipfile.ZipFile(archive_path, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for name in entries:
            entry = zipfile.ZipInfo(f"againrom/{name}", (1980, 1, 1, 0, 0, 0))
            entry.create_system = 3
            mode = 0o755 if name in versions else 0o644
            entry.external_attr = (stat.S_IFREG | mode) << 16
            entry.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(entry, (stage / name).read_bytes(), compresslevel=9)
    digest = hashlib.sha256(archive_path.read_bytes()).hexdigest()
    checksum_path = output / (archive_name + ".sha256")
    checksum_path.write_text(f"{digest}  {archive_name}\n", encoding="utf-8")
    return {"zip": str(archive_path), "checksum": str(checksum_path), "manifest": str(manifest_path), "sha256": digest}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binaries-directory", required=True)
    parser.add_argument("--output-directory", required=True)
    parser.add_argument("--revision", required=True)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--arch", required=True, choices=("amd64", "arm64"))
    try:
        print(json.dumps(package(parser.parse_args()), sort_keys=True))
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        print(f"macos-package: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
