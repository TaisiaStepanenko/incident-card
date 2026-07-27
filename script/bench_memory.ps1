$ErrorActionPreference = "Stop"

# Пути
$RepoRoot = Split-Path -Parent $PSScriptRoot
$BenchDir = Join-Path $RepoRoot "benchmarks"
$GeneratedEvents = Join-Path $BenchDir "generated_events.jsonl"
$ResultFile = Join-Path $BenchDir "memory.log"
$Binary = Join-Path $RepoRoot "incident-card.exe"

# Создаём папку benchmarks, если нет
if (-not (Test-Path $BenchDir)) {
    New-Item -ItemType Directory -Path $BenchDir -Force | Out-Null
}

# Генерируем события, если файл отсутствует
if (-not (Test-Path $GeneratedEvents)) {
    Write-Host "Generating 1,000,000 events..." 
    & $Binary generate --count 1000000 --scenario external_send --out $GeneratedEvents --seed 42
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Generate failed"
        exit 1
    }
    Write-Host "Events generated."
} else {
    Write-Host "Using existing events file: $GeneratedEvents"
}

# Запускаем build и замеряем пиковую память
Write-Host "Running build and measuring peak memory..."

$arguments = @(
    "build",
    "--events", $GeneratedEvents,
    "--event-id", "evt_12345",
    "--out", (Join-Path $BenchDir "card.md"),
    "--json", (Join-Path $BenchDir "card.json")
)

$process = Start-Process -FilePath $Binary -ArgumentList $arguments -PassThru -NoNewWindow
$procId = $process.Id

# Мониторим память с интервалом 200 мс
$peakWorkingSet = 0
while (-not $process.HasExited) {
    try {
        $proc = Get-Process -Id $procId -ErrorAction SilentlyContinue
        if ($proc) {
            $currentWS = $proc.WorkingSet
            if ($currentWS -gt $peakWorkingSet) {
                $peakWorkingSet = $currentWS
            }
        }
    } catch {
        # Игнорируем ошибки (процесс мог завершиться)
    }
    Start-Sleep -Milliseconds 200
}

# Даём системе время обновить объект процесса
Start-Sleep -Milliseconds 100
$process.Refresh()
$exitCode = $process.ExitCode

# Если exit code не определён, считаем успешным завершением
if ($exitCode -eq $null) {
    $exitCode = 0
}

$peakMemoryMB = [math]::Round($peakWorkingSet / 1MB, 0)

$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$logEntry = @"
Timestamp: $timestamp
Events: 1000000
Peak Working Set: $peakWorkingSet bytes ($peakMemoryMB MB)
Exit Code: $exitCode
"@

if ($exitCode -ne 0) {
    $logEntry += "`nERROR: Build failed with exit code $exitCode"
    Write-Host "Build failed with exit code $exitCode" -ForegroundColor Red
} else {
    Write-Host "Build succeeded." -ForegroundColor Green
}

$logEntry | Out-File -FilePath $ResultFile -Encoding utf8
Write-Host "Memory benchmark result saved to $ResultFile"
Write-Host $logEntry