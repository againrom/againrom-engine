param(
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$Link,
    [string]$Changelog = 'CHANGELOG.md',
    [switch]$DryRun
)
# Announce one published release in Discord: one short embed with the version
# as the title, the release page link, and the first entries of the player and
# map-maker/modder parts of the CHANGELOG section, then a link to the full
# notes. The text comes from scripts/release-notes.ps1. The webhook address is
# read from DISCORD_RELEASE_WEBHOOK, never printed, and an empty or absent one
# makes a real run do nothing. -DryRun prints the JSON instead of sending it.
$ErrorActionPreference = 'Stop'
$webhook = $env:DISCORD_RELEASE_WEBHOOK
if (-not $DryRun -and [string]::IsNullOrWhiteSpace($webhook)) {
    Write-Output 'DISCORD_RELEASE_WEBHOOK is not set; nothing to announce.'
    return
}
$notes = (& (Join-Path $PSScriptRoot 'release-notes.ps1') -Version $Version -Changelog $Changelog -Section @('For players', 'For map makers and modders')) -join "`n"

# The description holds at most 6 entries and 2000 characters, players' entries
# first, each cut at an entry boundary (a top-level bullet with its nested
# lines); a heading shows only above an entry under it. Discord's own limits
# (4096 in the description, 6000 in the embed) stay as guards. The description
# ends with a line that links the full notes.
$maxEntries = 6
$limit = 2000
$units = New-Object System.Collections.Generic.List[string]
foreach ($line in $notes -split "`n") {
    if ($line -match '^\s*$') { continue }
    if ($line -match '^###\s+(.*?)\s*$') {
        $units.Add('**' + $Matches[1] + '**')
    } elseif ($line -match '^[-*+]\s' -or $units.Count -eq 0 -or $units[$units.Count - 1] -match '^\*\*') {
        $units.Add($line)
    } else {
        $units[$units.Count - 1] += "`n" + $line
    }
}
$total = @($units | Where-Object { $_ -notmatch '^\*\*' }).Count
$description = ''
$heading = ''
$taken = 0
$full = $false
foreach ($unit in $units) {
    if ($unit -match '^\*\*') { $heading = $unit; continue }
    if ($taken -ge $maxEntries) { break }
    $next = if ($description -eq '') { $unit } else { "$description`n$unit" }
    if ($heading -ne '') {
        $next = if ($description -eq '') { "$heading`n$unit" } else { "$description`n`n$heading`n$unit" }
    }
    $left = $total - $taken - 1
    $tail = if ($left -gt 0) { "`n`n+$left more changes: [Full notes]($Link)" } else { "`n`n[Full notes]($Link)" }
    if ($next.Length + $tail.Length -gt $limit -and $taken -gt 0) { break }
    $description = $next
    $heading = ''
    $taken++
}
$left = $total - $taken
$description += if ($left -gt 0) { "`n`n+$left more changes: [Full notes]($Link)" } else { "`n`n[Full notes]($Link)" }
if ($description.Length -gt 4096) { throw 'Announcement exceeds the Discord description limit.' }
$title = "Againrom $Version"
if ($title.Length + $description.Length -gt 6000) { throw 'Announcement exceeds the Discord embed limit.' }
$payload = [ordered]@{
    allowed_mentions = [ordered]@{ parse = @() }
    embeds = @([ordered]@{ title = $title; url = $Link; description = $description; color = 5793266 })
}
$json = $payload | ConvertTo-Json -Depth 6 -Compress
if ($DryRun) {
    $json
    return
}
if ($webhook -cnotmatch '^https://(discord|discordapp)\.com/api/webhooks/') { throw 'DISCORD_RELEASE_WEBHOOK is not a Discord webhook address.' }
try {
    Invoke-RestMethod -Method Post -Uri $webhook -ContentType 'application/json; charset=utf-8' -Body ([Text.UTF8Encoding]::new($false).GetBytes($json)) | Out-Null
} catch {
    $status = ''
    if ($_.Exception.Response) { $status = " (HTTP $([int]$_.Exception.Response.StatusCode))" }
    throw "The Discord post failed$status."
}
Write-Output "Announced $Version in Discord."
