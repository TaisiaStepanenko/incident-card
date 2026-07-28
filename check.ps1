param(
    [string]$OutRoot = '',
    [switch]$AllowDirty
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'
if (Get-Variable -Name PSNativeCommandUseErrorActionPreference -ErrorAction SilentlyContinue) {
    $PSNativeCommandUseErrorActionPreference = $false
}

function Get-CheckGoCommand {
    $preferred = 'K:\go\go1.20.14\bin\go.exe'
    if (Test-Path -LiteralPath $preferred) { return $preferred }
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) { return $go.Source }
    throw 'go executable was not found in PATH and K:\go\go1.20.14\bin\go.exe is missing.'
}

function New-CheckContext {
    param([string]$Student, [string]$OutRoot)
    $repo = (Get-Location).Path
    if ($OutRoot -eq '') { $OutRoot = Join-Path $repo '.check-results' }
    $stamp = Get-Date -Format 'yyyyMMdd_HHmmss'
    $safe = $Student -replace '[^A-Za-z0-9_.-]', '_'
    $resultDir = Join-Path $OutRoot "${safe}_${stamp}"
    $ctx = [ordered]@{
        Student = $Student
        RepoRoot = $repo
        ResultDir = $resultDir
        LogsDir = Join-Path $resultDir 'logs'
        InputsDir = Join-Path $resultDir 'inputs'
        OutputsDir = Join-Path $resultDir 'outputs'
        MetaDir = Join-Path $resultDir 'meta'
        TmpDir = Join-Path $resultDir 'tmp'
        CommandsPath = Join-Path $resultDir 'commands.jsonl'
        GoCmd = Get-CheckGoCommand
        StartedAt = (Get-Date).ToString('o')
        CommandResults = @{}
        Assessments = New-Object System.Collections.ArrayList
    }
    foreach ($dir in @($ctx.ResultDir, $ctx.LogsDir, $ctx.InputsDir, $ctx.OutputsDir, $ctx.MetaDir, $ctx.TmpDir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
    }
    Write-CheckFileText -Path $ctx.CommandsPath -Content ''
    return $ctx
}

# Кодировки задаём явно: `Set-Content -Encoding UTF8` добавляет BOM в Windows PowerShell 5.1
# и не добавляет в PowerShell 7, а `Get-Content` без -Encoding читает файл без BOM как ANSI.
# Из-за этого одни и те же артефакты решения давали разный результат проверки на разных хостах.
$script:Utf8NoBom = New-Object System.Text.UTF8Encoding($false)
$script:Utf8Bom = New-Object System.Text.UTF8Encoding($true)

function Write-CheckFileText {
    param([string]$Path, [string]$Content, [switch]$WithBom)
    $encoding = if ($WithBom) { $script:Utf8Bom } else { $script:Utf8NoBom }
    [System.IO.File]::WriteAllText($Path, $Content, $encoding)
}

# Читает UTF-8 и снимает BOM, если он есть.
function Get-CheckFileText {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) { return '' }
    return [System.IO.File]::ReadAllText($Path, [System.Text.Encoding]::UTF8)
}

function Save-CheckJson {
    param([string]$Path, $Value)
    Write-CheckFileText -Path $Path -Content (($Value | ConvertTo-Json -Depth 50) + [Environment]::NewLine)
}

function Write-CheckText {
    param($Ctx, [string]$RelativePath, [string]$Content)
    $path = Join-Path $Ctx.ResultDir $RelativePath
    $parent = Split-Path -Parent $path
    if ($parent) { New-Item -ItemType Directory -Force -Path $parent | Out-Null }
    # Входные данные для решения — без BOM: иначе байты входа зависят от хоста проверки.
    Write-CheckFileText -Path $path -Content $Content
    return $path
}

