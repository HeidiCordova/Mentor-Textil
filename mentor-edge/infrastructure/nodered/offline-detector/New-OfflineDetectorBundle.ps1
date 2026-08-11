[CmdletBinding()]
param(
    [string]$OutputArchive = ".\detector-offline-v5.tgz",
    [switch]$Force
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$artifactRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$serviceRoot = [IO.Path]::GetFullPath(
    (Join-Path $artifactRoot "..\..\..\services\vision-event-detector")
)
$databaseRoot = [IO.Path]::GetFullPath(
    (Join-Path $artifactRoot "..\..\database")
)
$outputPath = [IO.Path]::GetFullPath($OutputArchive)
$checksumPath = "$outputPath.sha256"
$tempBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$tempRoot = Join-Path $tempBase ("mentor-detector-bundle-" + [Guid]::NewGuid().ToString("N"))
$bundleRoot = Join-Path $tempRoot "detector-offline-v5"

if (-not (Test-Path -LiteralPath (Join-Path $serviceRoot "app") -PathType Container)) {
    throw "No se encontró el código del detector: $serviceRoot"
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
    New-Item -ItemType Directory -Path (Join-Path $bundleRoot "tests") | Out-Null
    New-Item -ItemType Directory -Path (Join-Path $bundleRoot "wheels") | Out-Null

    $appSource = Join-Path $serviceRoot "app"
    $appSourcePrefix = $appSource.TrimEnd("\") + "\"
    foreach ($sourceFile in Get-ChildItem -LiteralPath $appSource -File -Recurse -Filter "*.py") {
        $relative = $sourceFile.FullName.Substring($appSourcePrefix.Length)
        $destination = Join-Path (Join-Path $bundleRoot "app") $relative
        $destinationParent = Split-Path -Parent $destination
        if (-not (Test-Path -LiteralPath $destinationParent -PathType Container)) {
            New-Item -ItemType Directory -Path $destinationParent | Out-Null
        }
        Copy-Item -LiteralPath $sourceFile.FullName -Destination $destination
    }

    foreach ($testName in @(
        "test_cv_image_processor.py",
        "test_event_fsm.py",
        "test_durable_event_output.py",
        "test_calibration_persistence.py"
    )) {
        Copy-Item -LiteralPath (Join-Path $serviceRoot "tests\$testName") `
            -Destination (Join-Path $bundleRoot "tests\$testName")
    }

    Copy-Item -LiteralPath (
        Join-Path $artifactRoot `
            "wheels\psycopg2_binary-2.9.9-cp310-cp310-manylinux_2_17_aarch64.manylinux2014_aarch64.whl"
    ) -Destination (Join-Path $bundleRoot "wheels")

    Copy-Item -LiteralPath (Join-Path $artifactRoot "Dockerfile") `
        -Destination (Join-Path $bundleRoot "Dockerfile")
    Copy-Item -LiteralPath (Join-Path $artifactRoot "compose-detector-v5.patch") `
        -Destination (Join-Path $bundleRoot "compose-detector-v5.patch")
    Copy-Item -LiteralPath (Join-Path $databaseRoot "30_detector_calibration.sql") `
        -Destination (Join-Path $bundleRoot "30_detector_calibration.sql")
    Copy-Item -LiteralPath (Join-Path $databaseRoot "33_vision_detections_existing_lines.sql") `
        -Destination (Join-Path $bundleRoot "33_vision_detections_existing_lines.sql")
    Copy-Item -LiteralPath (Join-Path $artifactRoot "detector-image.sh") `
        -Destination (Join-Path $bundleRoot "detector-image.sh")
    Copy-Item -LiteralPath (Join-Path $artifactRoot "README.md") `
        -Destination (Join-Path $bundleRoot "README.md")

    # The bundle is consumed by Linux. Normalize every text artifact to LF so
    # shebangs, sha256sum manifests, patches and SQL never retain Windows CRLF.
    $linuxTextNames = @("Dockerfile", "README.md")
    $linuxTextExtensions = @(".py", ".sh", ".sql", ".patch")
    foreach ($textFile in Get-ChildItem -LiteralPath $bundleRoot -File -Recurse) {
        if (
            $linuxTextNames -contains $textFile.Name -or
            $linuxTextExtensions -contains $textFile.Extension.ToLowerInvariant()
        ) {
            $content = [IO.File]::ReadAllText($textFile.FullName)
            $content = $content.Replace("`r`n", "`n").Replace("`r", "`n")
            [IO.File]::WriteAllText($textFile.FullName, $content, $utf8NoBom)
        }
    }

    $bundlePrefix = $bundleRoot.TrimEnd("\") + "\"
    $manifestLines = Get-ChildItem -LiteralPath $bundleRoot -File -Recurse |
        Sort-Object FullName |
        ForEach-Object {
            $relative = $_.FullName.Substring($bundlePrefix.Length).Replace("\", "/")
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

    & tar.exe -czf $outputPath -C $tempRoot "detector-offline-v5"
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
        ([IO.Path]::GetFileName($resolvedTempRoot) -like "mentor-detector-bundle-*") -and
        (Test-Path -LiteralPath $resolvedTempRoot)
    ) {
        Remove-Item -LiteralPath $resolvedTempRoot -Recurse -Force
    }
}
