# Windows PowerShell 5.1 or later. Keep this file beside Play-Againrom.cmd and againrom.exe.
[CmdletBinding()]
param(
    [Parameter(Position = 0)][string]$Assets,
    [string]$LogDirectory,
    [switch]$NoMovies,
    [switch]$NoPause
)

$ErrorActionPreference = 'Stop'
$requiredArchives = @('main.res', 'graphics.res', 'scenario.res', 'world.res', 'movies.res')
$packageDirectory = [IO.Path]::GetFullPath($PSScriptRoot)
$utf8 = New-Object Text.UTF8Encoding($false)
[Console]::OutputEncoding = $utf8
[Console]::InputEncoding = $utf8
$runLog = $null
$childExit = $null
$exitCode = 1

if ($env:AGAINROM_LAUNCH_WRAPPER -eq '1') {
    $Assets = $env:AGAINROM_LAUNCH_WRAPPER_ASSETS
    $LogDirectory = $env:AGAINROM_LAUNCH_WRAPPER_LOGS
    $NoMovies = [switch]($env:AGAINROM_LAUNCH_WRAPPER_NO_MOVIES -eq '1')
}

function Get-FolderPath([string]$Value) {
    $Value = $Value.Trim()
    if ($Value.Length -ge 2 -and $Value.StartsWith('"') -and $Value.EndsWith('"')) {
        $Value = $Value.Substring(1, $Value.Length - 2)
    }
    if ([string]::IsNullOrWhiteSpace($Value)) { return '' }
    return [IO.Path]::GetFullPath($Value)
}

