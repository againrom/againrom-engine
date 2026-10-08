#!/usr/bin/env python3
"""Verify the macOS package boundary with native release executables."""

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


def assert_true(condition, message):
    if not condition:
        raise ValueError(message)


def run(command, cwd, environment):
    result = subprocess.run(command, cwd=cwd, env=environment, text=True, capture_output=True, check=False)
    assert_true(result.returncode == 0, f"Command failed: {command[0]}: {result.stderr.strip()}")
    return result.stdout.strip()


def inventory(directory):
    return {
        str(path.relative_to(directory)): (stat.S_IMODE(path.stat().st_mode), hashlib.sha256(path.read_bytes()).hexdigest())
        for path in directory.rglob("*") if path.is_file()
    }, sorted(str(path.relative_to(directory)) for path in directory.rglob("*") if path.is_dir())


def fresh_evidence(value):
    evidence = Path(os.path.abspath(value))
    assert_true(not os.path.lexists(evidence), "Evidence directory already exists; choose a fresh path.")
    assert_true(evidence.parent.is_dir(), "Evidence parent must be an existing directory.")
    archives = {"main.res", "graphics.res", "scenario.res", "world.res", "movies.res"}
    for parent in (evidence.parent, *evidence.parent.parents):
        assert_true(not parent.is_symlink(), "Evidence cannot traverse a symbolic link.")
        names = {entry.name.casefold() for entry in parent.iterdir() if entry.is_file()}
        assert_true(not archives <= names, "Evidence cannot be inside a game install.")
    return evidence


