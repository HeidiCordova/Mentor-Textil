[CmdletBinding()]
param(
    [string]$OutputArchive = ".\nodered-production-hardening-v1.tgz",
    [switch]$Force
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$artifactRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$customNodeRoot = [IO.Path]::GetFullPath(
    (Join-Path $artifactRoot `
        "..\custom-nodes\node-red-contrib-services-mentor")
)
$noderedRoot = [IO.Path]::GetFullPath((Join-Path $artifactRoot ".."))
$outputPath = [IO.Path]::GetFullPath($OutputArchive)
$checksumPath = "$outputPath.sha256"
$tempBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$tempRoot = Join-Path $tempBase (
    "mentor-nodered-hardening-bundle-" + [Guid]::NewGuid().ToString("N")
)
$bundleRoot = Join-Path $tempRoot "nodered-production-hardening"

$sourceFiles = [ordered]@{
    "deploy-runtime-hardening.sh" = Join-Path $artifactRoot `
        "deploy-runtime-hardening.sh"
    "runtime-context.js" = Join-Path $artifactRoot "runtime-context.js"
    "runtime-context.test.js" = Join-Path $artifactRoot `
        "runtime-context.test.js"
    "artifact-contract.test.js" = Join-Path $artifactRoot `
        "artifact-contract.test.js"
    "settings-persistent-context.js" = Join-Path $artifactRoot `
        "settings-persistent-context.js"
    "README.md" = Join-Path $artifactRoot "README.md"
    "release/sender-release-canary-v2.js" = Join-Path $noderedRoot `
        "sender-release-canary.js"
    "release/README.md" = Join-Path $noderedRoot `
        "SENDER_RELEASE_CANARY.md"
    "payload/services/sender.js" = Join-Path $customNodeRoot "services\sender.js"
    "payload/services/sender.html" = Join-Path $customNodeRoot "services\sender.html"
    "payload/test/sender.test.js" = Join-Path $customNodeRoot `
        "test\sender.test.js"
}

foreach ($entry in $sourceFiles.GetEnumerator()) {
    if (-not (Test-Path -LiteralPath $entry.Value -PathType Leaf)) {
        throw "Falta archivo fuente: $($entry.Value)"
    }
}

foreach ($target in @($outputPath, $checksumPath)) {
    if ((Test-Path -LiteralPath $target) -and -not $Force) {
        throw "Ya existe $target. Use -Force para reemplazar ese archivo exacto."
    }
}

$outputParent = Split-Path -Parent $outputPath
if (-not (Test-Path -LiteralPath $outputParent -PathType Container)) {
    New-Item -ItemType Directory -Path $outputParent | Out-Null
}

try {
    $utf8NoBom = [Text.UTF8Encoding]::new($false)
    New-Item -ItemType Directory -Path $bundleRoot | Out-Null

    foreach ($entry in $sourceFiles.GetEnumerator()) {
        $destination = Join-Path $bundleRoot $entry.Key
        $destinationParent = Split-Path -Parent $destination
        if (-not (Test-Path -LiteralPath $destinationParent -PathType Container)) {
            New-Item -ItemType Directory -Path $destinationParent | Out-Null
        }
        Copy-Item -LiteralPath $entry.Value -Destination $destination
    }

    foreach ($textFile in Get-ChildItem -LiteralPath $bundleRoot -File -Recurse) {
        $content = [IO.File]::ReadAllText($textFile.FullName)
        $content = $content.Replace("`r`n", "`n").Replace("`r", "`n")
        [IO.File]::WriteAllText($textFile.FullName, $content, $utf8NoBom)
    }

    $bundlePrefix = $bundleRoot.TrimEnd("\") + "\"
    $manifestLines = Get-ChildItem -LiteralPath $bundleRoot -File -Recurse |
        Sort-Object FullName |
        ForEach-Object {
            $relative = $_.FullName.Substring($bundlePrefix.Length).
                Replace("\", "/")
            $hash = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).
                Hash.ToLowerInvariant()
            "$hash  $relative"
        }
    [IO.File]::WriteAllText(
        (Join-Path $bundleRoot "SHA256SUMS"),
        (($manifestLines -join "`n") + "`n"),
        $utf8NoBom
    )

    if ($Force) {
        foreach ($target in @($outputPath, $checksumPath)) {
            if (Test-Path -LiteralPath $target) {
                Remove-Item -LiteralPath $target -Force
            }
        }
    }

    & tar.exe -czf $outputPath -C $tempRoot "nodered-production-hardening"
    if ($LASTEXITCODE -ne 0) {
        throw "tar.exe terminó con código $LASTEXITCODE"
    }

    $archiveHash = (Get-FileHash -LiteralPath $outputPath -Algorithm SHA256).
        Hash.ToLowerInvariant()
    $checksumLine = "$archiveHash  $([IO.Path]::GetFileName($outputPath))`n"
    [IO.File]::WriteAllText($checksumPath, $checksumLine, $utf8NoBom)

    [PSCustomObject]@{
        Archive = $outputPath
        SHA256 = $archiveHash
        ChecksumFile = $checksumPath
    }
}
finally {
    $resolvedTempRoot = [IO.Path]::GetFullPath($tempRoot)
    if (
        $resolvedTempRoot.StartsWith(
            $tempBase,
            [StringComparison]::OrdinalIgnoreCase
        ) -and
        ([IO.Path]::GetFileName($resolvedTempRoot) -like `
            "mentor-nodered-hardening-bundle-*") -and
        (Test-Path -LiteralPath $resolvedTempRoot)
    ) {
        Remove-Item -LiteralPath $resolvedTempRoot -Recurse -Force
    }
}
