#Requires -Version 5.1
<#
.SYNOPSIS
    PowerShell counterpart of the Makefile: build, test, development and Docker on Windows.
.EXAMPLE
    .\build.ps1
    .\build.ps1 dev
    .\build.ps1 verify
    .\build.ps1 cross -Version 1.0.0
#>
[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('build', 'web', 'run', 'dev', 'tidy', 'cross', 'check', 'test', 'test-go', 'test-web',
        'test-e2e', 'e2e-install', 'verify', 'graphics', 'image', 'binary-image', 'clean')]
    [string] $Task = 'build',

    [string] $Version
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$App = 'kinex-mark'
$Cmd = './cmd/kinex-mark'
$Targets = 'linux/amd64', 'linux/arm64', 'windows/amd64', 'darwin/arm64'
$DevAddr = '127.0.0.1:8080'
$CoverTotal = '85'
$CoverPackage = '80'
$OnWindows = $env:OS -eq 'Windows_NT'
$Exe = if ($OnWindows) { '.exe' } else { '' }
# npm.cmd and npx.cmd start regardless of the execution policy.
$Npm = if ($OnWindows) { 'npm.cmd' } else { 'npm' }
$Npx = if ($OnWindows) { 'npx.cmd' } else { 'npx' }
$Python = if ($OnWindows) { 'python' } else { 'python3' }

Set-Location -LiteralPath $PSScriptRoot

function Invoke-Native {
    $file = $args[0]
    # An array is required: a single string would be split into characters when splatted.
    [string[]] $rest = @($args | Select-Object -Skip 1)
    & $file @rest
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed ($LASTEXITCODE): $file $rest"
    }
}

function Get-AppVersion {
    if ($Version) { return $Version }
    if (Get-Command git -ErrorAction SilentlyContinue) {
        try {
            $previous = $ErrorActionPreference
            $ErrorActionPreference = 'Continue'
            $described = & git describe --tags --always --dirty 2>$null
            if ($LASTEXITCODE -eq 0 -and $described) { return "$described" }
        }
        catch { }
        finally { $ErrorActionPreference = $previous }
    }
    return 'dev'
}

function Invoke-WebBuild {
    Push-Location web
    try {
        Invoke-Native $Npm ci
        Invoke-Native $Npm run build
    }
    finally { Pop-Location }
}

function Assert-WebBuild {
    if (-not (Test-Path -LiteralPath 'web/build/index.html')) { Invoke-WebBuild }
}

function Assert-WebDeps {
    if (-not (Test-Path -LiteralPath 'web/node_modules')) {
        Push-Location web
        try { Invoke-Native $Npm ci }
        finally { Pop-Location }
    }
}

function Invoke-WebScript {
    param([Parameter(Mandatory)] [string] $Name)
    Assert-WebDeps
    Push-Location web
    try { Invoke-Native $Npm run $Name }
    finally { Pop-Location }
}

# Environment variables would survive for the whole session, so they are restored.
function Invoke-GoBuild {
    param(
        [Parameter(Mandatory)] [string] $Output,
        [string] $TargetOS,
        [string] $TargetArch
    )
    $saved = @{ CGO_ENABLED = $env:CGO_ENABLED; GOOS = $env:GOOS; GOARCH = $env:GOARCH }
    try {
        $env:CGO_ENABLED = '0'
        if ($TargetOS) { $env:GOOS = $TargetOS }
        if ($TargetArch) { $env:GOARCH = $TargetArch }
        Invoke-Native go build -trimpath -ldflags "-s -w -X main.version=$(Get-AppVersion)" -o $Output $Cmd
    }
    finally {
        $env:CGO_ENABLED = $saved.CGO_ENABLED
        $env:GOOS = $saved.GOOS
        $env:GOARCH = $saved.GOARCH
    }
}

