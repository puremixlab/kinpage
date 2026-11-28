[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$BundleDir,
    [string]$InstallerName = '',
    [string]$SevenZip = 'C:\Program Files\7-Zip\7z.exe',
    [string]$KuboScanPath = ''
)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSHOME 'Modules/Microsoft.PowerShell.Utility/Microsoft.PowerShell.Utility.psd1') -Force -ErrorAction Stop
Import-Module (Join-Path $PSHOME 'Modules/Microsoft.PowerShell.Security/Microsoft.PowerShell.Security.psd1') -Force -ErrorAction Stop
. (Join-Path $PSScriptRoot 'release-audit/archive-entries.ps1')
if (-not $InstallerName) { $InstallerName = 'kinpage-safe-setup-' + (([IO.File]::ReadAllText((Join-Path ([IO.Directory]::GetParent($PSScriptRoot).FullName) 'release.json')) | ConvertFrom-Json).version) + '-amd64.exe' }
$bundle = (Resolve-Path -LiteralPath $BundleDir).Path
if (-not (Test-Path -LiteralPath $SevenZip -PathType Leaf)) { throw "7-Zip not found: $SevenZip" }
if ([System.IO.Path]::GetFileName($InstallerName) -ne $InstallerName) { throw 'InstallerName must be a file name, not a path.' }
$installerName = $InstallerName
$installer = Join-Path $bundle $installerName
$files = @('Kinpage.exe', 'icon.png', 'ipfs.exe', 'kubo-LICENSE.txt', 'kubo-LICENSE-MIT.txt', 'kubo-LICENSE-APACHE.txt', 'LICENSE-PLANET.txt', 'THIRD-PARTY-NOTICES.txt', 'KUBO-BUILDINFO.txt', 'KUBO-DEPENDENCY-LICENSES.txt')
if (-not (Test-Path -LiteralPath $installer -PathType Leaf)) { throw "Installer not found: $installer" }

$header = & $SevenZip l -slt $installer
if ($LASTEXITCODE -ne 0 -or -not ($header -contains 'Type = Nsis')) { throw 'Installer is not a readable NSIS archive.' }
$listing = & $SevenZip l -slt -ba $installer
if ($LASTEXITCODE -ne 0) { throw 'Cannot read the installer with 7-Zip.' }
$entries = @($listing | Where-Object { $_ -like 'Path = *' } | ForEach-Object { $_.Substring(7) })

function Get-ArchiveEntryHash([string]$Name) {
    $p = New-Object System.Diagnostics.Process
    $p.StartInfo.FileName = $SevenZip
    # Windows paths cannot contain a double quote. Both arguments are quoted;
    # 7-Zip's -- ends its options so a payload name cannot become a flag.
    $p.StartInfo.Arguments = 'x -so -bd -bsp0 -bse0 -- "' + $installer + '" "' + $Name + '"'
    $p.StartInfo.UseShellExecute = $false
    $p.StartInfo.CreateNoWindow = $true
    $p.StartInfo.RedirectStandardOutput = $true
    if (-not $p.Start()) { throw "Failed to start 7-Zip for $Name" }
    try {
        $hash = [Security.Cryptography.SHA256]::Create()
        try { $digest = $hash.ComputeHash($p.StandardOutput.BaseStream) } finally { $hash.Dispose() }
        $p.WaitForExit()
        if ($p.ExitCode -ne 0) { throw "Could not read installer entry $Name (7-Zip exit $($p.ExitCode))." }
        return [BitConverter]::ToString($digest).Replace('-', '').ToLowerInvariant()
    } finally { $p.Dispose() }
}

$generatedUninstaller = Assert-InstallerArchiveEntries -Entries $entries -PayloadFiles $files -GetEntryHash ${function:Get-ArchiveEntryHash}
$results = @()
foreach ($name in $files) {
    if (@($entries | Where-Object { $_ -eq $name }).Count -ne 1) { throw "Installer must have exactly one $name entry." }
    $file = Join-Path $bundle $name
    if (-not (Test-Path -LiteralPath $file -PathType Leaf)) { throw "Bundle file is missing: $file" }
    $diskHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $file).Hash.ToLowerInvariant()
    $archiveHash = Get-ArchiveEntryHash $name
    if ($archiveHash -ne $diskHash) { throw "Installer and standalone payload differ: $name" }
    $results += [pscustomobject]@{ file = $name; sha256 = $diskHash }
}
$signatures = @('Kinpage.exe', $installerName, 'ipfs.exe') | ForEach-Object {
    $result = Get-AuthenticodeSignature -LiteralPath (Join-Path $bundle $_)
    [pscustomobject]@{ file = $_; status = [string]$result.Status }
}
$manualReviews = @([System.IO.File]::ReadLines((Join-Path $bundle 'KUBO-DEPENDENCY-LICENSES.txt')) | Where-Object { $_ -like 'REQUIRES MANUAL REVIEW:*' }).Count
$installerHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $installer).Hash.ToLowerInvariant()
$audit = [ordered]@{
    installer = $installerName
    installerSha256 = $installerHash
    payload = $results
    generatedUninstaller = $generatedUninstaller
    signatures = $signatures
    kuboModulesRequiringManualLicenseReview = $manualReviews
    note = 'Payload equality does not prove install/uninstall safety, Windows execution, or license compliance.'
}

if ($KuboScanPath) {
    $kubo = Join-Path $bundle 'ipfs.exe'
    $kuboHash = ($results | Where-Object { $_.file -eq 'ipfs.exe' }).sha256
    $scanPath = [System.IO.Path]::GetFullPath($KuboScanPath)
    if (Test-Path -LiteralPath $scanPath) { throw "Refusing to overwrite Kubo scan evidence: $scanPath" }
    $findings = & go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -mode=binary -format=json $kubo
    if ($LASTEXITCODE -ne 0 -or -not $findings) { throw 'Kubo vulnerability scan did not complete.' }
    if ((Get-FileHash -LiteralPath $installer -Algorithm SHA256).Hash -ine $installerHash -or
        (Get-FileHash -LiteralPath $kubo -Algorithm SHA256).Hash -ine $kuboHash) {
        throw 'Installer or verified Kubo payload changed during the vulnerability scan.'
    }
    $evidence = [ordered]@{
        scanner = 'govulncheck v1.9.0'
        scannedAt = [DateTime]::UtcNow.ToString('o')
        installerSha256 = $installerHash
        kuboSha256 = $kuboHash
        report = $findings -join "`n"
    } | ConvertTo-Json -Depth 4
    $stream = [System.IO.File]::Open($scanPath, [System.IO.FileMode]::CreateNew)
    try {
        $bytes = [System.Text.UTF8Encoding]::new($false).GetBytes($evidence + "`n")
        $stream.Write($bytes, 0, $bytes.Length)
    } finally { $stream.Dispose() }
    $audit['kuboSecurityEvidence'] = $scanPath
}
$audit | ConvertTo-Json -Depth 6
