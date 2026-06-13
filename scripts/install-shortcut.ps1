# install-shortcut.ps1 - put a "keys7" shortcut on the current user's Desktop.
#
# The shortcut launches keys7.ps1 in a PowerShell console (which renders the
# Bubble Tea TUI's ANSI/256-colour output). Generic: no hard-coded user or
# path - pass the install directory.
#
#   powershell -ExecutionPolicy Bypass -File install-shortcut.ps1 -InstallDir "C:\Users\me\Documents\keys7"

param(
    [Parameter(Mandatory = $true)][string]$InstallDir
)

$ErrorActionPreference = "Stop"

$launcher = Join-Path $InstallDir "keys7.ps1"
if (-not (Test-Path $launcher)) { throw "keys7.ps1 not found in $InstallDir" }

$desktop = [Environment]::GetFolderPath("Desktop")
$lnkPath = Join-Path $desktop "keys7.lnk"

# Target powershell.exe directly (real path under System32). We deliberately do
# NOT target wt.exe: the WindowsApps\wt.exe is an execution alias (a reparse
# point), which a .lnk can't resolve as a target - it fails with 0x80070002.
# PowerShell's console handles the TUI's ANSI/256-colour output fine.
$target = "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"
$arguments = "-NoExit -ExecutionPolicy Bypass -File `"$launcher`""

$shell = New-Object -ComObject WScript.Shell
$lnk = $shell.CreateShortcut($lnkPath)
$lnk.TargetPath       = $target
$lnk.Arguments        = $arguments
$lnk.WorkingDirectory = $InstallDir
$lnk.IconLocation     = (Join-Path $InstallDir "keys7.exe")  # use the exe's icon
$lnk.Description      = "Launch a live keys7 session"
$lnk.Save()

Write-Host "Desktop shortcut created: $lnkPath" -ForegroundColor Green
