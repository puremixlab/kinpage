[CmdletBinding()]
param(
    [string]$PayloadDir = '',
    [string]$OutputFile = '',
    [string]$ConfigPath = '',
    [string]$PackageName = '',
    [string]$Publisher = '',
    [string]$PublisherDisplayName = '',
    [string]$MsixVersion = '',
    [string]$MakeAppx = '',
    [switch]$Store,
    [switch]$AllowPrerelease,
    [switch]$StageOnly,
    [string]$StageDir = '',
    [switch]$SkipVersionProbe
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

Import-Module (Join-Path $PSHOME 'Modules/Microsoft.PowerShell.Utility/Microsoft.PowerShell.Utility.psd1') -Force -ErrorAction Stop
Import-Module (Join-Path $PSHOME 'Modules/Microsoft.PowerShell.Management/Microsoft.PowerShell.Management.psd1') -Force -ErrorAction Stop

$projectRoot = [System.IO.Directory]::GetParent($PSScriptRoot).FullName
$msixDir = Join-Path $projectRoot 'msix'
if ([string]::IsNullOrWhiteSpace($ConfigPath)) {
    $ConfigPath = Join-Path $msixDir 'msix.config.json'
}
$templatePath = Join-Path $msixDir 'AppxManifest.template.xml'
$defaultIconPath = Join-Path $projectRoot 'resources/icon.png'

if (-not (Test-Path -LiteralPath $ConfigPath -PathType Leaf)) {
    throw "MSIX config not found: $ConfigPath"
}
if (-not (Test-Path -LiteralPath $templatePath -PathType Leaf)) {
    throw "MSIX manifest template not found: $templatePath"
}
if (-not (Test-Path -LiteralPath $defaultIconPath -PathType Leaf)) {
    throw "Source app icon not found: $defaultIconPath"
}

$releaseVersion = ([IO.File]::ReadAllText((Join-Path $projectRoot 'release.json')) | ConvertFrom-Json).version
if ($releaseVersion -notmatch '^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$') {
    throw "Invalid release version in release.json: '$releaseVersion'."
}
$versionMatch = [regex]::Match($releaseVersion, '^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$')
foreach ($idx in 1..3) {
    if ([uint64]$versionMatch.Groups[$idx].Value -gt 4294967295) {
        throw "Invalid release version in release.json: '$releaseVersion'."
    }
}
$prereleasePart = $versionMatch.Groups[4].Value
$isPrereleaseVersion = -not [string]::IsNullOrEmpty($prereleasePart)
if ($isPrereleaseVersion) {
    foreach ($ident in $prereleasePart.Split('.')) {
        if ($ident -match '^\d+$' -and $ident.Length -gt 1 -and $ident.StartsWith('0')) {
            throw "Invalid release version in release.json: '$releaseVersion'."
        }
    }
}

function ConvertTo-XmlText {
    param([string]$Value)
    return [System.Security.SecurityElement]::Escape($Value)
}

function Resolve-MsixVersion {
    param(
        [string]$AppVersion,
        [string]$OverrideVersion,
        [bool]$IsStoreMode,
        [bool]$IsPrerelease,
        [bool]$AllowPrereleaseFlag
    )

    if ($IsStoreMode -and $IsPrerelease -and -not $AllowPrereleaseFlag) {
        throw "Store MSIX packaging rejected prerelease version '$AppVersion'. Update windows/release.json to a stable version or pass -AllowPrerelease explicitly."
    }

    $candidate = $OverrideVersion
    if ([string]::IsNullOrWhiteSpace($candidate)) {
        $base = ($AppVersion -split '-', 2)[0]
        $candidate = "$base.0"
    }

    if ($candidate -notmatch '^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$') {
        throw "Invalid MSIX version '$candidate': expected four dot-separated uint16 numbers (Major.Minor.Build.Revision)."
    }

    $parts = @($candidate.Split('.') | ForEach-Object { [int]$_ })
    foreach ($part in $parts) {
        if ($part -lt 0 -or $part -gt 65535) {
            throw "Invalid MSIX version '$candidate': each segment must be between 0 and 65535."
        }
    }
    if ($parts[0] -eq 0 -and $parts[1] -eq 0 -and $parts[2] -eq 0 -and $parts[3] -eq 0) {
        throw "Invalid MSIX version '$candidate': version cannot be 0.0.0.0."
    }
    if ($IsStoreMode -and $parts[3] -ne 0) {
        throw "Invalid Store MSIX version '$candidate': Microsoft Store reserves the 4th segment (Revision) and requires it to be 0."
    }

    return $candidate
}

function Find-MakeAppx {
    param([string]$RequestedPath)
    if (-not [string]::IsNullOrWhiteSpace($RequestedPath)) {
        if (-not (Test-Path -LiteralPath $RequestedPath -PathType Leaf)) {
            throw "Specified makeappx.exe not found: $RequestedPath"
        }
        return (Resolve-Path -LiteralPath $RequestedPath).Path
    }

    $cmd = Get-Command 'makeappx.exe' -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }

    $kitsBin = 'C:\Program Files (x86)\Windows Kits\10\bin'
    if (Test-Path -LiteralPath $kitsBin -PathType Container) {
        $candidates = @(
            Get-ChildItem -LiteralPath $kitsBin -Directory -ErrorAction SilentlyContinue |
                Where-Object { $_.Name -match '^\d+\.\d+\.\d+\.\d+$' } |
                Sort-Object { [version]$_.Name } -Descending |
                ForEach-Object { Join-Path $_.FullName 'x64\makeappx.exe' } |
                Where-Object { Test-Path -LiteralPath $_ -PathType Leaf }
        )
        if ($candidates.Count -gt 0) {
            return $candidates[0]
        }
    }

    throw 'makeappx.exe was not found. Install the Windows 10/11 SDK or pass -MakeAppx <path-to-makeappx.exe>.'
}

