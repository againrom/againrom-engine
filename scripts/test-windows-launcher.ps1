# Controlled processes only; no game or original executable is launched.
[CmdletBinding()]
param([Parameter(Mandatory = $true)][string]$EvidenceDirectory)

$ErrorActionPreference = 'Stop'
$repository = Split-Path -Parent $PSScriptRoot
$evidence = [IO.Path]::GetFullPath($EvidenceDirectory)
if (Test-Path -LiteralPath $evidence) { throw 'Use a new evidence directory; previous runs are retained.' }
[IO.Directory]::CreateDirectory($evidence) | Out-Null
$utf8 = New-Object Text.UTF8Encoding($false)

function Assert-True([bool]$Condition, [string]$Message) {
    if (-not $Condition) { throw $Message }
}

function Get-ProofHash([string]$Path) {
    $sha = [Security.Cryptography.SHA256]::Create()
    $stream = [IO.File]::OpenRead($Path)
    try { return [BitConverter]::ToString($sha.ComputeHash($stream)) }
    finally { $sha.Dispose(); $stream.Dispose() }
}

function New-Package([string]$Name, [string]$Binary) {
    $package = [IO.Path]::Combine($evidence, $Name)
    [IO.Directory]::CreateDirectory($package) | Out-Null
    foreach ($file in @('Play-Againrom.cmd', 'launch-againrom.ps1')) {
        Copy-Item -LiteralPath ([IO.Path]::Combine($repository, $file)) -Destination $package
    }
    if ($Binary) { Copy-Item -LiteralPath $Binary -Destination ([IO.Path]::Combine($package, 'againrom.exe')) }
    return $package
}

function Invoke-Launcher([string]$Package, [string]$AssetRoot, [int]$FixtureExit = 0,
                         [string]$LogRoot = '', [switch]$NoMovies, [switch]$Prompt,
                         [switch]$Pause, [switch]$BlockFallback) {
    $start = New-Object Diagnostics.ProcessStartInfo
    $start.FileName = [IO.Path]::Combine([Environment]::GetFolderPath('System'), 'cmd.exe')
    $start.Arguments = '/d /v:off /s /c ""%TEST_LAUNCH_CMD%" -NoPause %TEST_LAUNCH_OPTIONS%"'
    if ($Prompt) { $start.Arguments = '/d /v:off /s /c ""%TEST_LAUNCH_CMD%" -NoPause"' }
    if ($Pause) { $start.Arguments = '/d /v:off /s /c ""%TEST_LAUNCH_CMD%" %TEST_LAUNCH_OPTIONS%"' }
    $start.UseShellExecute = $false
    $start.CreateNoWindow = $true
    $start.RedirectStandardInput = $true
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    $start.StandardOutputEncoding = $utf8
    $start.StandardErrorEncoding = $utf8
    $start.WorkingDirectory = $evidence
    $start.EnvironmentVariables['TEST_LAUNCH_CMD'] = [IO.Path]::Combine($Package, 'Play-Againrom.cmd')
    # Values in TEST_LAUNCH_OPTIONS are fixed switches and quoted filesystem
    # paths created by this test; the metacharacter case also crosses the wrapper.
    $options = if ($Prompt) { '' } else { '-Assets "' + $AssetRoot + '"' }
    if ($LogRoot) { $options += ' -LogDirectory "' + $LogRoot + '"' }
    if ($NoMovies) { $options += ' -NoMovies' }
    $start.EnvironmentVariables['TEST_LAUNCH_OPTIONS'] = $options
    $start.EnvironmentVariables['AGAINROM_LAUNCH_TEST_EXIT'] = [string]$FixtureExit
    $start.EnvironmentVariables['AGAINROM_ASSETS'] = 'C:\this-environment-root-must-not-be-used'
    $tempRoot = [IO.Path]::Combine($evidence, 'local-temp')
    [IO.Directory]::CreateDirectory($tempRoot) | Out-Null
    if ($BlockFallback) {
        $tempRoot = [IO.Path]::Combine($evidence, 'blocked-temp')
        [IO.Directory]::CreateDirectory($tempRoot) | Out-Null
        [IO.File]::WriteAllText([IO.Path]::Combine($tempRoot, 'Againrom'), 'retain this file')
    }
    $start.EnvironmentVariables['TEMP'] = $tempRoot
    $start.EnvironmentVariables['TMP'] = $tempRoot
    $process = New-Object Diagnostics.Process
    $process.StartInfo = $start
    try {
        Assert-True $process.Start() 'Could not start the controlled launcher.'
        $stdout = $process.StandardOutput.ReadToEndAsync()
        $stderr = $process.StandardError.ReadToEndAsync()
        if ($Prompt) {
            # Read-Host receives UTF-8 redirected input once the launcher sets
            # its console encoding. Use the byte stream so this probe preserves
            # Unicode independently of the parent PowerShell host's code page.
            $inputBytes = $utf8.GetBytes($AssetRoot + "`r`n")
            $process.StandardInput.BaseStream.Write($inputBytes, 0, $inputBytes.Length)
            $process.StandardInput.BaseStream.Flush()
        }
        if ($Pause) { $process.StandardInput.WriteLine('x') }
        $process.StandardInput.Close()
        if (-not $process.WaitForExit(30000)) {
            $process.Kill()
            throw 'Controlled launcher did not finish in 30 seconds.'
        }
        $text = $stdout.Result + $stderr.Result
        $consolePath = [IO.Path]::Combine($evidence, ('console-' + [Guid]::NewGuid().ToString('N') + '.txt'))
        [IO.File]::WriteAllText($consolePath, $text, $utf8)
        $logLines = @($text -split '\r?\n' | Where-Object { $_.StartsWith('Log: ') })
        $log = if ($logLines.Count) { $logLines[-1].Substring(5) } else { '' }
        $body = if ($log -and [IO.File]::Exists($log)) { [IO.File]::ReadAllText($log, $utf8) } else { '' }
        return [pscustomobject]@{ Exit = $process.ExitCode; Console = $text; Log = $log; Body = $body }
    } finally { $process.Dispose() }
}

