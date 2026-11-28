[CmdletBinding()]
param(
    [string]$Makensis = (Join-Path $env:LOCALAPPDATA 'mygo/nsis-3.13/makensis.exe'),
    [string]$SevenZip = 'C:\Program Files\7-Zip\7z.exe'
)
$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath $Makensis -PathType Leaf)) { throw 'NSIS 3.13 compiler not found.' }
$projectRoot = [System.IO.Directory]::GetParent($PSScriptRoot).FullName
$fixture = Join-Path $projectRoot ('.mygo/release-audit-tests/payload-' + [Guid]::NewGuid().ToString('N'))
[System.IO.Directory]::CreateDirectory($fixture) | Out-Null
$payload = @('Kinpage.exe', 'ipfs.exe', 'icon.png', 'kubo-LICENSE.txt', 'kubo-LICENSE-MIT.txt', 'kubo-LICENSE-APACHE.txt', 'LICENSE-PLANET.txt', 'THIRD-PARTY-NOTICES.txt', 'KUBO-BUILDINFO.txt', 'KUBO-DEPENDENCY-LICENSES.txt')
foreach ($name in $payload) { [System.IO.File]::WriteAllText((Join-Path $fixture $name), "test-only: $name", (New-Object System.Text.UTF8Encoding($false))) }
$installer = Join-Path $fixture 'kinpage-setup-0.1.0-amd64.exe'
$harness = Join-Path $PSScriptRoot 'release-audit/payload-harness.nsi'
& $Makensis "/DOUTPUT=$installer" "/DFIXTURE_PATH=$fixture" $harness | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Failed to compile synthetic NSIS archive.' }
& powershell -NoProfile -File (Join-Path $PSScriptRoot 'audit-release.ps1') -BundleDir $fixture -InstallerName 'kinpage-setup-0.1.0-amd64.exe' -SevenZip $SevenZip | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Audit rejected an unchanged synthetic payload.' }
Write-Host 'PASS: exact synthetic payload accepted.'

Add-Type -AssemblyName System.IO.Compression, System.IO.Compression.FileSystem
$msix = Join-Path $fixture 'fixture.msix'
$zip = [IO.Compression.ZipFile]::Open($msix, [IO.Compression.ZipArchiveMode]::Create)
try {
    foreach ($name in $payload) {
        [void][IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zip, (Join-Path $fixture $name), $name)
    }
} finally { $zip.Dispose() }
$combined = & powershell -NoProfile -File (Join-Path $PSScriptRoot 'audit-release.ps1') -BundleDir $fixture -InstallerName 'kinpage-setup-0.1.0-amd64.exe' -SevenZip $SevenZip -MsixPath $msix
if ($LASTEXITCODE -ne 0 -or ($combined | ConvertFrom-Json).msixSha256 -ine (Get-FileHash -LiteralPath $msix).Hash) {
    throw 'Audit did not attest the matching MSIX payload.'
}
Write-Host 'PASS: matching MSIX payload accepted.'

$zip = [IO.Compression.ZipFile]::Open($msix, [IO.Compression.ZipArchiveMode]::Update)
try {
    $zip.GetEntry('ipfs.exe').Delete()
    $different = $zip.CreateEntry('ipfs.exe').Open()
    try { $different.WriteByte(0) } finally { $different.Dispose() }
} finally { $zip.Dispose() }
$oldPreference = $ErrorActionPreference
try {
    $ErrorActionPreference = 'Continue'
    $negative = & powershell -NoProfile -File (Join-Path $PSScriptRoot 'audit-release.ps1') -BundleDir $fixture -InstallerName 'kinpage-setup-0.1.0-amd64.exe' -SevenZip $SevenZip -MsixPath $msix 2>&1 | Out-String
    $negativeExit = $LASTEXITCODE
} finally { $ErrorActionPreference = $oldPreference }
if ($negativeExit -eq 0 -or $negative -notmatch 'MSIX and standalone payload differ: ipfs\.exe') { throw "Audit did not reject the modified MSIX payload: $negative" }
Write-Host 'PASS: changed MSIX Kubo payload rejected.'

[System.IO.File]::WriteAllText((Join-Path $fixture 'ipfs.exe'), 'different data')
$oldPreference = $ErrorActionPreference
try {
    $ErrorActionPreference = 'Continue'
    $negative = & powershell -NoProfile -File (Join-Path $PSScriptRoot 'audit-release.ps1') -BundleDir $fixture -InstallerName 'kinpage-setup-0.1.0-amd64.exe' -SevenZip $SevenZip 2>&1 | Out-String
    $negativeExit = $LASTEXITCODE
} finally { $ErrorActionPreference = $oldPreference }
if ($negativeExit -eq 0 -or $negative -notmatch 'Installer and standalone payload differ: ipfs\.exe') { throw "Audit did not reject the modified payload for the expected reason: $negative" }
Write-Host 'PASS: modified standalone payload rejected.'
Write-Host "Synthetic fixture retained at $fixture"
