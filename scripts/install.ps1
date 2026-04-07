# termshare Windows installer
# Usage (PowerShell): iwr https://raw.githubusercontent.com/ashutoshsinghai/termshare/main/scripts/install.ps1 | iex
# Usage (CMD):        powershell -ExecutionPolicy Bypass -Command "iwr https://raw.githubusercontent.com/ashutoshsinghai/termshare/main/scripts/install.ps1 | iex"

$ErrorActionPreference = "Stop"

$repo    = "ashutoshsinghai/termshare"
$installDir = "$env:LOCALAPPDATA\termshare"

# Detect architecture
$arch = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else {
    Write-Error "32-bit Windows is not supported."
    exit 1
}

# Get latest version
Write-Host "Fetching latest version..."
$release = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
$version = $release.tag_name

if (-not $version) {
    Write-Error "Could not determine latest version. Check your internet connection."
    exit 1
}

Write-Host "Installing termshare $version (windows/$arch)..."

# Download zip
$url     = "https://github.com/$repo/releases/download/$version/termshare_windows_$arch.zip"
$tmp     = New-TemporaryFile | ForEach-Object { $_.FullName + ".zip" }
$extract = New-Item -ItemType Directory -Path "$env:TEMP\termshare_install_$([System.IO.Path]::GetRandomFileName())"

try {
    Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing
    Expand-Archive -Path $tmp -DestinationPath $extract.FullName -Force

    # Install to LOCALAPPDATA\termshare
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir | Out-Null
    }
    Copy-Item "$($extract.FullName)\termshare.exe" "$installDir\termshare.exe" -Force
} finally {
    Remove-Item $tmp -ErrorAction SilentlyContinue
    Remove-Item $extract.FullName -Recurse -ErrorAction SilentlyContinue
}

# Add to user PATH if not already there
$userPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$installDir*") {
    [System.Environment]::SetEnvironmentVariable("PATH", "$userPath;$installDir", "User")
    Write-Host "Added $installDir to your PATH."
    Write-Host "Restart your terminal for PATH changes to take effect."
}

Write-Host ""
Write-Host "termshare $version installed to $installDir\termshare.exe"
Write-Host "Run 'termshare --help' to get started."
