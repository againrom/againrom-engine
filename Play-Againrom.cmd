@echo off
setlocal EnableExtensions DisableDelayedExpansion
rem Pass paths in the child environment, not through native argument quoting.
rem In particular, a quoted directory ending in \ must not escape its quote.
set "AGAINROM_LAUNCH_WRAPPER=1"
set "AGAINROM_LAUNCH_WRAPPER_ASSETS="
set "AGAINROM_LAUNCH_WRAPPER_LOGS="
set "AGAINROM_LAUNCH_WRAPPER_NO_MOVIES="
set "againrom_no_pause="
:arguments
if "%~1"=="" goto launch
if /i "%~1"=="-Assets" goto assets
if /i "%~1"=="-LogDirectory" goto logs
if /i "%~1"=="-NoMovies" goto no_movies
if /i "%~1"=="-NoPause" goto no_pause
if defined AGAINROM_LAUNCH_WRAPPER_ASSETS goto usage
set "againrom_argument=%~1"
if "%againrom_argument:~0,1%"=="-" goto usage
set "AGAINROM_LAUNCH_WRAPPER_ASSETS=%~1"
shift /1
goto arguments
:assets
if "%~2"=="" goto usage
set "AGAINROM_LAUNCH_WRAPPER_ASSETS=%~2"
shift /1
shift /1
goto arguments
:logs
if "%~2"=="" goto usage
set "AGAINROM_LAUNCH_WRAPPER_LOGS=%~2"
shift /1
shift /1
goto arguments
:no_movies
set "AGAINROM_LAUNCH_WRAPPER_NO_MOVIES=1"
shift /1
goto arguments
:no_pause
set "againrom_no_pause=1"
shift /1
goto arguments
:usage
echo Usage: Play-Againrom.cmd [-NoPause] [-Assets "folder"] [-LogDirectory "folder"] [-NoMovies]
set "againrom_exit=2"
goto finished
:launch
"%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe" -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0launch-againrom.ps1"
set "againrom_exit=%errorlevel%"
:finished
if defined againrom_no_pause exit /b %againrom_exit%
echo.
pause
exit /b %againrom_exit%