$fixtureSource = @'
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "strconv"
    "strings"
    "time"
)

func main() {
    cwd, _ := os.Getwd()
    state, _ := json.Marshal(struct { Args []string; Directory string }{os.Args[1:], cwd})
    fmt.Printf("fixture=%s\n", state)
    fmt.Fprintln(os.Stdout, "stdout sentinel: \u041f\u0440\u0438\u0432\u0435\u0442")
    fmt.Fprintln(os.Stderr, "stderr sentinel: panic-like diagnostic")
    for i := 0; i < 2048; i++ {
        fmt.Fprintf(os.Stdout, "OUT %04d %s\n", i, strings.Repeat("o", 120))
        fmt.Fprintf(os.Stderr, "ERR %04d %s\n", i, strings.Repeat("e", 120))
    }
    // A GUI-subsystem executable must be waited for too.
    time.Sleep(100 * time.Millisecond)
    fmt.Fprintln(os.Stderr, "fixture complete")
    code, _ := strconv.Atoi(os.Getenv("AGAINROM_LAUNCH_TEST_EXIT"))
    os.Exit(code)
}
'@
$source = [IO.Path]::Combine($evidence, 'fixture.go')
[IO.File]::WriteAllText($source, $fixtureSource, $utf8)
$fixture = [IO.Path]::Combine($evidence, 'fixture.exe')
$guiFixture = [IO.Path]::Combine($evidence, 'fixture-gui.exe')
& go build -trimpath -o $fixture $source
Assert-True ($LASTEXITCODE -eq 0) 'Fixture build failed.'
& go build -trimpath -ldflags '-H windowsgui' -o $guiFixture $source
Assert-True ($LASTEXITCODE -eq 0) 'GUI fixture build failed.'

