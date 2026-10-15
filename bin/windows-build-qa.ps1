# Build, compress, and compare Windows executables in a disposable QA directory.
param(
    [Parameter(Mandatory=$true)][string]$BaselineRef,
    [string]$Output = "build/windows-qa"
)
$ErrorActionPreference = "Stop"
$repo = (Get-Location).Path
$out = [IO.Path]::GetFullPath((Join-Path $repo $Output))
New-Item -ItemType Directory -Force $out | Out-Null
Copy-Item "bin/compress-windows.ps1" "$out/compress-windows.ps1"

$helperFixture = Join-Path ([IO.Path]::GetTempPath()) ("zclone-compression-check-" + [Guid]::NewGuid().ToString() + ".exe")
$helperBytes = [byte[]]::new(131072)
$helperBytes[0] = 0x4d
$helperBytes[1] = 0x5a
[IO.File]::WriteAllBytes($helperFixture, $helperBytes)
try {
    & powershell.exe -NoLogo -NoProfile -NonInteractive -File bin/compress-windows.ps1 -Path $helperFixture
    if ($LASTEXITCODE -ne 0) { throw "Compression helper startup check failed" }
    & powershell.exe -NoLogo -NoProfile -NonInteractive -File bin/compress-windows.ps1 -Path $helperFixture -Undo
    if ($LASTEXITCODE -ne 0) { throw "Compression helper undo startup check failed" }
} finally { Remove-Item -LiteralPath $helperFixture }

@{
    baseline_commit = (git rev-parse $BaselineRef)
    candidate_commit = (git rev-parse HEAD)
    go = (go version)
    baseline_tags = @()
    candidate_tags = @("arrow_json_stdlib", "cmount")
    json_baseline_tags = @("cmount")
    upx = "5.1.1"
    winfsp = "2.1.25156"
    native_compression = @("XPRESS4K", "XPRESS8K", "LZX")
    preferred_native_compression = "XPRESS4K"
} | ConvertTo-Json | Set-Content (Join-Path $out "build-context.json")

function Get-VerifiedFile($Url, $Path, $Hash) {
    Invoke-WebRequest -Uri $Url -OutFile $Path
    if ((Get-FileHash -Algorithm SHA256 $Path).Hash.ToLowerInvariant() -ne $Hash) {
        throw "Checksum mismatch for $Url"
    }
}

Get-VerifiedFile "https://github.com/upx/upx/releases/download/v5.1.1/upx-5.1.1-win64.zip" "$out/upx.zip" "fa5380bca4c2718547aaa0134bc0d8a7fa27e102f0ac6371573d60d1c21d64de"
Expand-Archive -Path "$out/upx.zip" -DestinationPath "$out/upx" -Force
$upx = "$out/upx/upx-5.1.1-win64/upx.exe"
Get-VerifiedFile "https://github.com/winfsp/winfsp/releases/download/v2.1/winfsp-2.1.25156.msi" "$out/winfsp.msi" "073a70e00f77423e34bed98b86e600def93393ba5822204fac57a29324db9f7a"
$msi = Join-Path $out "winfsp.msi"
$installLog = Join-Path $out "winfsp-install.log"
Unblock-File $msi
$install = Start-Process msiexec.exe -ArgumentList "/i `"$msi`" /qn /norestart /L*v `"$installLog`"" -Wait -PassThru
if ($install.ExitCode -ne 0) { throw "WinFsp installation failed or requires reboot: $($install.ExitCode)" }

$env:GOPROXY = "off"
$env:GOSUMDB = "off"
$env:GOFLAGS = "-mod=vendor"
$env:GOTOOLCHAIN = "local"
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$version = (Get-Content VERSION -Raw).Trim()
$baseline = "$out/baseline-source"
git worktree add --detach $baseline $BaselineRef
if ($LASTEXITCODE -ne 0) { throw "Cannot create baseline checkout" }
try {
    Push-Location $baseline
    try {
        go build -buildvcs=false -trimpath -ldflags "-s -w -X zclone/fs.Version=$version" -o "$out/baseline.exe" .
        if ($LASTEXITCODE -ne 0) { throw "Baseline build failed" }
    } finally { Pop-Location }
} finally {
    git worktree remove $baseline
    if ($LASTEXITCODE -ne 0) { throw "Cannot remove baseline checkout" }
}
go build -buildvcs=false -trimpath -tags cmount -ldflags "-s -w -X zclone/fs.Version=$version" -o "$out/json-baseline.exe" .
if ($LASTEXITCODE -ne 0) { throw "JSON baseline build failed" }
go build -buildvcs=false -trimpath -tags arrow_json_stdlib,cmount -ldflags "-s -w -X zclone/fs.Version=$version" -o "$out/candidate.exe" .
if ($LASTEXITCODE -ne 0) { throw "Candidate build failed" }
python bin/analyze-executable.py --baseline "$out/json-baseline.exe" --candidate "$out/candidate.exe" --output "$out/machine-code.json"
if ($LASTEXITCODE -ne 0) { throw "Machine-code analysis failed" }
go test -tags arrow_json_stdlib -timeout 10m ./backend/drive ./backend/crypt ./backend/azureblob/... ./cmd/gui ./lib/http
if ($LASTEXITCODE -ne 0) { throw "Backend or GUI regression tests failed" }
go test -c -o "$out/arrow-default.test.exe" ./backend/azureblob/arrowlist
if ($LASTEXITCODE -ne 0) { throw "Arrow baseline test build failed" }
go test -c -tags arrow_json_stdlib -o "$out/arrow-stdlib.test.exe" ./backend/azureblob/arrowlist
if ($LASTEXITCODE -ne 0) { throw "Arrow candidate test build failed" }
python bin/compare-arrow-json.py --baseline "$out/arrow-default.test.exe" --candidate "$out/arrow-stdlib.test.exe" --output "$out/arrow-performance.json"
if ($LASTEXITCODE -ne 0) { throw "Arrow compatibility or listing performance regressed" }
Copy-Item "$out/candidate.exe" "$out/packed.exe"
& $upx -9 "$out/packed.exe"
if ($LASTEXITCODE -ne 0) { throw "UPX compression failed" }
& $upx -t "$out/packed.exe"
if ($LASTEXITCODE -ne 0) { throw "UPX integrity failed" }
python bin/compare-executables.py --baseline "$out/baseline.exe" --candidate "$out/candidate.exe" --json-baseline "$out/json-baseline.exe" --packed "$out/packed.exe" --compact --mount --output "$out/comparison.json"
if ($LASTEXITCODE -ne 0) { throw "Executable regression comparison failed" }

# Scan only disposable build artifacts with the runner's installed Defender engine.
$scanner = Get-ChildItem "$env:ProgramData/Microsoft/Windows Defender/Platform/*/MpCmdRun.exe" | Sort-Object FullName -Descending | Select-Object -First 1
if (-not $scanner) { throw "Defender scanner is unavailable" }
$defender = @{}
foreach ($name in @("candidate", "packed")) {
    & $scanner.FullName -Scan -ScanType 3 -File "$out/$name.exe" -DisableRemediation
    if ($LASTEXITCODE -ne 0) { throw "Defender scan failed for $name.exe" }
    $defender[$name] = "passed"
}
$defender | ConvertTo-Json | Set-Content "$out/defender.json"