function Test-Within([string]$Path, [string]$Root) {
    if ([string]::IsNullOrEmpty($Root)) { return $false }
    $Root = $Root.TrimEnd('\', '/')
    return $Path.Equals($Root, [StringComparison]::OrdinalIgnoreCase) -or
        $Path.StartsWith($Root + '\', [StringComparison]::OrdinalIgnoreCase)
}

function Assert-LocalOutputDirectory([string]$Directory, [string]$AssetRoot) {
    # Refuse output through junctions/symlinks, under the selected install, or
    # under another recognizable install. No probe file is written by this check.
    if ($Directory -notmatch '^[A-Za-z]:\\') { throw 'Output needs a local drive folder.' }
    $drive = New-Object IO.DriveInfo([IO.Path]::GetPathRoot($Directory))
    if ($drive.DriveType -eq [IO.DriveType]::Network) { throw 'Output needs a local drive folder.' }
    if (Test-Within $Directory $AssetRoot) { throw 'Output cannot be inside the selected game installation.' }
    $probe = $Directory
    while ($probe) {
        $item = $null
        try { $item = Get-Item -LiteralPath $probe -Force } catch {
            if ($_.CategoryInfo.Category -ne 'ObjectNotFound') { throw }
        }
        if ($null -ne $item) {
            if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
                throw "Output cannot use a junction or symbolic link: $probe"
            }
            if (-not $item.PSIsContainer) { throw "Output folder is occupied by a file: $probe" }
            $archiveCount = 0
            foreach ($archive in $requiredArchives) {
                if ([IO.File]::Exists([IO.Path]::Combine($probe, $archive))) { $archiveCount++ }
            }
            if ($archiveCount -eq $requiredArchives.Count) {
                throw "Output cannot be inside a game installation: $probe"
            }
        }
        $parent = [IO.Directory]::GetParent($probe)
        if ($null -eq $parent) { break }
        $probe = $parent.FullName
    }
}

function New-RunLog([string]$Directory, [string]$AssetRoot) {
    Assert-LocalOutputDirectory $Directory $AssetRoot
    [IO.Directory]::CreateDirectory($Directory) | Out-Null
    $name = 'againrom-{0}-{1}.log' -f [DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss-fff'), [Guid]::NewGuid().ToString('N')
    $path = [IO.Path]::Combine($Directory, $name)
    $file = [IO.File]::Open($path, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::Read)
    $file.Dispose()
    return $path
}

function Write-LaunchLine([string]$Line) {
    if ($null -ne $runLog) { [IO.File]::AppendAllText($runLog, "[launcher] $Line`r`n", $utf8) }
}

function Get-ExecutableHash([string]$Path) {
    $stream = [IO.File]::OpenRead($Path)
    $sha = [Security.Cryptography.SHA256]::Create()
    try { return [BitConverter]::ToString($sha.ComputeHash($stream)).Replace('-', '').ToLowerInvariant() }
    finally { $sha.Dispose(); $stream.Dispose() }
}

function ConvertTo-NativePathArgument([string]$Path) {
    # A native Windows parser consumes backslashes before a closing quote in
    # pairs. Preserve terminal separators, including the one in a drive root.
    if ($Path.Contains('"')) { throw 'A filesystem path cannot contain a quote.' }
    return '"' + [regex]::Replace($Path, '(\\+)$', '$1$1') + '"'
}

function New-GameStartInfo([string]$Executable, [string]$AssetRoot, [string]$SaveDirectory,
                          [string]$LogPath, [string]$WorkingDirectory, [bool]$MoviesDisabled) {
    # cmd captures the native streams together without PowerShell error-record
    # formatting, event callbacks or full-run buffering. Path arguments are
    # quoted for the native child; the log redirection uses cmd's own quoting.
    # Delayed expansion stays off so literal %, ! and & survive.
    $start = New-Object Diagnostics.ProcessStartInfo
    $start.FileName = [IO.Path]::Combine([Environment]::GetFolderPath('System'), 'cmd.exe')
    $movieArgument = if ($MoviesDisabled) { ' -movies=false' } else { '' }
    $start.Arguments = '/d /v:off /s /c ""%AGAINROM_LAUNCH_GAME%" -assets %AGAINROM_LAUNCH_ASSETS_ARG% -saves %AGAINROM_LAUNCH_SAVES_ARG%' + $movieArgument + ' >> "%AGAINROM_LAUNCH_LOG%" 2>&1"'
    $start.UseShellExecute = $false
    $start.CreateNoWindow = $true
    $start.WorkingDirectory = $WorkingDirectory
    $start.EnvironmentVariables['AGAINROM_LAUNCH_GAME'] = $Executable
    $start.EnvironmentVariables['AGAINROM_LAUNCH_ASSETS_ARG'] = ConvertTo-NativePathArgument $AssetRoot
    $start.EnvironmentVariables['AGAINROM_LAUNCH_SAVES_ARG'] = ConvertTo-NativePathArgument $SaveDirectory
    $start.EnvironmentVariables['AGAINROM_LAUNCH_LOG'] = $LogPath
    return $start
}

try {
    Write-Host 'Againrom'
    if ([string]::IsNullOrWhiteSpace($Assets)) {
        $Assets = Read-Host 'Lawful Rage of Mages folder (blank cancels)'
    }
    $assetRoot = ''
    $assetError = $null
    try { $assetRoot = Get-FolderPath $Assets } catch { $assetError = $_.Exception.Message }

    $logError = $null
    try {
        $preferredLogDirectory = if ($LogDirectory) { Get-FolderPath $LogDirectory } else { [IO.Path]::Combine($packageDirectory, 'logs') }
        $runLog = New-RunLog $preferredLogDirectory $assetRoot
    } catch {
        $logError = $_.Exception.Message
        $fallbackDirectory = [IO.Path]::Combine([IO.Path]::GetTempPath(), 'Againrom', 'logs')
        $runLog = New-RunLog $fallbackDirectory $assetRoot
        Write-Host 'Package log unavailable; using a temporary log.'
    }
    Write-Host "Log: $runLog"
    Write-LaunchLine ('Started UTC: ' + [DateTime]::UtcNow.ToString('o'))
    Write-LaunchLine "Package: $packageDirectory"
    if ($logError) { Write-LaunchLine "Preferred log unavailable: $logError" }
    if ($assetError) { throw "Invalid asset folder: $assetError" }
    if (-not $assetRoot) {
        $exitCode = 2
        throw 'No asset folder selected. Launch cancelled.'
    }
    Write-LaunchLine "Assets: $assetRoot"
    if (-not [IO.Directory]::Exists($assetRoot)) { throw "Asset folder not found: $assetRoot" }
    foreach ($archive in $requiredArchives) {
        if (-not [IO.File]::Exists([IO.Path]::Combine($assetRoot, $archive))) {
            throw "Asset folder is missing $archive. Choose the installed game folder."
        }
    }

    $saveDirectory = [IO.Path]::Combine($packageDirectory, 'saves')
    Assert-LocalOutputDirectory $saveDirectory $assetRoot
    $executable = [IO.Path]::Combine($packageDirectory, 'againrom.exe')
    if (-not [IO.File]::Exists($executable)) { throw "againrom.exe is missing beside the launcher: $packageDirectory" }
    Write-LaunchLine "Executable: $executable"
    Write-LaunchLine ('SHA256: ' + (Get-ExecutableHash $executable))
    Write-LaunchLine "Saves: $saveDirectory"
    Write-LaunchLine "Movies disabled: $($NoMovies.IsPresent)"

    $child = New-Object Diagnostics.Process
    $child.StartInfo = New-GameStartInfo $executable $assetRoot $saveDirectory $runLog $packageDirectory $NoMovies.IsPresent
    try {
        Write-Host 'Starting game...'
        if (-not $child.Start()) { throw 'Could not start the game process.' }
        $child.WaitForExit()
        $childExit = $child.ExitCode
        $exitCode = $childExit
    } finally {
        $child.Dispose()
    }
} catch {
    $message = $_.Exception.Message
    Write-Host "Launch error: $message"
    try { Write-LaunchLine "Error: $message" } catch { Write-Host 'Could not append the launch error to the log.' }
} finally {
    $status = if ($null -ne $childExit) { "Game exit code: $childExit" } else { "Game not started; launcher exit code: $exitCode" }
    Write-Host $status
    try {
        Write-LaunchLine $status
        Write-LaunchLine ('Finished UTC: ' + [DateTime]::UtcNow.ToString('o'))
    } catch { Write-Host 'Could not append the exit status to the log.' }
    if ($null -ne $runLog) { Write-Host "Log: $runLog" }
}
exit $exitCode
