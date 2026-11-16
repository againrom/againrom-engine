param(
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$Link,
    [string]$Changelog = 'CHANGELOG.md',
    [switch]$DryRun
)
# Announce one published release in Discord: one embed with the version as the
# title, the release page link, and the player and map-maker/modder parts of
# the CHANGELOG section. The text comes from scripts/release-notes.ps1. The
# webhook address is read from DISCORD_RELEASE_WEBHOOK, never printed, and an
# empty or absent one makes a real run do nothing. -DryRun prints the JSON
# instead of sending it.
$ErrorActionPreference = 'Stop'
$webhook = $env:DISCORD_RELEASE_WEBHOOK
if (-not $DryRun -and [string]::IsNullOrWhiteSpace($webhook)) {
    Write-Output 'DISCORD_RELEASE_WEBHOOK is not set; nothing to announce.'
    return
}
$notes = (& (Join-Path $PSScriptRoot 'release-notes.ps1') -Version $Version -Changelog $Changelog -Section @('For players', 'For map makers and modders')) -join "`n"

# Discord allows 4096 characters in an embed description and 6000 in the whole
# embed. The description is cut at an entry boundary: a heading, or a top-level
# bullet with its nested lines. A cut description ends with a line that links
# the full notes.
$limit = 4000
$closing = "`n`n[Full notes]($Link)"
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
$description = ''
$cut = $false
for ($i = 0; $i -lt $units.Count; $i++) {
    $next = if ($description -eq '') { $units[$i] } elseif ($units[$i] -match '^\*\*') { "$description`n`n$($units[$i])" } else { "$description`n$($units[$i])" }
    if ($next.Length + $closing.Length -gt $limit) { $cut = $true; break }
    $description = $next
}
if ($cut) {
    # Never end on a bare heading.
    $trimmed = $description -replace '(\n\n)?\*\*[^\n]*\*\*$', ''
    if ($trimmed -ne '') { $description = $trimmed }
    if ($description -eq '') { $description = $units[0].Substring(0, [Math]::Min($units[0].Length, $limit - $closing.Length - 1)) + [char]0x2026 }
    $description += $closing
}
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