function To-Rel {
    param($Ctx, [string]$Path)
    return $Path.Replace($Ctx.ResultDir, '').TrimStart('\')
}

function Get-ProcessTreePeakWorkingSet {
    param([int]$RootProcessId)
    $queue = New-Object System.Collections.Queue
    $queue.Enqueue($RootProcessId)
    $seen = @{}
    [long]$total = 0

    while ($queue.Count -gt 0) {
        $currentId = [int]$queue.Dequeue()
        $key = [string]$currentId
        if ($seen.ContainsKey($key)) { continue }
        $seen[$key] = $true

        try {
            $item = Get-Process -Id $currentId -ErrorAction Stop
            $workingSet = [Math]::Max([long]$item.WorkingSet64, [long]$item.PeakWorkingSet64)
            $total += $workingSet
        } catch {}

        try {
            $children = @(Get-CimInstance -ClassName Win32_Process -Filter "ParentProcessId = $currentId" -ErrorAction Stop)
            foreach ($child in $children) {
                $queue.Enqueue([int]$child.ProcessId)
            }
        } catch {}
    }

    return $total
}

# Приоритет у кода, записанного самим раннером; хостовое значение — только резервный источник.
# Если код не получен ни оттуда, ни отсюда, возвращаем 1, а не 0: молчаливый успех недопустим.
function Resolve-CheckExitCode {
    param([string]$ExitCodePath, [bool]$TimedOut, $Fallback)
    if ($TimedOut) { return 124 }
    if (Test-Path -LiteralPath $ExitCodePath) {
        $raw = (Get-CheckFileText -Path $ExitCodePath).Trim()
        $parsed = 0
        if ([int]::TryParse($raw, [ref]$parsed)) { return $parsed }
    }
    if ($null -ne $Fallback -and $Fallback -ne '') { return [int]$Fallback }
    return 1
}

function Invoke-CheckCommand {
    param($Ctx, [string]$Name, [string]$Command, [string]$WorkingDirectory = '', [int]$TimeoutSec = 0)
    if ($WorkingDirectory -eq '') { $WorkingDirectory = $Ctx.RepoRoot }
    $safe = $Name -replace '[^A-Za-z0-9_.-]', '_'
    $runnerPath = Join-Path $Ctx.TmpDir "$safe.ps1"
    $stdoutPath = Join-Path $Ctx.TmpDir "$safe.stdout.log"
    $stderrPath = Join-Path $Ctx.TmpDir "$safe.stderr.log"
    # Код возврата раннер записывает сам: под Windows PowerShell 5.1 `Start-Process -PassThru`
    # возвращает объект с пустым ExitCode, а `[int]$null` даёт 0, из-за чего любая упавшая
    # команда с таймаутом попадала в отчёт как успешная.
    $exitCodePath = Join-Path $Ctx.TmpDir "$safe.exitcode"
    if (Test-Path -LiteralPath $exitCodePath) { Remove-Item -LiteralPath $exitCodePath -Force }
    $logPath = Join-Path $Ctx.LogsDir "$safe.log"
    $started = Get-Date

    $runner = @(
        '$ErrorActionPreference = ''Stop'''
        "Set-Location -LiteralPath '$($WorkingDirectory.Replace("'", "''"))'"
        'try {'
        '    $global:LASTEXITCODE = $null'
        '    $Error.Clear()'
        "    $Command"
        '    $success = $?'
        '    $exitCode = $global:LASTEXITCODE'
        '    if ($null -eq $exitCode) {'
        '        if ($success -and $Error.Count -eq 0) { $exitCode = 0 } else { $exitCode = 1 }'
        '    }'
        "    [System.IO.File]::WriteAllText('$($exitCodePath.Replace("'", "''"))', [string][int]`$exitCode)"
        '    exit $exitCode'
        '} catch {'
        '    Write-Error $_'
        "    [System.IO.File]::WriteAllText('$($exitCodePath.Replace("'", "''"))', '1')"
        '    exit 1'
        '}'
    ) -join "`r`n"
    # Раннер исполняет powershell.exe (5.1), которому нужен BOM, иначе пути с кириллицей ломаются.
    Write-CheckFileText -Path $runnerPath -Content $runner -WithBom
    [long]$peak = 0
    $timedOut = $false
    $stdout = ''
    $stderr = ''
    $exitCode = 1
    if ($TimeoutSec -le 0) {
        $prevErrorAction = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        $output = & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $runnerPath 2>&1
        $ErrorActionPreference = $prevErrorAction
        $exitCode = Resolve-CheckExitCode -ExitCodePath $exitCodePath -TimedOut $false -Fallback $LASTEXITCODE
        $stderr = ($output | Where-Object { $_ -is [System.Management.Automation.ErrorRecord] } | Out-String)
        $stdout = ($output | Where-Object { $_ -isnot [System.Management.Automation.ErrorRecord] } | Out-String)
    } else {
        if (Test-Path -LiteralPath $stdoutPath) { Remove-Item -LiteralPath $stdoutPath -Force }
        if (Test-Path -LiteralPath $stderrPath) { Remove-Item -LiteralPath $stderrPath -Force }
        $args = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $runnerPath)
        $proc = Start-Process -FilePath 'powershell.exe' -ArgumentList $args -PassThru -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath
        while (-not $proc.HasExited) {
            $treePeak = Get-ProcessTreePeakWorkingSet -RootProcessId $proc.Id
            if ($treePeak -gt $peak) { $peak = $treePeak }
            if (((Get-Date) - $started).TotalSeconds -gt $TimeoutSec) {
                $timedOut = $true
                try {
                    & taskkill.exe /PID ([string]$proc.Id) /T /F 2>&1 | Out-Null
                } catch {
                    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
                }
                break
            }
            Start-Sleep -Milliseconds 200
        }
        $proc.WaitForExit()
        $proc.Refresh()
        $exitCode = Resolve-CheckExitCode -ExitCodePath $exitCodePath -TimedOut $timedOut -Fallback $proc.ExitCode
        $stdout = if (Test-Path -LiteralPath $stdoutPath) { Get-Content -LiteralPath $stdoutPath -Raw } else { '' }
        $stderr = if (Test-Path -LiteralPath $stderrPath) { Get-Content -LiteralPath $stderrPath -Raw } else { '' }
    }
    $ended = Get-Date

    $logText = @(
        "name: $Name"
        "working_directory: $WorkingDirectory"
        "command:"
        $Command
        "timeout_sec: $TimeoutSec"
        "timed_out: $timedOut"
        "peak_working_set_bytes: $peak"
        "exit_code: $exitCode"
        "started_at: $($started.ToString('o'))"
        "ended_at: $($ended.ToString('o'))"
        ''
        'stdout:'
        $stdout
        ''
        'stderr:'
        $stderr
    ) -join [Environment]::NewLine
    Write-CheckFileText -Path $logPath -Content $logText

    $record = [ordered]@{
        name = $Name
        command = $Command
        working_directory = $WorkingDirectory
        exit_code = $exitCode
        timed_out = $timedOut
        peak_working_set_bytes = $peak
        started_at = $started.ToString('o')
        ended_at = $ended.ToString('o')
        duration_ms = [int](($ended - $started).TotalMilliseconds)
        log = "logs/$safe.log"
    }
    ($record | ConvertTo-Json -Compress) | Add-Content -LiteralPath $Ctx.CommandsPath -Encoding UTF8
    $Ctx.CommandResults[$Name] = $record
}

function Add-FeatureAssessment {
    param($Ctx, [string]$Id, [string]$Level, [string]$Category, [string]$Requirement, [string]$Implementation, [string]$Conformance, [string[]]$Evidence = @(), [string]$Details = '')
    $Ctx.Assessments.Add([ordered]@{
        id = $Id
        level = $Level
        category = $Category
        requirement = $Requirement
        implementation = $Implementation
        conformance = $Conformance
        evidence = @($Evidence)
        details = $Details
    }) | Out-Null
}

function Add-CommandFeatureAssessment {
    param($Ctx, [string]$Id, [string]$Level, [string]$Category, [string]$Requirement, [string]$CommandName, [string[]]$RequiredArtifacts = @(), [bool]$ExtraConformant, [string[]]$ExtraEvidence = @(), [string]$Details = '')
    $has = $Ctx.CommandResults.ContainsKey($CommandName)
    $code = if ($has) { [int]$Ctx.CommandResults[$CommandName].exit_code } else { -999 }
    $missing = @($RequiredArtifacts | Where-Object { -not (Test-Path -LiteralPath $_) })
    $ok = $has -and $code -eq 0 -and $missing.Count -eq 0 -and $ExtraConformant
    $implementation = if (-not $has) { 'not_implemented' } elseif ($ok) { 'full' } else { 'partial' }
    $conformance = if ($ok) { 'conformant' } else { 'nonconformant' }
    $evidence = @()
    if ($has) { $evidence += [string]$Ctx.CommandResults[$CommandName].log }
    foreach ($artifact in $RequiredArtifacts) { if (Test-Path -LiteralPath $artifact) { $evidence += To-Rel -Ctx $Ctx -Path $artifact } }
    $evidence += $ExtraEvidence
    Add-FeatureAssessment -Ctx $Ctx -Id $Id -Level $Level -Category $Category -Requirement $Requirement -Implementation $implementation -Conformance $conformance -Evidence ($evidence | Select-Object -Unique) -Details ("exit_code=$code; $Details")
}

function Add-BooleanFeatureAssessment {
    param($Ctx, [string]$Id, [string]$Level, [string]$Category, [string]$Requirement, [bool]$Ok, [string[]]$Evidence = @(), [string]$Details = '')
    $implementation = 'not_implemented'
    $conformance = 'nonconformant'
    if ($Ok) {
        $implementation = 'full'
        $conformance = 'conformant'
    }
    Add-FeatureAssessment -Ctx $Ctx -Id $Id -Level $Level -Category $Category -Requirement $Requirement -Implementation $implementation -Conformance $conformance -Evidence $Evidence -Details $Details
}

function Read-CheckJson {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) { return $null }
    $raw = Get-CheckFileText -Path $Path
    if ([string]::IsNullOrWhiteSpace($raw)) { return $null }
    return $raw | ConvertFrom-Json
}

# Поля выходного JSON опциональны (omitempty), а Set-StrictMode 2.0 запрещает обращение
# к несуществующим свойствам, поэтому читаем их только через эти обёртки.
function Get-JsonProperty {
    param($Object, [string]$Name)
    if ($null -eq $Object) { return $null }
    if ($null -eq $Object.PSObject) { return $null }
    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property) { return $null }
    return $property.Value
}

function Get-JsonArray {
    param($Object, [string]$Name)
    $value = Get-JsonProperty -Object $Object -Name $Name
    if ($null -eq $value) { return @() }
    return @($value)
}

function Test-ArrayExact {
    param([object[]]$Actual, [object[]]$Expected)
    if ($Actual.Count -ne $Expected.Count) { return $false }
    for ($i = 0; $i -lt $Actual.Count; $i++) { if ([string]$Actual[$i] -ne [string]$Expected[$i]) { return $false } }
    return $true
}

# Задание не фиксирует ни включение главного события в разделы same_*, ни включение
# границы временного окна, поэтому корректной считается любая из допустимых последовательностей.
function Test-ArrayIsOneOf {
    param([object[]]$Actual, [object[]]$Allowed)
    foreach ($variant in $Allowed) {
        if (Test-ArrayExact -Actual $Actual -Expected @($variant)) { return $true }
    }
    return $false
}

# Порядок подозрительных факторов заданием не определён, сравниваем как множества.
function Test-SetEquals {
    param([object[]]$Actual, [object[]]$Expected)
    $actualSet = @($Actual | ForEach-Object { [string]$_ } | Sort-Object -Unique)
    $expectedSet = @($Expected | ForEach-Object { [string]$_ } | Sort-Object -Unique)
    return (Test-ArrayExact -Actual $actualSet -Expected $expectedSet)
}

function Test-Rfc3339 {
    param([string]$Value)
    if ([string]::IsNullOrWhiteSpace($Value)) { return $false }
    [ref]$parsed = [datetimeoffset]::MinValue
    return [datetimeoffset]::TryParse(
        $Value,
        [System.Globalization.CultureInfo]::InvariantCulture,
        [System.Globalization.DateTimeStyles]::RoundtripKind,
        $parsed)
}

function Get-EventIdAtLine {
    param([string]$Path, [int]$LineNumber)
    if (-not (Test-Path -LiteralPath $Path)) { return '' }
    $reader = [System.IO.File]::OpenText($Path)
    try {
        $number = 0
        while ($true) {
            $line = $reader.ReadLine()
            if ($null -eq $line) { return '' }
            if ([string]::IsNullOrWhiteSpace($line)) { continue }
            $number++
            if ($number -eq $LineNumber) {
                if ($line -match '"event_id"\s*:\s*"([^"]+)"') { return $Matches[1] }
                return ''
            }
        }
    } finally {
        $reader.Close()
    }
}

function Get-RepoTestFiles {
    param($Ctx)
    return @(Get-ChildItem -LiteralPath $Ctx.RepoRoot -Recurse -File -Filter '*_test.go' -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -notlike '*\.check-results\*' })
}

# Возвращает имена тестов, завершившихся статусом pass, из вывода go test -json.
function Get-PassedTestNames {
    param([string]$JsonLines)
    $names = New-Object 'System.Collections.Generic.HashSet[string]'
    foreach ($line in ($JsonLines -split "`r?`n")) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        if ($line -notmatch '"Action"\s*:\s*"pass"') { continue }
        if ($line -match '"Test"\s*:\s*"([^"]+)"') { [void]$names.Add($Matches[1]) }
    }
    return $names
}

function Test-TimelineSortedDedup {
    param([object[]]$Timeline)
    $seen = @{}
    $lastTs = ''
    $lastID = ''
    foreach ($item in $Timeline) {
        $id = [string](Get-JsonProperty -Object $item -Name 'event_id')
        $ts = [string](Get-JsonProperty -Object $item -Name 'timestamp')
        if ($seen.ContainsKey($id)) { return $false }
        $seen[$id] = $true
        if ($lastTs -gt $ts) { return $false }
        if ($lastTs -eq $ts -and $lastID -gt $id) { return $false }
        $lastTs = $ts
        $lastID = $id
    }
    return $true
}

function Get-CommandLogText {
    param($Ctx, [string]$Name)
    if (-not $Ctx.CommandResults.ContainsKey($Name)) { return '' }
    $path = Join-Path $Ctx.ResultDir ([string]$Ctx.CommandResults[$Name].log)
    return Get-CheckFileText -Path $path
}

function Get-FileStatsUniqueIDs {
    param([string]$Path)
    $set = New-Object 'System.Collections.Generic.HashSet[string]'
    $count = 0
    $dupes = 0
    $reader = [System.IO.File]::OpenText($Path)
    try {
        while ($true) {
            $line = $reader.ReadLine()
            if ($null -eq $line) { break }
            if ([string]::IsNullOrWhiteSpace($line)) { continue }
            $count++
            if ($line -notmatch '"event_id"\s*:\s*"([^"]+)"') { throw "line $count does not contain event_id" }
            $id = $Matches[1]
            if (-not $set.Add($id)) { $dupes++ }
        }
    } finally {
        $reader.Close()
    }
    return [ordered]@{ lines = $count; unique_ids = $set.Count; duplicates = $dupes }
}

function Add-StandardEngineeringAssessments {
    param($Ctx)
    $testFiles = Get-RepoTestFiles -Ctx $Ctx
    $tests = @($testFiles | Select-String -Pattern '^\s*func\s+Test[A-Za-z0-9_]+\s*\(' -ErrorAction SilentlyContinue)
    $benches = @($testFiles | Select-String -Pattern '^\s*func\s+Benchmark[A-Za-z0-9_]+\s*\(' -ErrorAction SilentlyContinue)
    # Свидетельства берём по факту: файлы с тестами ищем в репозитории, а не подставляем ожидаемый путь.
    $testEvidence = @($tests | ForEach-Object { $_.Path.Replace($Ctx.RepoRoot, '').TrimStart('\').Replace('\', '/') } | Select-Object -Unique)
    $benchEvidence = @($benches | ForEach-Object { $_.Path.Replace($Ctx.RepoRoot, '').TrimStart('\').Replace('\', '/') } | Select-Object -Unique)
    if ($testEvidence.Count -eq 0) { $testEvidence = @('нет файлов *_test.go с функциями Test') }
    if ($benchEvidence.Count -eq 0) { $benchEvidence = @('нет файлов *_test.go с функциями Benchmark') }
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.unit_tests_present' -Level 'engineering' -Category 'tests' -Requirement 'Go unit tests are present' -Ok ($tests.Count -gt 0) -Evidence $testEvidence
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.benchmarks_present' -Level 'engineering' -Category 'benchmarks' -Requirement 'Go benchmark tests are present' -Ok ($benches.Count -gt 0) -Evidence $benchEvidence
    Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.go_test_passes' -Level 'engineering' -Category 'tests' -Requirement 'go test ./... passes' -CommandName 'go_test_all' -ExtraConformant $true
    Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.make_test_runs' -Level 'engineering' -Category 'reproducibility' -Requirement 'make test passes' -CommandName 'make_test' -ExtraConformant $true
    Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.make_bench_runs' -Level 'engineering' -Category 'reproducibility' -Requirement 'make bench passes' -CommandName 'make_bench' -ExtraConformant $true
    Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.make_demo_runs' -Level 'engineering' -Category 'reproducibility' -Requirement 'make demo passes' -CommandName 'make_demo' -ExtraConformant $true
    $readmePath = Join-Path $Ctx.RepoRoot 'README.md'
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.readme' -Level 'engineering' -Category 'documentation' -Requirement 'README.md exists and is not empty' -Ok ((Test-Path -LiteralPath $readmePath) -and ((Get-Item -LiteralPath $readmePath).Length -gt 100)) -Evidence @('repo_snapshot/README.md')
    $makefilePath = Join-Path $Ctx.RepoRoot 'Makefile'
    $makefileText = if (Test-Path -LiteralPath $makefilePath) { Get-Content -LiteralPath $makefilePath -Raw } else { '' }
    foreach ($target in @('test','bench','demo')) {
        Add-BooleanFeatureAssessment -Ctx $Ctx -Id "engineering.make_$target" -Level 'engineering' -Category 'reproducibility' -Requirement "Makefile has target $target" -Ok ($makefileText -match "(?m)^\s*${target}\s*:") -Evidence @('repo_snapshot/Makefile')
    }
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.control_data' -Level 'engineering' -Category 'reproducibility' -Requirement 'Fixed testdata/control set exists' -Ok (Test-Path -LiteralPath (Join-Path $Ctx.RepoRoot 'testdata\control')) -Evidence @('repo_snapshot/testdata/control')
    $solutionPath = Join-Path $Ctx.RepoRoot 'docs\reshenie.md'
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.solution_doc' -Level 'engineering' -Category 'documentation' -Requirement 'Non-empty docs/reshenie.md exists' -Ok ((Test-Path -LiteralPath $solutionPath) -and ((Get-Item -LiteralPath $solutionPath).Length -gt 100)) -Evidence @('repo_snapshot/docs/reshenie.md')
}

function Copy-CheckPath {
    param($Ctx, [string]$Source, [string]$RelativeDestination)
    if (-not (Test-Path -LiteralPath $Source)) { return }
    $dst = Join-Path $Ctx.ResultDir $RelativeDestination
    $parent = Split-Path -Parent $dst
    if ($parent) { New-Item -ItemType Directory -Force -Path $parent | Out-Null }
    Copy-Item -LiteralPath $Source -Destination $dst -Recurse -Force
}

function Complete-Check {
    param($Ctx, [hashtable]$Extra = @{})
    Add-StandardEngineeringAssessments -Ctx $Ctx
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_git_head' -Command "git rev-parse HEAD | Set-Content -LiteralPath '$($Ctx.MetaDir)\git_head.txt' -Encoding UTF8"
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_git_status' -Command "git status --short | Set-Content -LiteralPath '$($Ctx.MetaDir)\git_status_short.txt' -Encoding UTF8"
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_go_version' -Command "& '$($Ctx.GoCmd)' version | Set-Content -LiteralPath '$($Ctx.MetaDir)\go_version.txt' -Encoding UTF8"
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_go_env' -Command "& '$($Ctx.GoCmd)' env GOVERSION GOOS GOARCH | Set-Content -LiteralPath '$($Ctx.MetaDir)\go_env.txt' -Encoding UTF8"
    foreach ($name in @('README.md', 'Makefile', 'go.mod', 'docs', 'testdata')) {
        Copy-CheckPath -Ctx $Ctx -Source (Join-Path $Ctx.RepoRoot $name) -RelativeDestination "repo_snapshot/$name"
    }
    $items = @($Ctx.Assessments)
    $summary = [ordered]@{}
    foreach ($level in @('minimum','good','excellent','engineering')) {
        $lvl = @($items | Where-Object { $_.level -eq $level })
        $summary[$level] = [ordered]@{
            total = $lvl.Count
            full = @($lvl | Where-Object { $_.implementation -eq 'full' }).Count
            partial = @($lvl | Where-Object { $_.implementation -eq 'partial' }).Count
            not_implemented = @($lvl | Where-Object { $_.implementation -eq 'not_implemented' }).Count
            conformant = @($lvl | Where-Object { $_.conformance -eq 'conformant' }).Count
            nonconformant = @($lvl | Where-Object { $_.conformance -eq 'nonconformant' }).Count
            not_tested = @($lvl | Where-Object { $_.conformance -eq 'not_tested' }).Count
        }
    }
    Save-CheckJson -Path (Join-Path $Ctx.ResultDir 'assessment.json') -Value ([ordered]@{
        schema_version = 1
        statuses = [ordered]@{
            implementation = @('not_implemented', 'partial', 'full')
            conformance = @('not_tested', 'nonconformant', 'conformant')
        }
        summary = $summary
        features = $items
    })
    Save-CheckJson -Path (Join-Path $Ctx.ResultDir 'manifest.json') -Value ([ordered]@{
        student = $Ctx.Student
        repo_root = $Ctx.RepoRoot
        started_at = $Ctx.StartedAt
        completed_at = (Get-Date).ToString('o')
        result_dir = $Ctx.ResultDir
        commands_file = 'commands.jsonl'
        assessment_file = 'assessment.json'
        notes = $Extra
    })
    $zip = "$($Ctx.ResultDir).zip"
    if (Test-Path -LiteralPath $zip) { Remove-Item -LiteralPath $zip -Force }
    Compress-Archive -Path (Join-Path $Ctx.ResultDir '*') -DestinationPath $zip -Force
    Write-Host "CHECK_RESULT_DIR=$($Ctx.ResultDir)"
    Write-Host "CHECK_RESULT_ZIP=$zip"
}

# Прогон должен выполняться на чистом рабочем дереве: иначе результаты нельзя отнести к коммиту.
$preflightStatus = @()
$preflightHead = ''
try {
    $preflightHead = (& git rev-parse HEAD 2>$null | Select-Object -First 1)
    $preflightStatus = @(& git status --short 2>$null |
        Where-Object { -not [string]::IsNullOrWhiteSpace($_) } |
        Where-Object { $_ -notmatch '(^|\s)\.check-results' })
} catch {
    $preflightStatus = @()
}
if ($preflightStatus.Count -gt 0 -and -not $AllowDirty) {
    $joined = ($preflightStatus -join [Environment]::NewLine)
    throw @"
Рабочее дерево не чистое, проверка остановлена: результаты нельзя привязать к коммиту.
$joined
Закоммитьте или отмените изменения, либо запустите с -AllowDirty, если грязный прогон нужен осознанно.
"@
}

$ctx = New-CheckContext -Student 'incident_card_check' -OutRoot $OutRoot
$notes = [ordered]@{}
$notes['worktree_clean'] = ($preflightStatus.Count -eq 0)
$notes['worktree_status'] = @($preflightStatus)
$notes['head_at_start'] = [string]$preflightHead
$cleanupTargets = New-Object System.Collections.ArrayList

# Все обязательные по заданию поля должны присутствовать: event_id, timestamp, user_id,
# machine_id, action, channel. Значения action берутся из перечисления задания.
$eventsLines = @(
    '{"event_id":"evt_after_boundary","timestamp":"2026-06-16T10:10:00Z","user_id":"user_ctx","machine_id":"pc_ctx","action":"delete_file","channel":"local","file_id":"ctx_6"}',
    '{"event_id":"evt_same_file_file_1","timestamp":"2026-06-16T11:01:00Z","user_id":"user_file_a","machine_id":"pc_file_a","action":"copy_file","channel":"local","file_id":"file_1"}',
    '{"event_id":"evt_before_mid","timestamp":"2026-06-16T09:45:00Z","user_id":"user_ctx","machine_id":"pc_ctx","action":"open_file","channel":"local","file_id":"ctx_2"}',
    '{"event_id":"evt_after_mid","timestamp":"2026-06-16T10:05:00Z","user_id":"user_ctx","machine_id":"pc_ctx","action":"print_file","channel":"printer","file_id":"ctx_4"}',
    '{"event_id":"evt_main","timestamp":"2026-06-16T10:00:00Z","user_id":"user_main","machine_id":"pc_main","action":"email_send","channel":"email","file_id":"file_1","file_name":"client_base.xlsx","destination_id":"dest_1","destination_type":"external","destination":"external_email_001","content_classes":["client_data"],"size_bytes":204800,"severity":"high"}',
    '{"event_id":"evt_outside_before","timestamp":"2026-06-16T09:29:59Z","user_id":"user_ctx","machine_id":"pc_ctx","action":"open_file","channel":"local","file_id":"ctx_0"}',
    '{"event_id":"evt_same_destination_dest_2","timestamp":"2026-06-16T11:07:00Z","user_id":"user_dest_b","machine_id":"pc_dest_b","action":"email_send","channel":"email","destination_id":"dest_1","destination_type":"external"}',
    '{"event_id":"evt_same_user_user_2","timestamp":"2026-06-16T11:05:00Z","user_id":"user_main","machine_id":"pc_main","action":"open_file","channel":"local","file_id":"user_2"}',
    '{"event_id":"evt_before_boundary","timestamp":"2026-06-16T09:30:00Z","user_id":"user_ctx","machine_id":"pc_ctx","action":"open_file","channel":"local","file_id":"ctx_1"}',
    '{"event_id":"evt_after_near","timestamp":"2026-06-16T10:09:00Z","user_id":"user_ctx","machine_id":"pc_ctx","action":"print_file","channel":"printer","file_id":"ctx_5"}',
    '{"event_id":"evt_same_destination_dest_1","timestamp":"2026-06-16T11:02:00Z","user_id":"user_dest_a","machine_id":"pc_dest_a","action":"email_send","channel":"email","destination_id":"dest_1","destination_type":"external"}',
    '{"event_id":"evt_same_overlap","timestamp":"2026-06-16T11:10:00Z","user_id":"user_main","machine_id":"pc_main","action":"email_send","channel":"email","file_id":"file_1","destination_id":"dest_1","destination_type":"external"}',
    '{"event_id":"evt_same_file_file_2","timestamp":"2026-06-16T11:06:00Z","user_id":"user_file_b","machine_id":"pc_file_b","action":"copy_file","channel":"local","file_id":"file_1"}',
    '{"event_id":"evt_before_near","timestamp":"2026-06-16T09:59:00Z","user_id":"user_ctx","machine_id":"pc_ctx","action":"open_file","channel":"local","file_id":"ctx_3"}',
    '{"event_id":"evt_same_user_user_1","timestamp":"2026-06-16T11:00:00Z","user_id":"user_main","machine_id":"pc_main","action":"open_file","channel":"local","file_id":"user_1"}',
    '{"event_id":"evt_outside_after","timestamp":"2026-06-16T10:10:01Z","user_id":"user_ctx","machine_id":"pc_ctx","action":"delete_file","channel":"local","file_id":"ctx_7"}'
)
$eventsPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/events.jsonl' -Content (($eventsLines -join "`n") + "`n")
# Первая строка валидна, битой должна быть именно вторая: так проверяется номер строки в сообщении.
$badEventsPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/events_bad.jsonl' -Content ('{"event_id":"evt_main","timestamp":"2026-06-16T10:00:00Z","user_id":"user_main","machine_id":"pc_main","action":"email_send","channel":"email"}' + "`n{bad-json-line}`n")
$requestPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/request.json' -Content '{"incident_id":"inc_001","main_event_id":"evt_main","window_before":"30m","window_after":"10m","include_same_user":true,"include_same_file":true,"include_same_destination":true,"max_events_per_section":50}'
$requestLimitPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/request_limit2.json' -Content '{"incident_id":"inc_limit","main_event_id":"evt_main","window_before":"30m","window_after":"10m","include_same_user":true,"include_same_file":true,"include_same_destination":true,"max_events_per_section":2}'
$requestFlagsOffPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/request_flags_off.json' -Content '{"incident_id":"inc_flags_off","main_event_id":"evt_main","window_before":"30m","window_after":"10m","include_same_user":false,"include_same_file":false,"include_same_destination":false,"max_events_per_section":3}'
# Значения вне документированного диапазона 1..1000 передаём через формат запроса из задания,
# а не через недокументированные CLI-флаги.
$requestLimitZeroPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/request_limit0.json' -Content '{"incident_id":"inc_limit0","main_event_id":"evt_main","window_before":"30m","window_after":"10m","max_events_per_section":0}'
$requestLimitHugePath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/request_limit1001.json' -Content '{"incident_id":"inc_limit1001","main_event_id":"evt_main","window_before":"30m","window_after":"10m","max_events_per_section":1001}'
$factorsPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/factors.yaml' -Content "factors:`n  - factor_id: factor_equals`n    title: Equals operator`n    condition:`n      field: destination_type`n      equals: external`n  - factor_id: factor_in`n    title: In operator`n    condition:`n      field: severity`n      in: [high, critical]`n  - factor_id: factor_contains`n    title: Contains operator`n    condition:`n      field: content_classes`n      contains: client_data`n  - factor_id: factor_exists`n    title: Exists operator`n    condition:`n      field: destination_id`n      exists: true`n  - factor_id: factor_negative`n    title: Negative operator`n    condition:`n      field: destination_type`n      equals: internal`n"

$tool = Join-Path $ctx.OutputsDir 'incident-card.exe'
$cardMd = Join-Path $ctx.OutputsDir 'card.md'
$cardJson = Join-Path $ctx.OutputsDir 'card.json'
$cardDot = Join-Path $ctx.OutputsDir 'card.dot'
$cardFlagsMd = Join-Path $ctx.OutputsDir 'card_flags.md'
$cardFlagsJson = Join-Path $ctx.OutputsDir 'card_flags.json'
$cardLimitMd = Join-Path $ctx.OutputsDir 'card_limit.md'
$cardLimitJson = Join-Path $ctx.OutputsDir 'card_limit.json'
$cardFlagsOffJson = Join-Path $ctx.OutputsDir 'card_flags_off.json'
$genA = Join-Path $ctx.OutputsDir 'generated_a.jsonl'
$genB = Join-Path $ctx.OutputsDir 'generated_b.jsonl'
$targetedJsonPath = Join-Path $ctx.OutputsDir 'targeted_go_test.jsonl'

Invoke-CheckCommand -Ctx $ctx -Name 'go_test_all' -Command "& '$($ctx.GoCmd)' test ./..."
Invoke-CheckCommand -Ctx $ctx -Name 'make_test' -Command 'make test'
Invoke-CheckCommand -Ctx $ctx -Name 'make_bench' -Command 'make bench'
Invoke-CheckCommand -Ctx $ctx -Name 'make_demo' -Command 'make demo'
Invoke-CheckCommand -Ctx $ctx -Name 'build_cli' -Command "& '$($ctx.GoCmd)' build -o '$tool' ./cmd/incident-card"
# Тесты запускаем по всему модулю: пакет с тестами заранее неизвестен.
Invoke-CheckCommand -Ctx $ctx -Name 'targeted_go_test' -Command "& '$($ctx.GoCmd)' test -json ./... | Set-Content -LiteralPath '$targetedJsonPath' -Encoding UTF8"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_request' -Command "& '$tool' build --events '$eventsPath' --request '$requestPath' --factors '$factorsPath' --out '$cardMd' --json '$cardJson' --dot '$cardDot'"
# Только флаги, документированные в разделе CLI задания.
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_flags' -Command "& '$tool' build --events '$eventsPath' --event-id evt_main --before 30m --after 10m --out '$cardFlagsMd' --json '$cardFlagsJson'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_limit2' -Command "& '$tool' build --events '$eventsPath' --request '$requestLimitPath' --out '$cardLimitMd' --json '$cardLimitJson'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_flags_off' -Command "& '$tool' build --events '$eventsPath' --request '$requestFlagsOffPath' --json '$cardFlagsOffJson'"
# Негативные сценарии запускаем без обёрток с `exit`: ожидаемый ненулевой код возврата
# проверяется ниже по записанному значению. Обёртка ещё и мешала раннеру зафиксировать код.
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_malformed' -Command "& '$tool' build --events '$badEventsPath' --event-id evt_main --out '$($ctx.OutputsDir)\invalid_malformed.md'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_unknown_main' -Command "& '$tool' build --events '$eventsPath' --event-id evt_unknown --out '$($ctx.OutputsDir)\invalid_unknown.md'"
# Задание не требует отвергать значения вне диапазона, поэтому фиксируем только устойчивость:
# без паники и без зависания, поведение записываем в runtime_validation.json.
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_limit0' -Command "& '$tool' build --events '$eventsPath' --request '$requestLimitZeroPath' --out '$($ctx.OutputsDir)\limit0.md' --json '$($ctx.OutputsDir)\limit0.json'" -TimeoutSec 120
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_limit1001' -Command "& '$tool' build --events '$eventsPath' --request '$requestLimitHugePath' --out '$($ctx.OutputsDir)\limit1001.md' --json '$($ctx.OutputsDir)\limit1001.json'" -TimeoutSec 120
Invoke-CheckCommand -Ctx $ctx -Name 'cli_generate_25_a' -Command "& '$tool' generate --count 25 --scenario external_send --seed 42 --out '$genA'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_generate_25_b' -Command "& '$tool' generate --count 25 --scenario external_send --seed 42 --out '$genB'"

$card = Read-CheckJson -Path $cardJson
$cardLimit = Read-CheckJson -Path $cardLimitJson
$cardFlagsOff = Read-CheckJson -Path $cardFlagsOffJson
$cardMdText = Get-CheckFileText -Path $cardMd
$cardDotText = Get-CheckFileText -Path $cardDot
$targetedText = Get-CheckFileText -Path $targetedJsonPath

# Имена полей взяты из примера выходного JSON в задании.
# Граница временного окна и включение главного события в разделы same_* заданием не оговорены,
# поэтому для каждого раздела допускается несколько корректных последовательностей.
$allowedBefore = @(
    ,@('evt_before_boundary', 'evt_before_mid', 'evt_before_near')
    ,@('evt_before_mid', 'evt_before_near')
)
$allowedAfter = @(
    ,@('evt_after_mid', 'evt_after_near', 'evt_after_boundary')
    ,@('evt_after_mid', 'evt_after_near')
)
$allowedSameUser = @(
    ,@('evt_same_user_user_1', 'evt_same_user_user_2', 'evt_same_overlap')
    ,@('evt_main', 'evt_same_user_user_1', 'evt_same_user_user_2', 'evt_same_overlap')
)
$allowedSameFile = @(
    ,@('evt_same_file_file_1', 'evt_same_file_file_2', 'evt_same_overlap')
    ,@('evt_main', 'evt_same_file_file_1', 'evt_same_file_file_2', 'evt_same_overlap')
)
$allowedSameDestination = @(
    ,@('evt_same_destination_dest_1', 'evt_same_destination_dest_2', 'evt_same_overlap')
    ,@('evt_main', 'evt_same_destination_dest_1', 'evt_same_destination_dest_2', 'evt_same_overlap')
)
# Пятое правило factor_negative не должно срабатывать; порядок факторов заданием не определён.
$expectedFactors = @('factor_equals', 'factor_in', 'factor_contains', 'factor_exists')

$arraysOk = $false
$timelineOk = $false
$summaryOk = $false
$factorsOk = $false
$sameFileOk = $false
$sameDestinationOk = $false
if ($null -ne $card) {
    $arraysOk = (Test-ArrayIsOneOf -Actual @(Get-JsonArray -Object $card -Name 'context_before') -Allowed $allowedBefore) -and
        (Test-ArrayIsOneOf -Actual @(Get-JsonArray -Object $card -Name 'context_after') -Allowed $allowedAfter) -and
        (Test-ArrayIsOneOf -Actual @(Get-JsonArray -Object $card -Name 'same_user_events') -Allowed $allowedSameUser)
    $sameFileOk = Test-ArrayIsOneOf -Actual @(Get-JsonArray -Object $card -Name 'same_file_events') -Allowed $allowedSameFile
    $sameDestinationOk = Test-ArrayIsOneOf -Actual @(Get-JsonArray -Object $card -Name 'same_destination_events') -Allowed $allowedSameDestination
    $timeline = @(Get-JsonArray -Object $card -Name 'timeline')
    $mainRows = @($timeline | Where-Object {
        ([string](Get-JsonProperty -Object $_ -Name 'role') -eq 'main_event') -and
        ([string](Get-JsonProperty -Object $_ -Name 'event_id') -eq 'evt_main')
    })
    $timelineOk = ($timeline.Count -gt 0) -and (Test-TimelineSortedDedup -Timeline $timeline) -and ($mainRows.Count -eq 1)
    # Резюме должно быть содержательным и зависеть от данных, конкретных слов задание не требует.
    $summaryText = [string](Get-JsonProperty -Object $card -Name 'summary')
    $summaryOk = ($summaryText.Trim().Length -ge 10) -and (($summaryText -match 'evt_main') -or ($summaryText -match 'user_main') -or ($summaryText -match 'client_base'))
    $factorsOk = (Test-SetEquals -Actual @(Get-JsonArray -Object $card -Name 'suspicious_factors') -Expected $expectedFactors)
}
# Задание требует десять разделов, таблицу таймлайна и ссылки на исходные события.
$mdHeadings = @([regex]::Matches($cardMdText, '(?m)^#{1,6}\s+\S'))
$mdTimelineTable = ($cardMdText -match '(?m)^\|(?:[^|\r\n]*\|){7,}')
$markdownOk = ($mdHeadings.Count -ge 10) -and $mdTimelineTable -and ($cardMdText -match 'evt_main')
# Имя графа заданием не задано, проверяем структуру: объявление, главный узел и хотя бы одно ребро.
$dotOk = ($cardDotText -match '(?m)^\s*(strict\s+)?digraph\b') -and ($cardDotText -match 'evt_main') -and ($cardDotText -match '->')

# Минимум, пункт 5: тесты поиска и сортировки. Пакет и имена тестов заданием не заданы,
# поэтому ищем прошедшие тесты по смыслу имени во всём модуле.
$passedTests = Get-PassedTestNames -JsonLines $targetedText
$searchTests = @($passedTests | Where-Object { $_ -match '(?i)find|search|index|lookup|relation' })
$sortTests = @($passedTests | Where-Object { $_ -match '(?i)timeline|sort|dedup|order' })
$targetedTestsOk = ($ctx.CommandResults['targeted_go_test'].exit_code -eq 0) -and ($searchTests.Count -gt 0) -and ($sortTests.Count -gt 0)

$malformedLog = Get-CommandLogText -Ctx $ctx -Name 'cli_build_malformed'
$unknownMainLog = Get-CommandLogText -Ctx $ctx -Name 'cli_build_unknown_main'
$limit0Log = Get-CommandLogText -Ctx $ctx -Name 'cli_build_limit0'
$limit1001Log = Get-CommandLogText -Ctx $ctx -Name 'cli_build_limit1001'
# Язык и формулировка сообщения заданием не заданы; требуется указание файла, строки или поля.
$malformedRejected = ([int]$ctx.CommandResults['cli_build_malformed'].exit_code -ne 0) -and ($malformedLog -match 'events_bad\.jsonl[:\s]*2\b')
$unknownMainRejected = ([int]$ctx.CommandResults['cli_build_unknown_main'].exit_code -ne 0) -and ($unknownMainLog -match 'evt_unknown')
# Отвергать значение вне диапазона задание не обязывает: допустимы и явная ошибка, и подстановка
# значения по умолчанию. Недопустимы паника, зависание и превышение лимита в отчёте.
function Get-LimitBehaviour {
    param($Ctx, [string]$Name, [string]$LogText, [string]$JsonPath, [int]$MaxAllowedRows)
    $result = $Ctx.CommandResults[$Name]
    $panicked = ($LogText -match '(?m)^panic:') -or ($LogText -match 'goroutine \d+ \[running\]')
    $timedOut = [bool]$result.timed_out
    $behaviour = 'unknown'
    $sectionsOk = $true
    if ([int]$result.exit_code -ne 0) {
        $behaviour = 'rejected'
    } else {
        $behaviour = 'accepted_with_fallback'
        $produced = Read-CheckJson -Path $JsonPath
        if ($null -eq $produced) {
            $sectionsOk = $false
        } else {
            foreach ($section in @('context_before', 'context_after', 'same_user_events', 'same_file_events', 'same_destination_events', 'timeline')) {
                if (@(Get-JsonArray -Object $produced -Name $section).Count -gt $MaxAllowedRows) { $sectionsOk = $false }
            }
        }
    }
    return [ordered]@{
        behaviour = $behaviour
        handled = ((-not $panicked) -and (-not $timedOut) -and $sectionsOk)
        panicked = $panicked
        timed_out = $timedOut
        exit_code = [int]$result.exit_code
    }
}
$limitZero = Get-LimitBehaviour -Ctx $ctx -Name 'cli_build_limit0' -LogText $limit0Log -JsonPath (Join-Path $ctx.OutputsDir 'limit0.json') -MaxAllowedRows 1000
$limitHuge = Get-LimitBehaviour -Ctx $ctx -Name 'cli_build_limit1001' -LogText $limit1001Log -JsonPath (Join-Path $ctx.OutputsDir 'limit1001.json') -MaxAllowedRows 1001

# Задание: в разделе Markdown не больше max_events_per_section событий,
# при усечении отчёт должен явно об этом сказать.
$limitMdText = Get-CheckFileText -Path $cardLimitMd
$limitsOk = $false
if ($null -ne $cardLimit) {
    $limitsOk = $true
    foreach ($section in @('context_before', 'context_after', 'same_user_events', 'same_file_events', 'same_destination_events', 'timeline')) {
        if (@(Get-JsonArray -Object $cardLimit -Name $section).Count -gt 2) { $limitsOk = $false }
    }
    $limitTimeline = @(Get-JsonArray -Object $cardLimit -Name 'timeline')
    if ($limitTimeline.Count -eq 0) { $limitsOk = $false }
    if (-not (Test-TimelineSortedDedup -Timeline $limitTimeline)) { $limitsOk = $false }
}
# Ищем «усеч», «обрез», «показан», truncat, shown.
# Файл обязан храниться в UTF-8 с BOM: Windows PowerShell 5.1 читает UTF-8 без BOM как ANSI,
# и тогда этот шаблон, как и весь скрипт, разбирается неверно.
$truncationNoticed = ($limitMdText -match '(?i)усеч|обрез|показан|truncat|shown')
$limitsOk = $limitsOk -and $truncationNoticed

# Значения из файла запроса должны применяться: разделы same_* выключены, лимит 3.
$requestRespectedOk = $false
if ($null -ne $cardFlagsOff) {
    $requestRespectedOk = (@(Get-JsonArray -Object $cardFlagsOff -Name 'same_user_events').Count -eq 0) -and
        (@(Get-JsonArray -Object $cardFlagsOff -Name 'same_file_events').Count -eq 0) -and
        (@(Get-JsonArray -Object $cardFlagsOff -Name 'same_destination_events').Count -eq 0) -and
        (@(Get-JsonArray -Object $cardFlagsOff -Name 'context_before').Count -le 3) -and
        (@(Get-JsonArray -Object $cardFlagsOff -Name 'context_after').Count -le 3) -and
        (@(Get-JsonArray -Object $cardFlagsOff -Name 'timeline').Count -gt 0)
}

# Сценарий external_send описывает адресата, а не действие: значение action берётся
# из перечисления задания, где действия external_send нет.
$allowedActions = @('open_file', 'copy_file', 'create_archive', 'email_send', 'cloud_upload', 'messenger_send', 'copy_to_usb', 'delete_file', 'print_file')
$generatorOk = $false
$generatorDetails = ''
if ((Test-Path -LiteralPath $genA) -and (Test-Path -LiteralPath $genB)) {
    $genAStats = Get-FileStatsUniqueIDs -Path $genA
    # Детерминированность сравниваем по байтам, а не по декодированному тексту.
    $rawA = (Get-FileHash -LiteralPath $genA -Algorithm SHA256).Hash
    $rawB = (Get-FileHash -LiteralPath $genB -Algorithm SHA256).Hash
    $scenarioOk = $true
    $lines = @((Get-CheckFileText -Path $genA) -split "`r?`n" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    foreach ($line in $lines) {
        $obj = $line | ConvertFrom-Json
        if (-not (Test-Rfc3339 -Value ([string](Get-JsonProperty -Object $obj -Name 'timestamp')))) { $scenarioOk = $false; $generatorDetails = 'timestamp не в формате RFC3339'; break }
        foreach ($field in @('event_id', 'user_id', 'machine_id', 'action', 'channel')) {
            if ([string]::IsNullOrWhiteSpace([string](Get-JsonProperty -Object $obj -Name $field))) { $scenarioOk = $false; $generatorDetails = "отсутствует обязательное поле $field" ; break }
        }
        if (-not $scenarioOk) { break }
        $action = [string](Get-JsonProperty -Object $obj -Name 'action')
        if ($allowedActions -notcontains $action) { $scenarioOk = $false; $generatorDetails = "action вне перечисления: $action"; break }
        if ([string](Get-JsonProperty -Object $obj -Name 'destination_type') -ne 'external') { $scenarioOk = $false; $generatorDetails = 'сценарий external_send не даёт destination_type=external'; break }
    }
    $generatorOk = ($genAStats.lines -eq 25) -and ($genAStats.duplicates -eq 0) -and ($rawA -eq $rawB) -and $scenarioOk
}
Save-CheckJson -Path (Join-Path $ctx.OutputsDir 'runtime_validation.json') -Value ([ordered]@{
    arrays_ok = ($arraysOk -and $sameFileOk -and $sameDestinationOk)
    timeline_sorted_dedup = $timelineOk
    summary_dynamic = $summaryOk
    markdown_required_sections = $markdownOk
    markdown_headings = $mdHeadings.Count
    dot_required_markers = $dotOk
    targeted_tests_ok = $targetedTestsOk
    targeted_tests_search = @($searchTests)
    targeted_tests_sort = @($sortTests)
    malformed_rejected = $malformedRejected
    unknown_main_rejected = $unknownMainRejected
    limit_zero = $limitZero
    limit_1001 = $limitHuge
    section_limit_respected = $limitsOk
    truncation_reported = $truncationNoticed
    request_respected = $requestRespectedOk
    generator_ok = $generatorOk
    generator_details = $generatorDetails
    factors_ok = $factorsOk
})

$millionEventsPath = Join-Path $ctx.TmpDir 'events_1000000.jsonl'
$millionCardPath = Join-Path $ctx.TmpDir 'card_1000000.json'
$millionMdPath = Join-Path $ctx.TmpDir 'card_1000000.md'
$millionDotPath = Join-Path $ctx.TmpDir 'card_1000000.dot'
$null = $cleanupTargets.Add($millionEventsPath)
$null = $cleanupTargets.Add($millionCardPath)
$null = $cleanupTargets.Add($millionMdPath)
$null = $cleanupTargets.Add($millionDotPath)
$driveName = [System.IO.Path]::GetPathRoot($ctx.ResultDir).TrimEnd('\').TrimEnd(':')
$drive = Get-PSDrive -Name $driveName
$requiredBytes = 2GB
$diskPreflightOk = $drive.Free -ge $requiredBytes
$millionStart = Get-Date
$millionMainEventId = ''
if ($diskPreflightOk) {
    Invoke-CheckCommand -Ctx $ctx -Name 'cli_generate_1m' -Command "& '$tool' generate --count 1000000 --scenario external_send --seed 42 --out '$millionEventsPath'" -TimeoutSec 600
    # Схема идентификаторов генератора заданием не определена, поэтому главное событие
    # берём из самого файла, а не подставляем ожидаемый evt_0500000.
    $millionMainEventId = Get-EventIdAtLine -Path $millionEventsPath -LineNumber 500000
    if ($millionMainEventId -ne '') {
        $millionRequestPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/request_1m.json' -Content ('{"incident_id":"inc_1m","main_event_id":"' + $millionMainEventId + '","window_before":"30m","window_after":"10m","include_same_user":false,"include_same_file":false,"include_same_destination":false,"max_events_per_section":50}')
        Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_1m' -Command "& '$tool' build --events '$millionEventsPath' --request '$millionRequestPath' --out '$millionMdPath' --json '$millionCardPath' --dot '$millionDotPath'" -TimeoutSec 600
    }
}
$millionEnd = Get-Date
$millionDurationMs = [int](($millionEnd - $millionStart).TotalMilliseconds)
$millionStats = if (Test-Path -LiteralPath $millionEventsPath) { Get-FileStatsUniqueIDs -Path $millionEventsPath } else { [ordered]@{ lines = 0; unique_ids = 0; duplicates = 1 } }
$millionEventsBytes = if (Test-Path -LiteralPath $millionEventsPath) { (Get-Item -LiteralPath $millionEventsPath).Length } else { 0 }
$millionCardBytes = if (Test-Path -LiteralPath $millionCardPath) { (Get-Item -LiteralPath $millionCardPath).Length } else { 0 }
$gen1m = if ($ctx.CommandResults.ContainsKey('cli_generate_1m')) { $ctx.CommandResults['cli_generate_1m'] } else { [ordered]@{ exit_code = -1; timed_out = $true; peak_working_set_bytes = 0; duration_ms = 0 } }
$build1m = if ($ctx.CommandResults.ContainsKey('cli_build_1m')) { $ctx.CommandResults['cli_build_1m'] } else { [ordered]@{ exit_code = -1; timed_out = $true; peak_working_set_bytes = 0; duration_ms = 0 } }
$peakLimit = 768MB
$millionMemoryOk = ($gen1m.peak_working_set_bytes -le $peakLimit) -and ($build1m.peak_working_set_bytes -le $peakLimit)
$benchLog = Get-CommandLogText -Ctx $ctx -Name 'make_bench'
# Имя бенчмарка заданием не задано: требуется работающий бенчмарк на 1 000 000 событий.
$benchRan = ($benchLog -match '(?m)^Benchmark\S+') -and ($benchLog -match 'ns/op')
$benchMillionSources = @(Get-RepoTestFiles -Ctx $ctx |
    Where-Object { (Get-Content -LiteralPath $_.FullName -Raw) -match '(?s)func\s+Benchmark' } |
    Where-Object { (Get-Content -LiteralPath $_.FullName -Raw) -match '1[_\s]?000[_\s]?000' })
$benchmarkOk = $benchRan -and ($benchMillionSources.Count -gt 0)
$millionOk = $diskPreflightOk -and ($gen1m.exit_code -eq 0) -and (-not [bool]$gen1m.timed_out) -and ($build1m.exit_code -eq 0) -and (-not [bool]$build1m.timed_out) -and ($millionStats.lines -eq 1000000) -and ($millionStats.duplicates -eq 0) -and ($millionCardBytes -gt 0) -and $millionMemoryOk -and $benchmarkOk
Save-CheckJson -Path (Join-Path $ctx.OutputsDir 'million_metrics.json') -Value ([ordered]@{
    disk_preflight_ok = $diskPreflightOk
    disk_free_bytes = $drive.Free
    required_bytes = $requiredBytes
    generate = $gen1m
    build = $build1m
    lines = $millionStats.lines
    events_file_bytes = $millionEventsBytes
    card_json_bytes = $millionCardBytes
    unique_ids = $millionStats.unique_ids
    duplicates = $millionStats.duplicates
    memory_limit_bytes = $peakLimit
    memory_ok = $millionMemoryOk
    main_event_id = $millionMainEventId
    benchmark_ok = $benchmarkOk
    benchmark_million_sources = @($benchMillionSources | ForEach-Object { $_.Name })
    duration_ms = $millionDurationMs
})
foreach ($target in @($cleanupTargets)) { if (Test-Path -LiteralPath $target) { Remove-Item -LiteralPath $target -Force -ErrorAction SilentlyContinue } }
$cleanupOk = (-not (Test-Path -LiteralPath $millionEventsPath)) -and (-not (Test-Path -LiteralPath $millionCardPath)) -and (-not (Test-Path -LiteralPath $millionMdPath)) -and (-not (Test-Path -LiteralPath $millionDotPath))

Save-CheckJson -Path (Join-Path $ctx.OutputsDir 'artifact_presence.json') -Value ([ordered]@{
    card_md = Test-Path -LiteralPath $cardMd
    card_json = Test-Path -LiteralPath $cardJson
    card_dot = Test-Path -LiteralPath $cardDot
    card_flags_md = Test-Path -LiteralPath $cardFlagsMd
    card_flags_json = Test-Path -LiteralPath $cardFlagsJson
    card_limit_md = Test-Path -LiteralPath $cardLimitMd
    card_limit_json = Test-Path -LiteralPath $cardLimitJson
    card_flags_off_json = Test-Path -LiteralPath $cardFlagsOffJson
    generated_a = Test-Path -LiteralPath $genA
    generated_b = Test-Path -LiteralPath $genB
})

Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.jsonl_reader' -Level 'minimum' -Category 'input' -Requirement 'Read JSONL events and reject malformed input with line number' -CommandName 'cli_build_flags' -RequiredArtifacts @($cardFlagsJson, $cardFlagsMd) -ExtraConformant $malformedRejected -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.main_event' -Level 'minimum' -Category 'algorithm' -Requirement 'Find exact main event and reject unknown event_id' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant $unknownMainRejected -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.time_context' -Level 'minimum' -Category 'algorithm' -Requirement 'Before/after windows select the correct events sorted by time' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant ($arraysOk -and $sameFileOk -and $sameDestinationOk) -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.markdown_card' -Level 'minimum' -Category 'format' -Requirement 'Markdown incident card has required headings/table and dynamic summary' -CommandName 'cli_build_request' -RequiredArtifacts @($cardMd) -ExtraConformant $markdownOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.search_sort_tests' -Level 'minimum' -Category 'tests' -Requirement 'Module tests cover event search and timeline sorting' -CommandName 'targeted_go_test' -RequiredArtifacts @($targetedJsonPath) -ExtraConformant $targetedTestsOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.same_file' -Level 'good' -Category 'algorithm' -Requirement 'Related events by same file are correct' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant $sameFileOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.same_destination' -Level 'good' -Category 'algorithm' -Requirement 'Related events by same destination are correct' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant $sameDestinationOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.json_card' -Level 'good' -Category 'format' -Requirement 'JSON card is consistent with arrays and timeline' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant ($arraysOk -and $timelineOk) -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.suspicious_factors' -Level 'good' -Category 'algorithm' -Requirement 'Suspicious factors from YAML operators equals/in/contains/exists' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant $factorsOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.generator' -Level 'good' -Category 'cli' -Requirement 'Generator produces 25 valid unique deterministic events for scenario external_send' -CommandName 'cli_generate_25_a' -RequiredArtifacts @($genA, $genB) -ExtraConformant $generatorOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.dot_graph' -Level 'excellent' -Category 'format' -Requirement 'DOT graph declares a digraph with the main node and relations' -CommandName 'cli_build_request' -RequiredArtifacts @($cardDot) -ExtraConformant $dotOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.section_limits' -Level 'excellent' -Category 'report' -Requirement 'Sections respect max_events_per_section, truncation is reported, out-of-range values are handled without panic' -CommandName 'cli_build_limit2' -RequiredArtifacts @($cardLimitJson, $cardLimitMd) -ExtraConformant ($limitsOk -and $requestRespectedOk -and [bool]$limitZero.handled -and [bool]$limitHuge.handled) -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.million_benchmark' -Level 'excellent' -Category 'performance' -Requirement 'Real 1M run and benchmark pass with memory/time constraints' -CommandName 'cli_build_1m' -RequiredArtifacts @((Join-Path $ctx.OutputsDir 'million_metrics.json')) -ExtraConformant $millionOk -ExtraEvidence @('outputs/million_metrics.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.polished_report' -Level 'excellent' -Category 'report' -Requirement 'Polished markdown+json report is sorted, deduped and consistent' -CommandName 'cli_build_request' -RequiredArtifacts @($cardMd, $cardJson) -ExtraConformant ($timelineOk -and $summaryOk -and $markdownOk -and $factorsOk) -ExtraEvidence @('outputs/runtime_validation.json')

$notes.cleanup_ok = $cleanupOk
$notes.expected_score = [ordered]@{ minimum = 5; good = 5; excellent = 4; engineering = 12; total = 26 }
$notes.runtime_validation = 'outputs/runtime_validation.json'
$notes.million_metrics = 'outputs/million_metrics.json'

Complete-Check -Ctx $ctx -Extra $notes