$assets = [IO.Path]::Combine($evidence, 'lawful-fixture')
[IO.Directory]::CreateDirectory($assets) | Out-Null
foreach ($archive in @('MAIN.RES', 'GRAPHICS.RES', 'SCENARIO.RES', 'WORLD.RES', 'MOVIES.RES')) {
    [IO.File]::WriteAllBytes([IO.Path]::Combine($assets, $archive), [byte[]]@())
}
$before = @(Get-ChildItem -LiteralPath $assets -Force -Recurse | ForEach-Object {
    $_.FullName + ':' + (Get-ProofHash $_.FullName)
}) -join "`n"

$package = New-Package 'normal package' $fixture
$normal = Invoke-Launcher $package $assets
Assert-True ($normal.Exit -eq 0) "Normal exit lost: $($normal.Console)"
Assert-True ($normal.Log.StartsWith([IO.Path]::Combine($package, 'logs'))) 'Default log is not beside the package.'
Assert-True ($normal.Body.Contains('Game exit code: 0')) 'Normal exit is not recorded.'
Assert-True ($normal.Body.Contains('stdout sentinel: ' + [string][char]0x41f + [string][char]0x440)) 'UTF-8 stdout was lost.'
Assert-True ($normal.Body.Contains('stderr sentinel: panic-like diagnostic')) 'Stderr was lost.'
Assert-True (([regex]::Matches($normal.Body, '(?m)^OUT \d{4} ')).Count -eq 2048) 'Large stdout was truncated.'
Assert-True (([regex]::Matches($normal.Body, '(?m)^ERR \d{4} ')).Count -eq 2048) 'Large stderr was truncated.'
Assert-True ($normal.Body.IndexOf('fixture complete') -lt $normal.Body.IndexOf('[launcher] Game exit code:')) 'Status was written before the child finished.'
$normalLine = @($normal.Body -split '\r?\n' | Where-Object { $_.StartsWith('fixture=') })[0]
$normalState = $normalLine.Substring(8) | ConvertFrom-Json
$normalArgs = @('-assets', $assets, '-saves', [IO.Path]::Combine($package, 'saves'))
Assert-True (($normalState.Args | ConvertTo-Json -Compress) -eq ($normalArgs | ConvertTo-Json -Compress)) 'Default launch added or changed arguments.'
$normalHash = Get-ProofHash $normal.Log
$failed = Invoke-Launcher $package $assets 37
Assert-True ($failed.Exit -eq 37 -and $failed.Body.Contains('Game exit code: 37')) 'Nonzero child status was lost.'
Assert-True ($failed.Log -ne $normal.Log -and (Get-ProofHash $normal.Log) -eq $normalHash) 'A later run replaced its predecessor.'
$paused = Invoke-Launcher $package $assets 37 -Pause
Assert-True ($paused.Exit -eq 37 -and $paused.Console.Contains('Game exit code: 37')) 'The default wrapper pause changed the child status.'
Write-Host 'PASS: stdout/stderr burst, UTF-8, normal/nonzero exit, separate retained logs'

# Literal cmd metacharacters must cross both the outer CMD and inner child.
$specialName = 'spaces & (brackets) [x] ! %PATH% ' + [string][char]0x416
$specialAssets = [IO.Path]::Combine($evidence, 'assets ' + $specialName)
Copy-Item -LiteralPath $assets -Destination $specialAssets -Recurse
$specialPackage = New-Package ('package ' + $specialName) $fixture
$specialLog = [IO.Path]::Combine($evidence, 'logs ' + $specialName)
$special = Invoke-Launcher $specialPackage $specialAssets 0 $specialLog -NoMovies
Assert-True ($special.Exit -eq 0) "Special paths failed: $($special.Console)"
$fixtureLine = @($special.Body -split '\r?\n' | Where-Object { $_.StartsWith('fixture=') })[0]
Assert-True ($null -ne $fixtureLine) "Special-path log is not readable: $($special.Console)"
$observed = $fixtureLine.Substring(8) | ConvertFrom-Json
$expected = @('-assets', $specialAssets, '-saves', [IO.Path]::Combine($specialPackage, 'saves'), '-movies=false')
Assert-True (($observed.Args | ConvertTo-Json -Compress) -eq ($expected | ConvertTo-Json -Compress)) 'Arguments were expanded, split or replaced.'
Assert-True ($observed.Directory -eq $specialPackage -and $special.Log.StartsWith($specialLog)) 'Working directory or explicit log path changed.'
Write-Host 'PASS: explicit assets override environment; spaces, Unicode, %, !, &, brackets; no-movies'

