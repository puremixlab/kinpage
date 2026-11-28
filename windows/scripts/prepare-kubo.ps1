$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSHOME 'Modules/Microsoft.PowerShell.Security/Microsoft.PowerShell.Security.psd1') -Force -ErrorAction Stop
$projectRoot = [System.IO.Directory]::GetParent($PSScriptRoot).FullName
$cache = [System.IO.Path]::Combine($projectRoot, '.mygo', 'kubo')
$version = '0.43.2'
$goVersion = 'go1.27.4'
$archiveName = "kubo-v$version-source.tar.gz"
$archive = [System.IO.Path]::Combine($cache, $archiveName)
$destination = [System.IO.Path]::Combine($projectRoot, 'resources', 'windows-amd64')
$sourceSHA512 = 'bfc6918ee74bd34345456a43ffe238f1abe194e656289bff3807042a1cedb6ac0ca36e3f815f444f015a886506bda4690cb127394810b338a7e771db5f02e9ff'
$binarySHA256 = 'ba75778d566239e1b0f068d6fdd8fd5490b5bba7306c507e33079881e5099e22'
[System.IO.Directory]::CreateDirectory($cache) | Out-Null
[System.IO.Directory]::CreateDirectory($destination) | Out-Null

function Assert-KuboSourceChecksum([string]$Path) {
    if ((Get-FileHash -LiteralPath $Path -Algorithm SHA512).Hash -ine $sourceSHA512) {
        throw "Kubo source archive SHA-512 mismatch: $Path"
    }
}

if (-not [System.IO.File]::Exists($archive)) {
    $pending = Join-Path $cache ($archiveName + '.' + [Guid]::NewGuid().ToString('N') + '.partial')
    try {
        & curl.exe -fsSL --retry 2 -o $pending "https://github.com/ipfs/kubo/releases/download/v$version/kubo-source.tar.gz"
        if ($LASTEXITCODE -ne 0) { throw 'Kubo source download failed.' }
        Assert-KuboSourceChecksum $pending
        try { [System.IO.File]::Move($pending, $archive) }
        catch [System.IO.IOException] { if (-not [System.IO.File]::Exists($archive)) { throw } }
    } finally { [System.IO.File]::Delete($pending) }
}
Assert-KuboSourceChecksum $archive

function Publish-KuboFile([string]$Source, [string]$Name, [string]$SHA256) {
    $target = Join-Path $destination $Name
    if ([System.IO.File]::Exists($target) -and (Get-FileHash -LiteralPath $target).Hash -ieq $SHA256) { return }
    try { [System.IO.File]::Move($Source, $target) }
    catch [System.IO.IOException] {
        if (-not [System.IO.File]::Exists($target)) { throw }
        [System.IO.File]::Replace($Source, $target, [NullString]::Value)
    }
}

$staging = [System.IO.Path]::GetFullPath((Join-Path $cache ('source-' + [Guid]::NewGuid().ToString('N'))))
if ([System.IO.Path]::GetDirectoryName($staging) -cne $cache) { throw 'Invalid Kubo staging path.' }
[System.IO.Directory]::CreateDirectory($staging) | Out-Null
try {
    & tar.exe -xzf $archive -C $staging
    if ($LASTEXITCODE -ne 0) { throw 'Kubo source extraction failed.' }
    $binary = Join-Path $staging 'ipfs.exe'
    # Keep the Kubo recipe independent of the app's GOFLAGS and workspace.
    $buildEnvironment = @{
        GOTOOLCHAIN = $goVersion; GOENV = 'off'; GOWORK = 'off'; GOFLAGS = ''; GOEXPERIMENT = ''
        GOOS = 'windows'; GOARCH = 'amd64'; GOAMD64 = 'v1'; CGO_ENABLED = '0'
        # Apply the module lock patch in the extracted tree, including when the
        # staging directory sits under a Git working tree.
        GIT_CEILING_DIRECTORIES = $cache; GIT_DIR = ''; GIT_WORK_TREE = ''
    }
    $previous = @{}
    foreach ($name in $buildEnvironment.Keys) { $previous[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
    Push-Location -LiteralPath $staging
    try {
        foreach ($name in $buildEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name, $buildEnvironment[$name], 'Process') }
        & git apply --no-index --whitespace=error -- (Join-Path $PSScriptRoot 'release-audit/kubo-dependencies.patch')
        if ($LASTEXITCODE -ne 0) { throw "Kubo v$version dependency patch did not apply to the pinned source." }
        & go build -mod=readonly -trimpath -buildvcs=false -o $binary ./cmd/ipfs
        if ($LASTEXITCODE -ne 0) { throw "Kubo v$version build with $goVersion failed." }
    } finally {
        foreach ($name in $previous.Keys) { [Environment]::SetEnvironmentVariable($name, $previous[$name], 'Process') }
        Pop-Location
    }
    $actual = (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -cne $binarySHA256) { throw "Rebuilt Kubo SHA-256 differs from the reviewed binary: $actual" }

    foreach ($name in @('LICENSE', 'LICENSE-APACHE', 'LICENSE-MIT')) {
        $license = Join-Path $staging "cmd/ipfs/dist/$name"
        Publish-KuboFile $license "kubo-$name.txt" (Get-FileHash -LiteralPath $license).Hash
    }
    Publish-KuboFile $binary 'ipfs.exe' $binarySHA256
    Write-Host "Prepared Kubo v$version from source with $goVersion (SHA-256 $binarySHA256)."
} finally { [System.IO.Directory]::Delete($staging, $true) }
