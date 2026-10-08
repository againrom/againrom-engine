# macOS release builds

## Intent and authority

Owner direction: build the game automatically for macOS Intel and Apple Silicon
beside Windows. Ship no installed game data. Signing and notarization are outside
this scope. No ROM1 behaviour claim or divergence is introduced.

Base: `5f00a110f6ecb9366d8cce84cef5a910c58a20e9`.

## As built

The reusable macOS workflow builds native `darwin/amd64` on `macos-15-intel` and
`darwin/arm64` on `macos-15`, with CGO enabled. These pinned labels select the
required architectures without a moving `macos-latest` label. GitHub lists both
as standard hosted runners in its [runner reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).

Both ZIPs include `againrom` and `starter`, the public notices and a build
manifest. Starter already selects the unsuffixed game executable outside Windows.
Program versions are `againrom 0.90.0` and `starter 0.5.0`; Windows resources are
regenerated. Existing platform files isolate Windows APIs; no gameplay, screen,
installed-data or save code changes are required.

Starter tests use native path separators and construct a settings-write failure
after loading a valid INI. Profile tests compare install-local paths with their
physical paths. Fallback, explicit paths and original-install expectations retain
their existing meaning. These are test changes; production profile code is unchanged.

The existing Windows workflow calls the macOS workflow for the same immutable
version tag. A single publisher verifies all three ZIPs and checksums before
uploading six assets through one draft. Its existing canonical-repository guard,
conflict refusal and no-overwrite policy remain. Branch verification uploads
Actions artifacts only and never publishes a release. Standalone manual macOS
verification accepts an empty tag; a tagged dispatch must come from `main`.

Packaging checks tracked cleanliness, exact HEAD and VERSION, native executable
stamps, architecture, CGO, public-file allowlist, executable permissions and stable
archive bytes. Output paths are explicit, new and outside installs and symlinks.
Native `-version` checks the program's 12-character revision stamp; the manifest
records the full SHA. Go metadata supplies the target, CGO and toolchain identity.
The witness disables Go telemetry only in its isolated HOME before observing
whether executable version inspection writes player files.

The game obtains its fallback profile root through `os.UserConfigDir()`. On
macOS this is `~/Library/Application Support/Againrom`. Inside an install it
first selects `Againrom/` beside the executable, with the existing fallback
when that directory is unwritable. The implementation is unchanged.

README documents terminal launch and `xattr -d com.apple.quarantine againrom
starter` for a trusted downloaded package. No `.app` bundle is produced.

## Proof

Private development Actions run 37778550492
passed both architectures at `b9c7777bf6957f371b788366358e7e380325b311` with Go
1.26.1. Both jobs built and executed the native programs, passed the launch/profile
and version tests, and uploaded ZIP/checksum artifacts.

| Target | Runner | Successful job | Public ZIP SHA-256 |
|---|---|---|---|
| darwin/amd64 | macos-15-intel | 113315521769 | `0da990aee1363d9c8310fb826e238b7addcc69f27b2eb4f9800705865b84b966` |
| darwin/arm64 | macos-15 | 113315522115 | `24267c38abd4f4b17a3f75317b97f0bf3150d52179545a2156915ab373ed8bce` |

Each native witness verified exactly 11 public entries, versions and stamps,
manifest identity, executable permissions after `ditto`, no profile writes,
checksums and repeated ZIP bytes. Existing output, four invalid tags, two invalid
revisions, a missing parent, two install-contained outputs and three symbolic-link
outputs were refused. The Windows package witness passed in 6.69 seconds at
`c57ffa5c7efbe3b14dde6658aca0eef4173af90c`; later production Windows source is
unchanged. The full ordinary tests of both packages containing profile-test
changes passed in 143.71 seconds before their commit.

## Open debt

Mac starter clipboard paste and Command+V are unsupported. Paths can be typed.
Native build and package evidence do not establish installed-game rendering,
audio, movies or campaign play on a Mac. No Mac gameplay witness, Developer ID
certificate, notarization or `.app` bundle is supplied.