def verify(arguments):
    assert_true(sys.platform == "darwin", "The witness requires a native macOS host.")
    native_arch = {"x86_64": "amd64", "arm64": "arm64"}.get(platform.machine())
    assert_true(native_arch == arguments.arch, "The witness must run on the selected native architecture.")
    repository = Path(__file__).resolve().parent.parent
    evidence = fresh_evidence(arguments.evidence_directory)
    evidence.mkdir()
    inputs = evidence / "inputs"
    inputs.mkdir()
    binaries = Path(os.path.abspath(arguments.binaries_directory))
    for name in ("againrom", "starter"):
        shutil.copy2(binaries / name, inputs / name)
    for name in ("starter.ini", "options.txt", "famehall.dat", "credential.txt", "saves/private.sav", "mods/private/mod.toml", "main.res"):
        sentinel = inputs / name
        sentinel.parent.mkdir(parents=True, exist_ok=True)
        sentinel.write_text("private sentinel", encoding="utf-8")
    environment = os.environ.copy()
    environment["HOME"] = str(evidence / "home")
    environment["XDG_CONFIG_HOME"] = str(evidence / "home/config")
    Path(environment["HOME"]).mkdir()
    run(["go", "telemetry", "off"], repository, environment)
    input_before = inventory(inputs)
    home_before = inventory(Path(environment["HOME"]))
    revision = run(["git", "rev-parse", "HEAD"], repository, environment)
    version = (repository / "cmd/againrom/VERSION").read_text(encoding="utf-8").strip()
    starter_version = (repository / "cmd/starter/VERSION").read_text(encoding="utf-8").strip()
    packager = repository / "scripts/package-macos.py"

    def invoke(output, tag=f"v{version}", source_revision=revision, arch=arguments.arch, refused=False):
        command = [sys.executable, str(packager), "--binaries-directory", str(inputs),
                   "--output-directory", str(output), "--revision", source_revision,
                   "--tag", tag, "--arch", arch]
        result = subprocess.run(command, cwd=repository, env=environment, text=True, capture_output=True, check=False)
        if refused:
            assert_true(result.returncode != 0, f"Invalid package input was accepted: {output}")
            return None
        assert_true(result.returncode == 0, f"Package failed: {result.stderr.strip()}")
        return json.loads(result.stdout)

    package = invoke(evidence / "package")
    assert_true(inventory(inputs) == input_before, "Package version inspection wrote into the input directory.")
    assert_true(inventory(Path(environment["HOME"])) == home_before, "Package version inspection wrote into the user profile.")
    expected = sorted((
        "againrom/againrom", "againrom/starter", "againrom/BUILD-INFO.json",
        "againrom/README.md", "againrom/LICENSE", "againrom/THIRD_PARTY_NOTICES.md",
        "againrom/LICENSES/Apache-2.0.txt", "againrom/LICENSES/BSD-3-Clause.txt",
        "againrom/LICENSES/LGPL-2.1.txt", "againrom/LICENSES/lz4-BSD-3-Clause.txt",
        "againrom/LICENSES/starlark-BSD-3-Clause.txt",
    ))
    with zipfile.ZipFile(package["zip"]) as archive:
        assert_true(archive.namelist() == expected, "ZIP contents differ from the exact public allowlist.")
        for entry in archive.infolist():
            mode = entry.external_attr >> 16
            executable = entry.filename in ("againrom/againrom", "againrom/starter")
            assert_true(entry.create_system == 3 and stat.S_ISREG(mode), "ZIP entry is not a regular Unix file.")
            assert_true(stat.S_IMODE(mode) == (0o755 if executable else 0o644), "ZIP entry permissions differ.")
            assert_true(entry.date_time == (1980, 1, 1, 0, 0, 0), "ZIP timestamps are not fixed.")
        manifest = json.loads(archive.read("againrom/BUILD-INFO.json"))
    assert_true(manifest["revision"] == revision and manifest["tag"] == f"v{version}" and
                manifest["target"] == f"darwin/{arguments.arch}" and manifest["version"] == version and
                manifest["starter_version"] == starter_version, "Manifest identity differs.")
    extracted = evidence / "extracted"
    run(["/usr/bin/ditto", "-x", "-k", package["zip"], str(extracted)], repository, environment)
    runtime = extracted / "againrom"
    runtime_before = inventory(runtime)
    for name, program_version in (("againrom", version), ("starter", starter_version)):
        executable = runtime / name
        assert_true(stat.S_IMODE(executable.stat().st_mode) == 0o755, "Extraction lost executable permissions.")
        actual = run([str(executable), "-version"], runtime, environment)
        assert_true(actual == f"{name} {program_version} {revision[:12]}", "Extracted executable version or source stamp differs.")
        metadata = run(["go", "version", "-m", str(executable)], runtime, environment)
        assert_true(re.search(r": " + re.escape(manifest["go"]) + r"$", metadata.splitlines()[0]) is not None,
                    "Extracted executable toolchain differs from manifest.")
        for setting in ("GOOS=darwin", f"GOARCH={arguments.arch}", "CGO_ENABLED=1", "-trimpath=true"):
            assert_true(any(line.strip() == f"build\t{setting}" for line in metadata.splitlines()),
                        f"Extracted executable lacks {setting}.")
    assert_true(inventory(runtime) == runtime_before, "Extracted version inspection wrote a runtime profile.")
    assert_true(inventory(Path(environment["HOME"])) == home_before, "Extracted version inspection wrote a user profile.")
    archive = Path(package["zip"])
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    assert_true(Path(package["checksum"]).read_text(encoding="utf-8") == f"{digest}  {archive.name}\n",
                "SHA256 sibling differs from archive bytes.")
    assert_true(package["sha256"] == digest, "Returned SHA256 differs from archive bytes.")
    repeat = invoke(evidence / "repeat")
    assert_true(Path(repeat["zip"]).read_bytes() == archive.read_bytes(), "Identical inputs produce different ZIP bytes.")
    invoke(evidence / "package", refused=True)
    assert_true(hashlib.sha256(archive.read_bytes()).hexdigest() == digest, "Existing output was modified.")
    for index, tag in enumerate(("main", "v00.90.0", f"v{version}-rc.1", "v999.999.999")):
        output = evidence / f"invalid-tag-{index}"
        invoke(output, tag=tag, refused=True)
        assert_true(not os.path.lexists(output), "Invalid tag wrote output.")
    for index, bad_revision in enumerate(("0" * 40, "not-a-revision")):
        output = evidence / f"wrong-revision-{index}"
        invoke(output, source_revision=bad_revision, refused=True)
        assert_true(not os.path.lexists(output), "Invalid revision wrote output.")
    missing_parent = evidence / "missing-parent/package"
    invoke(missing_parent, refused=True)
    assert_true(not missing_parent.parent.exists(), "Missing output parent was created.")
    install = evidence / "install-fixture"
    install.mkdir()
    for name in ("MAIN.RES", "GRAPHICS.RES", "SCENARIO.RES", "WORLD.RES", "MOVIES.RES"):
        (install / name).write_bytes(b"")
    (install / "child").mkdir()
    for output in (install / "package", install / "child/package"):
        invoke(output, refused=True)
        assert_true(not os.path.lexists(output), "Install refusal wrote output.")
    target = evidence / "symlink-target"
    target.mkdir()
    link = evidence / "symlink-parent"
    link.symlink_to(target, target_is_directory=True)
    invoke(link / "package", refused=True)
    assert_true(list(target.iterdir()) == [], "Symbolic-link refusal wrote output.")
    (target / "child").mkdir()
    invoke(link / "child/package", refused=True)
    assert_true(list((target / "child").iterdir()) == [], "Symbolic-link ancestor refusal wrote output.")
    dangling = evidence / "dangling-output"
    dangling.symlink_to(evidence / "missing-target", target_is_directory=True)
    invoke(dangling, refused=True)
    assert_true(not (evidence / "missing-target").exists(), "Dangling output symbolic link was followed.")
    print(f"macos-package {arguments.arch}: 11 entries; private sentinels excluded; native versions, stamps, manifest, "
          f"Unix extraction permissions, no profile writes, checksum and repeated ZIP bytes verified; "
          f"existing output, four invalid tags, two invalid revisions, missing parent, two install-contained outputs "
          f"and three symbolic-link outputs refused. SHA256={digest}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binaries-directory", required=True)
    parser.add_argument("--evidence-directory", required=True)
    parser.add_argument("--arch", required=True, choices=("amd64", "arm64"))
    try:
        verify(parser.parse_args())
    except (OSError, ValueError, subprocess.SubprocessError, zipfile.BadZipFile) as error:
        print(f"macos-package witness: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