function Invoke-Check {
    Assert-WebBuild
    Invoke-WebScript -Name 'check'
    Invoke-WebScript -Name 'lint'
    [string[]] $goFiles = @('cmd', 'internal', 'tools') + @(Get-ChildItem -LiteralPath web -Filter '*.go' | ForEach-Object { $_.FullName })
    $unformatted = & gofmt -l @goFiles
    if ($LASTEXITCODE -ne 0) { throw "Command failed ($LASTEXITCODE): gofmt -l" }
    if ($unformatted) { throw "gofmt needed: $($unformatted -join ', ')" }
    Invoke-Native go vet ./...
    Invoke-Native go run ./tools/langcheck
}

function Invoke-GoTest {
    Assert-WebBuild
    # The race detector needs cgo and a C compiler; it runs in CI on Linux, not here.
    # Quoted: Windows PowerShell 5.1 splits an unquoted -flag=a.b argument at the dot.
    Invoke-Native go test '-covermode=atomic' '-coverprofile=coverage.out' ./...
    Invoke-Native go run ./tools/covercheck -profile coverage.out -total $CoverTotal -package $CoverPackage
}

function Invoke-E2E {
    $binary = (Resolve-Path -LiteralPath "dist/$App$Exe").Path
    $saved = $env:KINEX_MARK_BIN
    try {
        $env:KINEX_MARK_BIN = $binary
        Invoke-WebScript -Name 'test:e2e'
    }
    finally { $env:KINEX_MARK_BIN = $saved }
}

switch ($Task) {
    'tidy' {
        Invoke-Native go mod tidy
    }
    'web' {
        Invoke-WebBuild
    }
    'build' {
        Invoke-WebBuild
        Invoke-GoBuild -Output "dist/$App$Exe"
    }
    'run' {
        Invoke-WebBuild
        Invoke-GoBuild -Output "dist/$App$Exe"
        Invoke-Native "./dist/$App$Exe"
    }
    'dev' {
        Assert-WebBuild
        Assert-WebDeps
        # A fixed binary path: the firewall does not ask again, and only one process has to be stopped.
        $server = "dist/$App-dev$Exe"
        Invoke-GoBuild -Output $server
        $process = Start-Process -FilePath (Resolve-Path -LiteralPath $server).Path `
            -ArgumentList '-addr', $DevAddr -NoNewWindow -PassThru
        Push-Location web
        try {
            Invoke-Native $Npm run dev
        }
        finally {
            Pop-Location
            Stop-Process -Id $process.Id -ErrorAction SilentlyContinue
        }
    }
    'cross' {
        Invoke-WebBuild
        foreach ($target in $Targets) {
            $os, $arch = $target -split '/'
            $extension = if ($os -eq 'windows') { '.exe' } else { '' }
            Write-Host "$os/$arch"
            Invoke-GoBuild -Output "dist/$App-$os-$arch$extension" -TargetOS $os -TargetArch $arch
        }
    }
    'check' {
        Invoke-Check
    }
    'test' {
        Invoke-GoTest
        Invoke-WebScript -Name 'test:coverage'
    }
    'test-go' {
        Invoke-GoTest
    }
    'test-web' {
        Invoke-WebScript -Name 'test:coverage'
    }
    'e2e-install' {
        Assert-WebDeps
        Push-Location web
        try { Invoke-Native $Npx playwright install chromium }
        finally { Pop-Location }
    }
    'test-e2e' {
        Invoke-WebBuild
        Invoke-GoBuild -Output "dist/$App$Exe"
        Invoke-E2E
    }
    'verify' {
        Invoke-WebBuild
        Invoke-Check
        Invoke-GoTest
        Invoke-WebScript -Name 'test:coverage'
        Invoke-GoBuild -Output "dist/$App$Exe"
        Invoke-E2E
    }
    'graphics' {
        # Plain "python": the user site that holds google-genai is ignored under -I.
        Invoke-Native $Python graphics/generate_graphics.py
    }
    'image' {
        $tag = Get-AppVersion
        Invoke-Native docker build --target runtime --build-arg "VERSION=$tag" -t "${App}:$tag" .
    }
    'binary-image' {
        Invoke-Native docker build --target binary --build-arg "VERSION=$(Get-AppVersion)" --output type=local,dest=dist .
    }
    'clean' {
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue dist, web/build, web/.svelte-kit, web/coverage,
            web/test-results, web/playwright-report, web/blob-report, coverage.out, coverage.html
    }
}
