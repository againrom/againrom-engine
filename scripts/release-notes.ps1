param(
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$Revision,
    [string]$Changelog = 'CHANGELOG.md'
)
# Print the release notes for one version: its CHANGELOG.md section, then the
# install text. A release without a written section is refused.
$ErrorActionPreference = 'Stop'
$text = [IO.File]::ReadAllText((Resolve-Path -LiteralPath $Changelog)) -replace "`r`n", "`n"
$pattern = '(?ms)^## ' + [regex]::Escape($Version) + '[ \t]*\n(.*?)(?=^## |\z)'
$match = [regex]::Match($text, $pattern)
if (-not $match.Success) { throw "CHANGELOG.md has no section for $Version." }
$changes = $match.Groups[1].Value.Trim()
if ($changes -eq '') { throw "CHANGELOG.md section $Version is empty." }
$install = 'Windows AMD64, macOS Intel and macOS Apple Silicon. Unpack into a separate folder and run starter.exe on Windows or ./starter on macOS. Mac builds are unsigned and not notarized; see README.md for Gatekeeper instructions. A lawfully owned game install is required. No game assets or player profiles are included.'
"$changes`n`n---`n`n$install`n`nSource commit: $Revision"
