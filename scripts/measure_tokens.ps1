# Token-efficiency measurement script (Section 20)
$ErrorActionPreference = "SilentlyContinue"

Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "          Token Efficiency & Footprint Measurement              " -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan

$rawStatus = git status 2>&1 | Out-String
$rawBranch = git branch -vv 2>&1 | Out-String
$rawCombined = "$rawStatus`n$rawBranch"
$rawChars = $rawCombined.Length
$rawTokens = [math]::Round($rawChars / 4)

$gwOutput = if (Test-Path ".\gw.exe") { .\gw.exe inspect --json | Out-String } else { "{}" }
$gwChars = $gwOutput.Length
$gwTokens = [math]::Round($gwChars / 4)

Write-Host "Raw multi-command shell output:"
Write-Host "  - Characters: $rawChars"
Write-Host "  - Est Tokens: ~$rawTokens"
Write-Host ""
Write-Host "Deterministic 'gw inspect --json' envelope:"
Write-Host "  - Characters: $gwChars"
Write-Host "  - Est Tokens: ~$gwTokens"
Write-Host "================================================================" -ForegroundColor Cyan