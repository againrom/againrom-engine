# Package only the public Windows runtime and notices into a new output directory.
[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$BinariesDirectory,
    [Parameter(Mandatory)][string]$OutputDirectory,
    [Parameter(Mandatory)][string]$Revision,
    [Parameter(Mandatory)][string]$Tag
)
$ErrorActionPreference = 'Stop'
$repository = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$version = (Get-Content -LiteralPath (Join-Path $repository 'cmd/againrom/VERSION') -Raw).Trim()
$starterVersion = (Get-Content -LiteralPath (Join-Path $repository 'cmd/starter/VERSION') -Raw).Trim()
if ($Tag -cnotmatch '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$' -or $Tag -cne "v$version") {
    throw 'Tag must be vMAJOR.MINOR.PATCH and match cmd/againrom/VERSION.'
}
$head = (& git -C $repository rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $Revision -cnotmatch '^[0-9a-f]{40}$' -or $Revision -cne $head) {
    throw 'Revision must be the exact checkout HEAD.'
}
$dirty = & git -C $repository status --porcelain --untracked-files=no
if ($LASTEXITCODE -ne 0 -or $dirty) { throw 'Package source must have no tracked changes.' }
$output = [IO.Path]::GetFullPath($OutputDirectory)
if (Test-Path -LiteralPath $output) { throw 'Output directory already exists; preserve it and choose a fresh path.' }
$parent = Get-Item -LiteralPath ([IO.Path]::GetDirectoryName($output))
if (-not $parent.PSIsContainer) { throw 'Output parent must be an existing directory.' }
$requiredArchives = @('main.res', 'graphics.res', 'scenario.res', 'world.res', 'movies.res')
for ($directory = $parent; $null -ne $directory; $directory = $directory.Parent) {
    if ($directory.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Output cannot traverse a reparse point.' }
    $names = @(Get-ChildItem -LiteralPath $directory.FullName -File | ForEach-Object { $_.Name.ToLowerInvariant() })
    if (@($requiredArchives | Where-Object { $_ -notin $names }).Count -eq 0) {
        throw 'Output cannot be inside a game install.'
    }
}
$binaries = [IO.Path]::GetFullPath($BinariesDirectory)
$goVersion = ''
foreach ($program in @(@{ Name = 'againrom'; Version = $version }, @{ Name = 'starter'; Version = $starterVersion })) {
    $exe = Join-Path $binaries ($program.Name + '.exe')
    $line = (& $exe -version | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $line -cne "$($program.Name) $($program.Version) $($Revision.Substring(0, 12))") {
        throw "Unexpected source stamp: $exe"
    }
    if ([Diagnostics.FileVersionInfo]::GetVersionInfo($exe).FileVersion -cne $program.Version) {
        throw "Windows version resource does not match VERSION: $exe"
    }
    $metadata = @(& go version -m $exe)
    if ($LASTEXITCODE -ne 0 -or $metadata[0] -notmatch ': (go\S+)$') { throw "Cannot read Go build identity: $exe" }
    $compiledGo = $Matches[1]
    if ($goVersion -ne '' -and $compiledGo -cne $goVersion) { throw 'The executables use different Go toolchains.' }
    $goVersion = $compiledGo
    foreach ($setting in @('GOOS=windows', 'GOARCH=amd64', 'CGO_ENABLED=0', '-trimpath=true')) {
        if (-not ($metadata | Where-Object { $_.Trim() -match ('\s' + [regex]::Escape($setting) + '$') })) {
            throw "Missing build setting $setting in $exe"
        }
    }
}
$files = @('README.md', 'LICENSE', 'THIRD_PARTY_NOTICES.md',
    'LICENSES/Apache-2.0.txt', 'LICENSES/BSD-3-Clause.txt', 'LICENSES/LGPL-2.1.txt',
    'LICENSES/lz4-BSD-3-Clause.txt', 'LICENSES/starlark-BSD-3-Clause.txt')
foreach ($file in $files) {
    if (-not (Test-Path -LiteralPath (Join-Path $repository $file) -PathType Leaf)) { throw "Missing public file: $file" }
}
$stage = Join-Path $output 'againrom'
New-Item -ItemType Directory -Path $output | Out-Null
New-Item -ItemType Directory -Path (Join-Path $stage 'LICENSES') | Out-Null
foreach ($file in $files) { Copy-Item -LiteralPath (Join-Path $repository $file) -Destination (Join-Path $stage $file) }
foreach ($name in @('againrom.exe', 'starter.exe')) { Copy-Item -LiteralPath (Join-Path $binaries $name) -Destination $stage }
$manifest = [ordered]@{
    version = $version; starter_version = $starterVersion; revision = $Revision; tag = $Tag
    target = 'windows/amd64'; go = $goVersion
    repository = 'https://github.com/againrom/againrom-engine'
}
[IO.File]::WriteAllText((Join-Path $stage 'BUILD-INFO.json'), ($manifest | ConvertTo-Json) + "`n", [Text.UTF8Encoding]::new($false))
$name = "againrom-$version-windows-amd64.zip"
$zip = Join-Path $output $name
Add-Type -AssemblyName System.IO.Compression.FileSystem
$archive = [IO.Compression.ZipFile]::Open($zip, [IO.Compression.ZipArchiveMode]::Create)
try {
    foreach ($file in @($files + @('againrom.exe', 'starter.exe', 'BUILD-INFO.json') | Sort-Object)) {
        $entry = $archive.CreateEntry(('againrom/' + $file), [IO.Compression.CompressionLevel]::Optimal)
        $entry.LastWriteTime = [DateTimeOffset]::new(1980, 1, 1, 0, 0, 0, [TimeSpan]::Zero)
        $input = [IO.File]::OpenRead((Join-Path $stage $file))
        $destination = $entry.Open()
        try { $input.CopyTo($destination) } finally { $destination.Dispose(); $input.Dispose() }
    }
} finally { $archive.Dispose() }
$checksum = Join-Path $output ($name + '.sha256')
$hash = (Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash.ToLowerInvariant()
[IO.File]::WriteAllText($checksum, "$hash  $name`n", [Text.UTF8Encoding]::new($false))
[pscustomobject]@{ Zip = $zip; Checksum = $checksum; Manifest = (Join-Path $stage 'BUILD-INFO.json'); SHA256 = $hash }
