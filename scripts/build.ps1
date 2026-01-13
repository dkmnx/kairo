$ErrorActionPreference = 'SilentlyContinue'
$v = git describe --tags --always --dirty
if (!$v) { $v = 'dev' }
$c = git rev-parse --short HEAD
if (!$c) { $c = 'unknown' }
$d = Get-Date -Format 'yyyy-MM-dd'

Write-Host "Building kairo $v..."
mkdir -p dist -ErrorAction SilentlyContinue | Out-Null

$ldflags = "-X github.com/dkmnx/kairo/internal/version.Version=$v -X github.com/dkmnx/kairo/internal/version.Commit=$c -X github.com/dkmnx/kairo/internal/version.Date=$d"
go build -ldflags $ldflags -o dist/kairo.exe .
