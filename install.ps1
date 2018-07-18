# This script will download and install Dividat Driver as a Windows service.

## Configuration ##########################################
$releaseUrl = "https://dist.dividat.com/releases/driver2/"
$channel = "master"
$installDir = "C:\dividat-driver"
###########################################################

$ErrorActionPreference = "Stop"

# Figure out the latest version
$latestTmpFile = Join-Path $env:TEMP "dividat-driver-latest.txt"
(New-Object System.Net.WebClient).DownloadFile($releaseUrl + $channel + "/latest",$latestTmpFile)
$latest = (Get-Content $latestTmpFile -Raw).trim()
Remove-Item -path $latestTmpFile

# Create install directory
if (![System.IO.Directory]::Exists($installDir)) {[void][System.IO.Directory]::CreateDirectory($installDir)}

# Download application
$downloadUrl = $releaseUrl + $channel + "/" + $latest + "/" + "dividat-driver-windows-amd64-" + $latest + ".exe"
$appPath = Join-Path $installDir "dividat-driver.exe"
(New-Object System.Net.WebClient).DownloadFile($downloadUrl,$appPath)

# Install as service
New-Service -Name "DividatDriver" -BinaryPathName $appPath -DisplayName "Dividat Driver" -StartupType Automatic

# Start the service
Start-Service DividatDriver