function New-ResizedAsset {
    param(
        [System.Drawing.Image]$SourceImage,
        [int]$Width,
        [int]$Height,
        [string]$DestinationPath,
        [double]$PaddingRatio = 0.0
    )

    $bitmap = New-Object System.Drawing.Bitmap($Width, $Height, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    try {
        $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
        try {
            $graphics.Clear([System.Drawing.Color]::Transparent)
            $graphics.CompositingMode = [System.Drawing.Drawing2D.CompositingMode]::SourceOver
            $graphics.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
            $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
            $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
            $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality

            $maxContentW = [int][Math]::Floor($Width * (1.0 - 2.0 * $PaddingRatio))
            $maxContentH = [int][Math]::Floor($Height * (1.0 - 2.0 * $PaddingRatio))
            $side = [Math]::Min($maxContentW, $maxContentH)
            $x = [int][Math]::Floor(($Width - $side) / 2.0)
            $y = [int][Math]::Floor(($Height - $side) / 2.0)

            $graphics.DrawImage($SourceImage, $x, $y, $side, $side)
        } finally {
            $graphics.Dispose()
        }
        $bitmap.Save($DestinationPath, [System.Drawing.Imaging.ImageFormat]::Png)
    } finally {
        $bitmap.Dispose()
    }
}

$config = [IO.File]::ReadAllText($ConfigPath, [Text.UTF8Encoding]::new($false)) | ConvertFrom-Json

$effectivePackageName = if (-not [string]::IsNullOrWhiteSpace($PackageName)) { $PackageName } else { [string]$config.packageName }
$effectivePublisher = if (-not [string]::IsNullOrWhiteSpace($Publisher)) { $Publisher } else { [string]$config.publisher }
$effectivePublisherDisplay = if (-not [string]::IsNullOrWhiteSpace($PublisherDisplayName)) { $PublisherDisplayName } else { [string]$config.publisherDisplayName }
$effectiveDisplayName = [string]$config.displayName
$effectiveDescription = [string]$config.description
$effectiveBgColor = [string]$config.backgroundColor
$effectiveMinVersion = [string]$config.minVersion
$effectiveMaxVersion = [string]$config.maxVersionTested
$effectiveLanguages = @($config.languages)

if ($effectivePackageName -notmatch '^[A-Za-z0-9.-]{3,50}$') {
    throw "Invalid MSIX packageName '$effectivePackageName': expected 3-50 ASCII letters, digits, dots, or hyphens."
}
if ($effectivePublisher -notmatch '^CN=.+') {
    throw "Invalid MSIX publisher '$effectivePublisher': expected an X.500 distinguished name starting with 'CN='."
}
if ($effectiveLanguages.Count -eq 0) {
    throw 'MSIX config must declare at least one language.'
}

$isPlaceholderPublisher = ($effectivePublisher -eq 'CN=KinpageLocalDev') -or ($effectivePublisher -match 'REPLACE-WITH')
if ($Store -and $isPlaceholderPublisher) {
    throw "Store MSIX packaging requires your real Partner Center Publisher ID (current: '$effectivePublisher'). Update windows/msix/msix.config.json or pass -Publisher."
}
if (-not $Store -and $isPlaceholderPublisher) {
    Write-Warning "Using local development Publisher '$effectivePublisher'. Replace it with your Partner Center Publisher ID before submitting to Microsoft Store."
}

$resolvedMsixVersion = Resolve-MsixVersion `
    -AppVersion $releaseVersion `
    -OverrideVersion $MsixVersion `
    -IsStoreMode ([bool]$Store) `
    -IsPrerelease $isPrereleaseVersion `
    -AllowPrereleaseFlag ([bool]$AllowPrerelease)

if ([string]::IsNullOrWhiteSpace($PayloadDir)) {
    $PayloadDir = Join-Path $projectRoot 'build/windows-amd64'
}
if (-not (Test-Path -LiteralPath $PayloadDir -PathType Container)) {
    throw "Build payload directory not found: $PayloadDir. Run 'bun run build' from windows/ first, or pass -PayloadDir."
}

$requiredPayloadFiles = @(
    'Kinpage.exe',
    'ipfs.exe',
    'icon.png',
    'kubo-LICENSE.txt',
    'kubo-LICENSE-MIT.txt',
    'kubo-LICENSE-APACHE.txt',
    'LICENSE-PLANET.txt',
    'THIRD-PARTY-NOTICES.txt',
    'KUBO-BUILDINFO.txt',
    'KUBO-DEPENDENCY-LICENSES.txt'
)

foreach ($fileName in $requiredPayloadFiles) {
    $sourcePath = Join-Path $PayloadDir $fileName
    if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) {
        throw "Required payload file missing in ${PayloadDir}: $fileName"
    }
}

$buildId = [Guid]::NewGuid().ToString('N')
if ([string]::IsNullOrWhiteSpace($StageDir)) {
    $StageDir = Join-Path $projectRoot ('.mygo/msix-staging/' + $buildId)
}
if ([string]::IsNullOrWhiteSpace($OutputFile)) {
    $outDir = Join-Path $projectRoot ('.mygo/msix/' + $buildId)
    [System.IO.Directory]::CreateDirectory($outDir) | Out-Null
    $OutputFile = Join-Path $outDir ("kinpage-$releaseVersion-x64.msix")
} else {
    $outParent = Split-Path -Parent $OutputFile
    if (-not [string]::IsNullOrWhiteSpace($outParent)) {
        [System.IO.Directory]::CreateDirectory($outParent) | Out-Null
    }
}

[System.IO.Directory]::CreateDirectory($StageDir) | Out-Null
$assetsDir = Join-Path $StageDir 'Assets'
[System.IO.Directory]::CreateDirectory($assetsDir) | Out-Null

if (-not $SkipVersionProbe) {
    $prevAppData = $env:APPDATA
    $prevLocalAppData = $env:LOCALAPPDATA
    $probeProfile = Join-Path $projectRoot ('.mygo/msix-staging/version-probe-' + $buildId)
    [System.IO.Directory]::CreateDirectory($probeProfile) | Out-Null
    $versionProc = $null
    try {
        $env:APPDATA = $probeProfile
        $env:LOCALAPPDATA = $probeProfile
        $versionProc = New-Object System.Diagnostics.Process
        $versionProc.StartInfo.FileName = Join-Path $PayloadDir 'Kinpage.exe'
        $versionProc.StartInfo.Arguments = '--planet-version'
        $versionProc.StartInfo.WorkingDirectory = $PayloadDir
        $versionProc.StartInfo.UseShellExecute = $false
        $versionProc.StartInfo.CreateNoWindow = $true
        $versionProc.StartInfo.RedirectStandardOutput = $true
        if (-not $versionProc.Start()) {
            throw 'Could not start built app for version verification.'
        }
        if (-not $versionProc.WaitForExit(10000)) {
            $versionProc.Kill()
            throw 'Built app timed out during --planet-version check.'
        }
        $observedVersion = $versionProc.StandardOutput.ReadToEnd().Trim()
        if ($versionProc.ExitCode -ne 0 -or $observedVersion -cne $releaseVersion) {
            throw "Payload version mismatch: exit=[$($versionProc.ExitCode)], observed=[$observedVersion], expected=[$releaseVersion]. Rebuild with 'bun run build' first."
        }
    } finally {
        if ($versionProc) { $versionProc.Dispose() }
        $env:APPDATA = $prevAppData
        $env:LOCALAPPDATA = $prevLocalAppData
        Remove-Item -LiteralPath $probeProfile -Recurse -Force -ErrorAction SilentlyContinue
    }
}

foreach ($fileName in $requiredPayloadFiles) {
    Copy-Item -LiteralPath (Join-Path $PayloadDir $fileName) -Destination (Join-Path $StageDir $fileName) -Force
}

$assetSpecs = @(
    @{ Name = 'StoreLogo.png'; Width = 50; Height = 50; Padding = 0.0 },
    @{ Name = 'Square44x44Logo.png'; Width = 44; Height = 44; Padding = 0.0 },
    @{ Name = 'Square44x44Logo.targetsize-48.png'; Width = 48; Height = 48; Padding = 0.0 },
    @{ Name = 'Square44x44Logo.targetsize-48_altform-unplated.png'; Width = 48; Height = 48; Padding = 0.0 },
    @{ Name = 'SmallTile.png'; Width = 71; Height = 71; Padding = 0.08 },
    @{ Name = 'Square150x150Logo.png'; Width = 150; Height = 150; Padding = 0.12 },
    @{ Name = 'Wide310x150Logo.png'; Width = 310; Height = 150; Padding = 0.14 },
    @{ Name = 'LargeTile.png'; Width = 310; Height = 310; Padding = 0.14 },
    @{ Name = 'SplashScreen.png'; Width = 620; Height = 300; Padding = 0.18 }
)

$customAssetsDir = Join-Path $msixDir 'Assets'
Add-Type -AssemblyName System.Drawing
$sourceIcon = [System.Drawing.Image]::FromFile($defaultIconPath)
try {
    foreach ($spec in $assetSpecs) {
        $customAsset = Join-Path $customAssetsDir $spec.Name
        $destAsset = Join-Path $assetsDir $spec.Name
        if (Test-Path -LiteralPath $customAsset -PathType Leaf) {
            Copy-Item -LiteralPath $customAsset -Destination $destAsset -Force
        } else {
            New-ResizedAsset -SourceImage $sourceIcon -Width $spec.Width -Height $spec.Height -DestinationPath $destAsset -PaddingRatio $spec.Padding
        }
    }
} finally {
    $sourceIcon.Dispose()
}

$resourcesXml = ($effectiveLanguages | ForEach-Object {
    "    <Resource Language=`"$(ConvertTo-XmlText $_)`" />"
}) -join "`n"

