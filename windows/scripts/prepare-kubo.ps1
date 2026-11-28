$ErrorActionPreference = 'Stop'
$projectRoot = [System.IO.Directory]::GetParent($PSScriptRoot).FullName
$cache = [System.IO.Path]::Combine($projectRoot, '.mygo', 'kubo')
$version = '0.43.2'
$archiveName = 'kubo_v' + $version + '_windows-amd64.zip'
$archive = [System.IO.Path]::Combine($cache, $archiveName)
$destination = [System.IO.Path]::Combine($projectRoot, 'resources', 'windows-amd64')
$expected = 'febadad25f817686c9edaffd74d7efea0333f6354fb910d8fc8f1949c4f968ad143a5e146a091c519ecbbd1cdf2b77ca008315419c54810e9acab753db8572e5'
[System.IO.Directory]::CreateDirectory($cache) | Out-Null
[System.IO.Directory]::CreateDirectory($destination) | Out-Null
function Get-KuboArchiveSHA512([string]$path) {
    $stream = [System.IO.File]::OpenRead($path)
    $sha = [System.Security.Cryptography.SHA512]::Create()
    try { return [System.BitConverter]::ToString($sha.ComputeHash($stream)).Replace('-', '').ToLowerInvariant() }
    finally { $sha.Dispose(); $stream.Dispose() }
}
if (-not [System.IO.File]::Exists($archive)) {
    $pending = Join-Path $cache ($archiveName + '.' + [Guid]::NewGuid().ToString('N') + '.partial')
    try {
        & curl.exe -fL --retry 2 -o $pending "https://github.com/ipfs/kubo/releases/download/v$version/$archiveName"
        if ($LASTEXITCODE -ne 0) { throw 'Kubo download failed.' }
        if ((Get-KuboArchiveSHA512 $pending) -cne $expected) { throw 'Kubo archive checksum mismatch.' }
        if (-not [System.IO.File]::Exists($archive)) { [System.IO.File]::Move($pending, $archive) }
    } finally {
        if ([System.IO.File]::Exists($pending)) { [System.IO.File]::Delete($pending) }
    }
}
$actual = Get-KuboArchiveSHA512 $archive
if ($actual -ne $expected) { throw 'Kubo archive checksum mismatch.' }
[System.Reflection.Assembly]::Load('System.IO.Compression.FileSystem, Version=4.0.0.0, Culture=neutral, PublicKeyToken=b77a5c561934e089') | Out-Null
$zip = [System.IO.Compression.ZipFile]::OpenRead($archive)
try {
    foreach ($name in @('ipfs.exe', 'LICENSE', 'LICENSE-APACHE', 'LICENSE-MIT')) {
        $entry = $zip.GetEntry("kubo/$name")
        if ($null -eq $entry) { throw "Missing Kubo archive entry: $name" }
        $targetName = if ($name -eq 'ipfs.exe') { $name } else { "kubo-$name.txt" }
        $target = [System.IO.Path]::Combine($destination, $targetName)
        $inputStream = $entry.Open()
        try { $outputStream = [System.IO.File]::Create($target); try { $inputStream.CopyTo($outputStream) } finally { $outputStream.Dispose() } } finally { $inputStream.Dispose() }
    }
} finally { $zip.Dispose() }
