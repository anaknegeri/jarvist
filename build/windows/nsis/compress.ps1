<#
.SYNOPSIS
Compress folder using 7-Zip with maximum compression
#>
param(
  [Parameter(Mandatory = $true, Position = 0)]
  [string]$SourceFolder,

  [Parameter(Mandatory = $false)]
  [string]$OutputFile = "bin.7z",

  [Parameter(Mandatory = $false)]
  [int]$CompressionLevel = 9
)

# Validate source folder
if (-not (Test-Path $SourceFolder -PathType Container)) {
  Write-Error "Source folder does not exist: $SourceFolder"
  exit 1
}

Write-Host "Remove excluded directories and files"

$ExcludePaths = @(
    (Join-Path $SourceFolder "__pycache__"),
    (Join-Path $SourceFolder ".git"),
    (Join-Path $SourceFolder ".vscode"),
    (Join-Path $SourceFolder ".idea")
)

$ExcludeFiles = @(
  "*.log",
  "*.sqlite3",
  "*.db",
  "*.pid",
  "*.bak",
  "*.tmp"
)

# Remove directories
foreach ($path in $ExcludePaths) {
  if (Test-Path $path) {
    try {
      Remove-Item -Path $path -Recurse -Force
      Write-Host "Removed directory: $path"
    }
    catch {
      Write-Host "Could not remove directory: $path"
    }
  }
}

# Remove files
foreach ($pattern in $ExcludeFiles) {
  $filesToRemove = Get-ChildItem -Path $SourceFolder -Recurse -Filter $pattern
  foreach ($file in $filesToRemove) {
    try {
      Remove-Item -Path $file.FullName -Force
      Write-Host "Removed file: $($file.FullName)"
    }
    catch {
      Write-Host "Could not remove file: $($file.FullName)"
    }
  }
}

# Ensure source folder path ends with a backslash for consistent behavior
$SourceFolder = $SourceFolder.TrimEnd('\') + '\'

# Compression command
$compressCommand = @"
./7za a -t7z -mx=$CompressionLevel -mfb=273 -ms=on -md=32m -myx=9 -mtc=on -mta=on "$OutputFile" "$SourceFolder"
"@

Write-Host "Compressing '$SourceFolder' to '$OutputFile'..."
Write-Host "Compression Level: $CompressionLevel"

# Execute compression
Invoke-Expression $compressCommand

# Check compression result
if ($LASTEXITCODE -eq 0) {
  $originalSize = (Get-ChildItem $SourceFolder -Recurse | Measure-Object -Property Length -Sum).Sum
  $compressedSize = (Get-Item $OutputFile | Measure-Object -Property Length -Sum).Sum

  $originalSizeFormatted = if ($originalSize -ge 1GB) {
    "{0:N2} GB" -f ($originalSize / 1GB)
  }
  elseif ($originalSize -ge 1MB) {
    "{0:N2} MB" -f ($originalSize / 1MB)
  }
  else {
    "{0:N2} KB" -f ($originalSize / 1KB)
  }

  $compressedSizeFormatted = if ($compressedSize -ge 1GB) {
    "{0:N2} GB" -f ($compressedSize / 1GB)
  }
  elseif ($compressedSize -ge 1MB) {
    "{0:N2} MB" -f ($compressedSize / 1MB)
  }
  else {
    "{0:N2} KB" -f ($compressedSize / 1KB)
  }

  $compressionRatio = [math]::Round(($compressedSize / $originalSize) * 100, 2)

  Write-Host "Compression complete!"
  Write-Host "Original Size: $originalSizeFormatted"
  Write-Host "Compressed Size: $compressedSizeFormatted"
  Write-Host "Compression Ratio: $compressionRatio%"
}
else {
  Write-Host "Compression failed!"
  exit 1
}
