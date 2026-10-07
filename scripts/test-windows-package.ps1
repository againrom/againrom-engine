[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$BinariesDirectory,
    [Parameter(Mandatory)][string]$EvidenceDirectory
)
$ErrorActionPreference = 'Stop'
function Assert-True([bool]$Condition, [string]$Message) { if (-not $Condition) { throw $Message } }
function Assert-Refused([scriptblock]$Action, [string]$Message) {
    $refused = $false
    try { & $Action | Out-Null } catch { $refused = $true }
    Assert-True $refused $Message
}
$repository = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$evidence = [IO.Path]::GetFullPath($EvidenceDirectory)
if (Test-Path -LiteralPath $evidence) { throw 'Evidence directory already exists; choose a fresh path.' }
New-Item -ItemType Directory -Path $evidence | Out-Null
$inputs = Join-Path $evidence 'inputs'
New-Item -ItemType Directory -Path $inputs | Out-Null
foreach ($name in @('againrom.exe', 'starter.exe')) { Copy-Item -LiteralPath (Join-Path $BinariesDirectory $name) -Destination $inputs }
foreach ($file in @('starter.ini', 'options.txt', 'famehall.dat', 'credential.txt', 'saves/private.sav', 'mods/private/mod.toml', 'main.res')) {
    $path = Join-Path $inputs $file
    New-Item -ItemType Directory -Path ([IO.Path]::GetDirectoryName($path)) -Force | Out-Null
    [IO.File]::WriteAllText($path, 'private sentinel')
}
$revision = (& git -C $repository rev-parse HEAD).Trim()
$version = (Get-Content -LiteralPath (Join-Path $repository 'cmd/againrom/VERSION') -Raw).Trim()
$packager = Join-Path $PSScriptRoot 'package-windows.ps1'
$arguments = @{ BinariesDirectory = $inputs; Revision = $revision; Tag = "v$version" }
$package = & $packager @arguments -OutputDirectory (Join-Path $evidence 'package')
Add-Type -AssemblyName System.IO.Compression.FileSystem
$archive = [IO.Compression.ZipFile]::OpenRead($package.Zip)
try {
    $actual = @($archive.Entries.FullName | Sort-Object)
    $expected = @('againrom/againrom.exe', 'againrom/starter.exe', 'againrom/BUILD-INFO.json',
        'againrom/README.md', 'againrom/LICENSE', 'againrom/THIRD_PARTY_NOTICES.md',
        'againrom/LICENSES/Apache-2.0.txt', 'againrom/LICENSES/BSD-3-Clause.txt',
        'againrom/LICENSES/LGPL-2.1.txt', 'againrom/LICENSES/lz4-BSD-3-Clause.txt',
        'againrom/LICENSES/starlark-BSD-3-Clause.txt' | Sort-Object)
    Assert-True (($actual -join "`n") -ceq ($expected -join "`n")) 'ZIP contents differ from the public allowlist.'
} finally { $archive.Dispose() }
$extracted = Join-Path $evidence 'extracted'
[IO.Compression.ZipFile]::ExtractToDirectory($package.Zip, $extracted)
$runtime = Join-Path $extracted 'againrom'
$manifest = Get-Content -LiteralPath (Join-Path $runtime 'BUILD-INFO.json') -Raw | ConvertFrom-Json
Assert-True ($manifest.revision -ceq $revision -and $manifest.tag -ceq "v$version" -and $manifest.target -ceq 'windows/amd64') 'Manifest identity differs.'
foreach ($name in @('againrom', 'starter')) {
    $line = (& (Join-Path $runtime ($name + '.exe')) -version | Out-String).Trim()
    Assert-True ($LASTEXITCODE -eq 0 -and $line.EndsWith(' ' + $revision.Substring(0, 12))) 'Extracted executable lost its stamp.'
}
Assert-True (-not (Test-Path -LiteralPath (Join-Path $runtime 'starter.ini'))) 'Version inspection created starter.ini.'
Assert-True (-not (Test-Path -LiteralPath (Join-Path $runtime 'options.txt'))) 'Version inspection created options.txt.'
Assert-True (-not (Test-Path -LiteralPath (Join-Path $runtime 'saves'))) 'Version inspection created saves.'
$hash = (Get-FileHash -LiteralPath $package.Zip -Algorithm SHA256).Hash.ToLowerInvariant()
Assert-True ((Get-Content -LiteralPath $package.Checksum -Raw) -ceq "$hash  $([IO.Path]::GetFileName($package.Zip))`n") 'Checksum differs.'
$repeat = & $packager @arguments -OutputDirectory (Join-Path $evidence 'repeat')
Assert-True ($repeat.SHA256 -ceq $package.SHA256) 'Identical inputs produce different ZIP bytes.'
$before = (Get-FileHash -LiteralPath $package.Zip -Algorithm SHA256).Hash
Assert-Refused { & $packager @arguments -OutputDirectory (Join-Path $evidence 'package') } 'An existing output was accepted.'
Assert-True ((Get-FileHash -LiteralPath $package.Zip -Algorithm SHA256).Hash -ceq $before) 'An existing archive changed.'
foreach ($badTag in @('main', 'v00.88.3', "v$version-rc.1", 'v999.999.999')) {
    $bad = $arguments.Clone(); $bad.Tag = $badTag
    $output = Join-Path $evidence ('invalid-' + $badTag)
    Assert-Refused { & $packager @bad -OutputDirectory $output } "Invalid tag was accepted: $badTag"
    Assert-True (-not (Test-Path -LiteralPath $output)) 'Invalid tag wrote output.'
}
$wrong = $arguments.Clone(); $wrong.Revision = '0' * 40
Assert-Refused { & $packager @wrong -OutputDirectory (Join-Path $evidence 'wrong-revision') } 'Wrong revision was accepted.'
$install = Join-Path $evidence 'install-fixture'
New-Item -ItemType Directory -Path $install | Out-Null
foreach ($name in @('MAIN.RES', 'GRAPHICS.RES', 'SCENARIO.RES', 'WORLD.RES', 'MOVIES.RES')) {
    [IO.File]::WriteAllBytes((Join-Path $install $name), [byte[]]@())
}
Assert-Refused { & $packager @arguments -OutputDirectory (Join-Path $install 'package') } 'Install-contained output was accepted.'
Assert-True (-not (Test-Path -LiteralPath (Join-Path $install 'package'))) 'Install refusal wrote output.'
Write-Output "windows-package: 11 entries; private sentinels excluded; extracted stamps, versions, manifest, checksum and repeat bytes verified; existing output, four invalid tags, wrong revision and install-contained output refused. SHA256=$hash"