# Inspect actual native argv, not the launcher's diagnostic path. Both native
# boundaries previously corrupted a valid folder ending in a separator.
foreach ($basePath in @($assets, $specialAssets)) {
    foreach ($separator in @('\', '/')) {
        foreach ($promptMode in @($false, $true)) {
            $inputPath = $basePath + $separator
            $trailing = Invoke-Launcher $specialPackage $inputPath 0 ($specialLog + $separator) -Prompt:$promptMode
            Assert-True ($trailing.Exit -eq 0) "Trailing separator refused (prompt=$promptMode): $($trailing.Console)"
            $line = @($trailing.Body -split '\r?\n' | Where-Object { $_.StartsWith('fixture=') })[0]
            Assert-True ($null -ne $line) 'Trailing-separator probe did not reach the native executable.'
            $state = $line.Substring(8) | ConvertFrom-Json
            $want = @('-assets', [IO.Path]::GetFullPath($inputPath), '-saves', [IO.Path]::Combine($specialPackage, 'saves'))
            Assert-True (($state.Args | ConvertTo-Json -Compress) -eq ($want | ConvertTo-Json -Compress)) 'A terminal separator escaped its native quote or swallowed the next switch.'
            Assert-True ($state.Directory -eq $specialPackage) 'Trailing-separator probe changed its working directory.'
            if (-not $promptMode) { Assert-True ($trailing.Log.StartsWith($specialLog)) 'A trailing log separator broke the bootstrap argument.' }
        }
    }
}
Write-Host 'PASS: eight prompted/CLI terminal-separator argv probes with spaces and Unicode'

# A drive root must keep its final separator. Probe the complete wrapper's
# read-only validation, then invoke the actual production native-start builder
# with a controlled executable. No archive is created in a drive root, and no
# drive alias or junction is needed to exercise the native argument parser.
$driveRoot = [IO.Path]::GetPathRoot($evidence)
foreach ($promptMode in @($false, $true)) {
    $rootRefusal = Invoke-Launcher $package $driveRoot -Prompt:$promptMode
    # Package and fallback logs are on this same drive, so choosing its entire
    # root correctly refuses both output locations before creating a file.
    Assert-True ($rootRefusal.Exit -eq 1 -and -not $rootRefusal.Log -and
        $rootRefusal.Console.Contains('Output cannot be inside the selected game installation.')) 'Drive-root input was corrupted before its output fence.'
}
$parseTokens = $null
$parseErrors = $null
$launcherAst = [Management.Automation.Language.Parser]::ParseFile([IO.Path]::Combine($repository, 'launch-againrom.ps1'), [ref]$parseTokens, [ref]$parseErrors)
Assert-True ($parseErrors.Count -eq 0) 'Launcher parsing failed.'
foreach ($name in @('ConvertTo-NativePathArgument', 'New-GameStartInfo')) {
    $definitions = @($launcherAst.FindAll({ param($node)
        $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq $name
    }, $false))
    Assert-True ($definitions.Count -eq 1) "Expected one production function: $name"
    . ([scriptblock]::Create($definitions[0].Extent.Text))
}
foreach ($rootInput in @($driveRoot, $driveRoot.Replace('\', '/'))) {
    $rootLog = [IO.Path]::Combine($evidence, 'native-root-' + [Guid]::NewGuid().ToString('N') + '.log')
    [IO.File]::WriteAllText($rootLog, '', $utf8)
    $native = New-Object Diagnostics.Process
    $native.StartInfo = New-GameStartInfo $fixture $rootInput $driveRoot $rootLog $evidence $false
    $native.StartInfo.EnvironmentVariables['AGAINROM_LAUNCH_TEST_EXIT'] = '0'
    try {
        Assert-True $native.Start() 'Native root probe did not start.'
        if (-not $native.WaitForExit(30000)) { $native.Kill(); throw 'Native root probe did not finish.' }
        Assert-True ($native.ExitCode -eq 0) 'Native root probe failed.'
    } finally { $native.Dispose() }
    $line = @([IO.File]::ReadAllText($rootLog, $utf8) -split '\r?\n' | Where-Object { $_.StartsWith('fixture=') })[0]
    $state = $line.Substring(8) | ConvertFrom-Json
    $want = @('-assets', $rootInput, '-saves', $driveRoot)
    Assert-True (($state.Args | ConvertTo-Json -Compress) -eq ($want | ConvertTo-Json -Compress)) 'A drive root lost its separator or swallowed a following native argument.'
}
$lossLog = [IO.Path]::Combine($evidence, 'native-root-loss-control.log')
[IO.File]::WriteAllText($lossLog, '', $utf8)
$loss = New-Object Diagnostics.Process
$loss.StartInfo = New-GameStartInfo $fixture $driveRoot $driveRoot $lossLog $evidence $false
# Reintroduce the returned defect only in this controlled child's argument.
$loss.StartInfo.EnvironmentVariables['AGAINROM_LAUNCH_ASSETS_ARG'] = '"' + $driveRoot + '"'
$loss.StartInfo.EnvironmentVariables['AGAINROM_LAUNCH_TEST_EXIT'] = '0'
try {
    Assert-True $loss.Start() 'Native loss control did not start.'
    if (-not $loss.WaitForExit(30000)) { $loss.Kill(); throw 'Native loss control did not finish.' }
    Assert-True ($loss.ExitCode -eq 0) 'Native loss-control fixture failed.'
} finally { $loss.Dispose() }
$line = @([IO.File]::ReadAllText($lossLog, $utf8) -split '\r?\n' | Where-Object { $_.StartsWith('fixture=') })[0]
$lost = $line.Substring(8) | ConvertFrom-Json
Assert-True ($lost.Args.Count -ne 4 -and $lost.Args[1] -ne $driveRoot) 'The argv instrument did not detect an unescaped terminal separator.'
Write-Host 'PASS: drive-root wrapper refusals, production-builder native argv and quote-loss control'

$prompted = Invoke-Launcher $package $assets -Prompt
Assert-True ($prompted.Exit -eq 0 -and $prompted.Body.Contains("Assets: $assets")) 'Interactive asset selection failed.'
$cancelled = Invoke-Launcher $package '' -Prompt
Assert-True ($cancelled.Exit -eq 2 -and $cancelled.Body.Contains('Launch cancelled.') -and -not $cancelled.Body.Contains('fixture=')) 'Blank input did not cancel before launch.'
$missingAssets = Invoke-Launcher $package ([IO.Path]::Combine($evidence, 'absent install'))
Assert-True ($missingAssets.Exit -eq 1 -and $missingAssets.Body.Contains('Asset folder not found:')) 'Missing install failure is not retained.'
$incomplete = [IO.Path]::Combine($evidence, 'incomplete install')
[IO.Directory]::CreateDirectory($incomplete) | Out-Null
$missingArchive = Invoke-Launcher $package $incomplete
Assert-True ($missingArchive.Exit -eq 1 -and $missingArchive.Body.Contains('missing main.res')) 'Missing archive failure is not retained.'
Write-Host 'PASS: prompted folder, cancellation, missing assets and archive diagnostics'

$missingPackage = New-Package 'missing executable' ''
$missing = Invoke-Launcher $missingPackage $assets
Assert-True ($missing.Exit -eq 1 -and $missing.Body.Contains('againrom.exe is missing')) 'Missing executable failure is not retained.'
$brokenPackage = New-Package 'invalid executable' ''
[IO.File]::WriteAllText([IO.Path]::Combine($brokenPackage, 'againrom.exe'), 'controlled invalid executable')
$broken = Invoke-Launcher $brokenPackage $assets
Assert-True ($broken.Exit -ne 0 -and $broken.Body.Contains('Game exit code:') -and -not $broken.Body.Contains('fixture=')) 'OS launch failure lost its diagnostics/status.'
$guiPackage = New-Package 'gui subsystem' $guiFixture
$gui = Invoke-Launcher $guiPackage $assets 29
Assert-True ($gui.Exit -eq 29 -and $gui.Body.Contains('fixture complete') -and $gui.Body.Contains('Game exit code: 29')) 'GUI executable was not waited for or lost its output.'
Write-Host 'PASS: missing/invalid executable and GUI-subsystem exit/output'

$blockedLog = [IO.Path]::Combine($evidence, 'blocked log directory')
[IO.File]::WriteAllText($blockedLog, 'retain this file')
$fallback = Invoke-Launcher $package $assets 0 $blockedLog
Assert-True ($fallback.Exit -eq 0 -and $fallback.Body.Contains('Preferred log unavailable:') -and $fallback.Log.Contains('\local-temp\Againrom\logs\')) 'Unavailable log path did not retain a temporary log.'
$noLog = Invoke-Launcher $package $assets 0 $blockedLog -BlockFallback
Assert-True ($noLog.Exit -eq 1 -and -not $noLog.Log -and $noLog.Console.Contains('Game not started; launcher exit code: 1')) 'Without a writable log, the game should not start and the console should explain.'
$bootstrapPackage = New-Package 'missing powershell script' ''
$bootstrapScript = [IO.Path]::Combine($bootstrapPackage, 'launch-againrom.ps1')
# Remove only this newly copied ordinary file, never a directory or install.
[IO.File]::Delete($bootstrapScript)
$bootstrap = Invoke-Launcher $bootstrapPackage $assets -Pause
Assert-True ($bootstrap.Exit -ne 0 -and -not $bootstrap.Log -and $bootstrap.Console.Contains('launch-againrom.ps1')) 'Pre-script failure left no console diagnostic.'
$fenced = Invoke-Launcher $package $assets 0 ([IO.Path]::Combine($assets, 'logs'))
Assert-True ($fenced.Exit -eq 0 -and $fenced.Body.Contains('Output cannot be inside the selected game installation.')) 'Install log target was not fenced.'
$inside = New-Package 'package inside fixture install' $fixture
foreach ($archive in Get-ChildItem -LiteralPath $assets) { Copy-Item -LiteralPath $archive.FullName -Destination $inside }
$refused = Invoke-Launcher $inside $assets
Assert-True ($refused.Exit -eq 1 -and $refused.Body.Contains('Output cannot be inside a game installation:') -and -not $refused.Body.Contains('fixture=')) 'An install-contained package was launched.'
Assert-True (-not (Test-Path -LiteralPath ([IO.Path]::Combine($inside, 'logs')))) 'An install-contained package wrote local logs.'
$after = @(Get-ChildItem -LiteralPath $assets -Force -Recurse | ForEach-Object {
    $_.FullName + ':' + (Get-ProofHash $_.FullName)
}) -join "`n"
Assert-True ($before -eq $after) 'Asset fixture changed.'
Write-Host 'PASS: fallback/no-log/bootstrap errors, selected/other install fences, asset files unchanged'
Write-Host "Evidence: $evidence"
