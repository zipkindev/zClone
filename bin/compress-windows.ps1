# Apply reversible native compression to one installed Windows executable.
[CmdletBinding(DefaultParameterSetName = "Compress")]
param(
    [Parameter(Position = 0)][string]$Path = (Join-Path $PSScriptRoot "zclone.exe"),
    [Parameter(ParameterSetName = "Compress")]
    [ValidateSet("XPRESS4K", "XPRESS8K", "LZX")][string]$Algorithm = "XPRESS4K",
    [Parameter(Mandatory = $true, ParameterSetName = "Undo")][switch]$Undo
)
$ErrorActionPreference = "Stop"
$file = Get-Item -LiteralPath $Path -ErrorAction Stop
if ($file.PSIsContainer -or $file.Extension -ine ".exe") {
    throw "Path must name one Windows executable"
}
$before = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash
$compact = Join-Path $env:SystemRoot "System32/compact.exe"
if ($Undo) {
    $arguments = @("/U", "/F", "/EXE", $file.FullName)
} else {
    $arguments = @("/C", "/F", "/EXE:$Algorithm", $file.FullName)
}
$details = (& $compact @arguments 2>&1 | Out-String)
$result = $LASTEXITCODE
$after = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash
if ($before -ne $after) { throw "Executable content hash changed during native compression" }
if ($result -ne 0) { throw "Native compression failed (NTFS required): $details" }
$operation = "compress"
$selectedAlgorithm = $Algorithm
if ($Undo) { $operation = "undo"; $selectedAlgorithm = "none" }
[pscustomobject]@{
    path = $file.FullName
    operation = $operation
    algorithm = $selectedAlgorithm
    logical_bytes = $file.Length
    sha256 = $after.ToLowerInvariant()
    compact_output = $details.Trim()
} | ConvertTo-Json