$manifestContent = [IO.File]::ReadAllText($templatePath, [Text.UTF8Encoding]::new($false))
$replacements = @{
    '{{PACKAGE_NAME}}' = ConvertTo-XmlText $effectivePackageName
    '{{PUBLISHER}}' = ConvertTo-XmlText $effectivePublisher
    '{{MSIX_VERSION}}' = ConvertTo-XmlText $resolvedMsixVersion
    '{{ARCHITECTURE}}' = 'x64'
    '{{DISPLAY_NAME}}' = ConvertTo-XmlText $effectiveDisplayName
    '{{PUBLISHER_DISPLAY_NAME}}' = ConvertTo-XmlText $effectivePublisherDisplay
    '{{DESCRIPTION}}' = ConvertTo-XmlText $effectiveDescription
    '{{BACKGROUND_COLOR}}' = ConvertTo-XmlText $effectiveBgColor
    '{{MIN_VERSION}}' = ConvertTo-XmlText $effectiveMinVersion
    '{{MAX_VERSION_TESTED}}' = ConvertTo-XmlText $effectiveMaxVersion
    '{{RESOURCES_XML}}' = $resourcesXml
}

foreach ($entry in $replacements.GetEnumerator()) {
    $manifestContent = $manifestContent.Replace($entry.Key, $entry.Value)
}
$manifestContent = $manifestContent.Replace("`r`n", "`n")
$manifestPath = Join-Path $StageDir 'AppxManifest.xml'
[IO.File]::WriteAllText($manifestPath, $manifestContent, [Text.UTF8Encoding]::new($false))

if ($StageOnly) {
    Write-Host "MSIX staging directory ready: $StageDir"
    Write-Host "MSIX version: $resolvedMsixVersion (app version: $releaseVersion)"
    exit 0
}

$makeAppxPath = Find-MakeAppx -RequestedPath $MakeAppx
Write-Host "Packing MSIX with $makeAppxPath ..."
& $makeAppxPath pack /d $StageDir /p $OutputFile /o
if ($LASTEXITCODE -ne 0) {
    throw "makeappx.exe failed with exit code $LASTEXITCODE."
}

$hash = (Get-FileHash -LiteralPath $OutputFile -Algorithm SHA256).Hash.ToLowerInvariant()
Write-Host "MSIX package created: $OutputFile"
Write-Host "MSIX version: $resolvedMsixVersion (app version: $releaseVersion)"
Write-Host "SHA-256: $hash"
