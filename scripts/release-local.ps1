[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$Version,
    [string]$CorePath,
    [string]$Output,
    [string[]]$Targets = @("windows/amd64", "linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64"),
    [switch]$SkipDocker,
    [switch]$Upload
)

$ErrorActionPreference = "Stop"
$scriptRoot = $PSScriptRoot
if (-not $scriptRoot) { $scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path }
if (-not $CorePath) { $CorePath = Join-Path $scriptRoot "..\..\Rheovela" }
if (-not $Output) { $Output = Join-Path $scriptRoot "..\dist" }
if ($Version -notmatch '^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$') {
    throw "Version must be a semantic tag such as v1.63.0 (got '$Version')"
}
$CorePath = (Resolve-Path $CorePath).Path
$Backend = Join-Path $CorePath "backend"
if (-not (Test-Path (Join-Path $Backend "go.mod"))) {
    throw "core backend module not found at $Backend"
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "go is required on PATH"
}
New-Item -ItemType Directory -Force $Output | Out-Null
$Output = (Resolve-Path $Output).Path

$map = @{
    "windows/amd64" = @{ OS = "windows"; Arch = "amd64"; Ext = ".exe" }
    "linux/amd64"   = @{ OS = "linux"; Arch = "amd64"; Ext = "" }
    "linux/arm64"   = @{ OS = "linux"; Arch = "arm64"; Ext = "" }
    "darwin/amd64"  = @{ OS = "darwin"; Arch = "amd64"; Ext = "" }
    "darwin/arm64"  = @{ OS = "darwin"; Arch = "arm64"; Ext = "" }
}
$selected = @()
foreach ($target in $Targets) {
    if (-not $map.ContainsKey($target)) { throw "unknown target '$target'" }
    $selected += $target
}

$saved = @{
    GOOS = $env:GOOS; GOARCH = $env:GOARCH; CGO_ENABLED = $env:CGO_ENABLED
}
Push-Location $Backend
try {
    $ldflags = "-s -w -X main.version=$Version"
    foreach ($target in $selected) {
        $spec = $map[$target]
        $env:GOOS = $spec.OS
        $env:GOARCH = $spec.Arch
        $env:CGO_ENABLED = "0"
        $name = "rheo-$($spec.OS)-$($spec.Arch)$($spec.Ext)"
        $path = Join-Path $Output $name
        Write-Host "==> building $name"
        go build -trimpath -ldflags $ldflags -o $path ./cmd/rheo
        if ($LASTEXITCODE -ne 0) { throw "build failed for $target" }
    }
}
finally {
    Pop-Location
    foreach ($key in $saved.Keys) {
        if ($null -eq $saved[$key]) { Remove-Item "Env:$key" -ErrorAction SilentlyContinue }
        else { Set-Item "Env:$key" $saved[$key] }
    }
}

$manifest = Join-Path $Output "SHA256SUMS"
Get-ChildItem $Output -Filter "rheo-*" |
    Where-Object { $_.Name -notlike "*.sha256" -and $_.Name -ne "SHA256SUMS" } |
    ForEach-Object {
        $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLower()
        "$hash  $($_.Name)"
    } |
    Sort-Object |
    Set-Content $manifest -Encoding ascii
Write-Host "==> wrote $manifest"

if (-not $SkipDocker) {
    if (Get-Command docker -ErrorAction SilentlyContinue) {
        Write-Host "==> building ghcr.io/axisvela/rheo:$Version"
        docker build -f (Join-Path $CorePath "release\Dockerfile") -t "ghcr.io/axisvela/rheo:$Version" --build-arg "RHEO_VERSION=$Version" $CorePath
        if ($LASTEXITCODE -ne 0) { throw "docker build failed" }
        Write-Host "Docker image built locally. Push with: docker push ghcr.io/axisvela/rheo:$Version"
    }
    else {
        Write-Warning "docker not found; skipping image build"
    }
}

if ($Upload) {
    if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
        throw "gh is required for -Upload"
    }
    $files = Get-ChildItem $Output -Filter "rheo-*" | ForEach-Object { $_.FullName }
    Write-Host "==> uploading artifacts to release $Version"
    gh release upload $Version $files $manifest --clobber
    if ($LASTEXITCODE -ne 0) { throw "gh release upload failed" }
}

Write-Host "local release artifacts ready in $Output"
