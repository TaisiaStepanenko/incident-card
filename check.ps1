param(
    [string]$OutRoot = ''
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
    '' | Set-Content -LiteralPath $ctx.CommandsPath -Encoding UTF8
    return $ctx
}

function Save-CheckJson {
    param([string]$Path, $Value)
    ($Value | ConvertTo-Json -Depth 50) | Set-Content -LiteralPath $Path -Encoding UTF8
}

function Write-CheckText {
    param($Ctx, [string]$RelativePath, [string]$Content)
    $path = Join-Path $Ctx.ResultDir $RelativePath
    $parent = Split-Path -Parent $path
    if ($parent) { New-Item -ItemType Directory -Force -Path $parent | Out-Null }
    Set-Content -LiteralPath $path -Value $Content -Encoding UTF8
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

function Invoke-CheckCommand {
    param($Ctx, [string]$Name, [string]$Command, [string]$WorkingDirectory = '', [int]$TimeoutSec = 0)
    if ($WorkingDirectory -eq '') { $WorkingDirectory = $Ctx.RepoRoot }
    $safe = $Name -replace '[^A-Za-z0-9_.-]', '_'
    $runnerPath = Join-Path $Ctx.TmpDir "$safe.ps1"
    $stdoutPath = Join-Path $Ctx.TmpDir "$safe.stdout.log"
    $stderrPath = Join-Path $Ctx.TmpDir "$safe.stderr.log"
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
        '    exit $exitCode'
        '} catch {'
        '    Write-Error $_'
        '    exit 1'
        '}'
    ) -join "`r`n"
    Set-Content -LiteralPath $runnerPath -Value $runner -Encoding UTF8
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
        $exitCode = if ($null -ne $LASTEXITCODE) { [int]$LASTEXITCODE } else { 0 }
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
        $exitCode = if ($timedOut) {
            124
        } else {
            $proc.Refresh()
            [int]$proc.ExitCode
        }
        $stdout = if (Test-Path -LiteralPath $stdoutPath) { Get-Content -LiteralPath $stdoutPath -Raw } else { '' }
        $stderr = if (Test-Path -LiteralPath $stderrPath) { Get-Content -LiteralPath $stderrPath -Raw } else { '' }
    }
    $ended = Get-Date

    @(
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
    ) | Set-Content -LiteralPath $logPath -Encoding UTF8

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
    $raw = Get-Content -LiteralPath $Path -Raw
    if ([string]::IsNullOrWhiteSpace($raw)) { return $null }
    return $raw | ConvertFrom-Json
}

function Test-ArrayExact {
    param([object[]]$Actual, [object[]]$Expected)
    if ($Actual.Count -ne $Expected.Count) { return $false }
    for ($i = 0; $i -lt $Actual.Count; $i++) { if ([string]$Actual[$i] -ne [string]$Expected[$i]) { return $false } }
    return $true
}

function Test-TimelineSortedDedup {
    param([object[]]$Timeline)
    $seen = @{}
    $lastTs = ''
    $lastID = ''
    foreach ($item in $Timeline) {
        $id = [string]$item.event_id
        $ts = [string]$item.timestamp
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
    if (-not (Test-Path -LiteralPath $path)) { return '' }
    return Get-Content -LiteralPath $path -Raw
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
    $testFiles = @(Get-ChildItem -LiteralPath $Ctx.RepoRoot -Recurse -File -Filter '*_test.go' -ErrorAction SilentlyContinue | Where-Object { $_.FullName -notlike '*\.check-results\*' })
    $tests = @($testFiles | Select-String -Pattern '^\s*func\s+Test[A-Za-z0-9_]+\s*\(' -ErrorAction SilentlyContinue)
    $benches = @($testFiles | Select-String -Pattern '^\s*func\s+Benchmark[A-Za-z0-9_]+\s*\(' -ErrorAction SilentlyContinue)
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.unit_tests_present' -Level 'engineering' -Category 'tests' -Requirement 'Go unit tests are present' -Ok ($tests.Count -gt 0) -Evidence @('cmd/incident-card/main_test.go')
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.benchmarks_present' -Level 'engineering' -Category 'benchmarks' -Requirement 'Go benchmark tests are present' -Ok ($benches.Count -gt 0) -Evidence @('cmd/incident-card/main_test.go')
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

$ctx = New-CheckContext -Student 'incident_card_check' -OutRoot $OutRoot
$notes = [ordered]@{}
$cleanupTargets = New-Object System.Collections.ArrayList

$eventsLines = @(
    '{"event_id":"evt_after_boundary","timestamp":"2026-06-16T10:10:00Z","user_id":"user_ctx","action":"delete_file","channel":"local","file_id":"ctx_6"}',
    '{"event_id":"evt_same_file_file_1","timestamp":"2026-06-16T11:01:00Z","user_id":"user_file_a","action":"copy_file","channel":"local","file_id":"file_1"}',
    '{"event_id":"evt_before_mid","timestamp":"2026-06-16T09:45:00Z","user_id":"user_ctx","action":"open_file","channel":"local","file_id":"ctx_2"}',
    '{"event_id":"evt_after_mid","timestamp":"2026-06-16T10:05:00Z","user_id":"user_ctx","action":"print","channel":"local","file_id":"ctx_4"}',
    '{"event_id":"evt_main","timestamp":"2026-06-16T10:00:00Z","user_id":"user_main","action":"external_send","channel":"email","file_id":"file_1","destination_id":"dest_1","destination_type":"external","content_classes":["client_data"],"severity":"high"}',
    '{"event_id":"evt_outside_before","timestamp":"2026-06-16T09:29:59Z","user_id":"user_ctx","action":"open_file","channel":"local","file_id":"ctx_0"}',
    '{"event_id":"evt_same_destination_dest_2","timestamp":"2026-06-16T11:07:00Z","user_id":"user_dest_b","action":"external_send","channel":"email","destination_id":"dest_1","destination_type":"external"}',
    '{"event_id":"evt_same_user_user_2","timestamp":"2026-06-16T11:05:00Z","user_id":"user_main","action":"open_file","channel":"local","file_id":"user_2"}',
    '{"event_id":"evt_before_boundary","timestamp":"2026-06-16T09:30:00Z","user_id":"user_ctx","action":"open_file","channel":"local","file_id":"ctx_1"}',
    '{"event_id":"evt_after_near","timestamp":"2026-06-16T10:09:00Z","user_id":"user_ctx","action":"print","channel":"local","file_id":"ctx_5"}',
    '{"event_id":"evt_same_destination_dest_1","timestamp":"2026-06-16T11:02:00Z","user_id":"user_dest_a","action":"external_send","channel":"email","destination_id":"dest_1","destination_type":"external"}',
    '{"event_id":"evt_same_overlap","timestamp":"2026-06-16T11:10:00Z","user_id":"user_main","action":"external_send","channel":"email","file_id":"file_1","destination_id":"dest_1","destination_type":"external"}',
    '{"event_id":"evt_same_file_file_2","timestamp":"2026-06-16T11:06:00Z","user_id":"user_file_b","action":"copy_file","channel":"local","file_id":"file_1"}',
    '{"event_id":"evt_before_near","timestamp":"2026-06-16T09:59:00Z","user_id":"user_ctx","action":"open_file","channel":"local","file_id":"ctx_3"}',
    '{"event_id":"evt_same_user_user_1","timestamp":"2026-06-16T11:00:00Z","user_id":"user_main","action":"open_file","channel":"local","file_id":"user_1"}',
    '{"event_id":"evt_outside_after","timestamp":"2026-06-16T10:10:01Z","user_id":"user_ctx","action":"delete_file","channel":"local","file_id":"ctx_7"}'
)
$eventsPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/events.jsonl' -Content (($eventsLines -join "`n") + "`n")
$badEventsPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/events_bad.jsonl' -Content ('{"event_id":"evt_main","timestamp":"2026-06-16T10:00:00Z","user_id":"user_main","action":"external_send","channel":"email"}' + "`n{bad-json-line}`n")
$requestPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/request.json' -Content '{"incident_id":"inc_001","main_event_id":"evt_main","window_before":"30m","window_after":"10m","include_same_user":true,"include_same_file":true,"include_same_destination":true,"max_events_per_section":50}'
$requestLimitPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/request_limit2.json' -Content '{"incident_id":"inc_limit","main_event_id":"evt_main","window_before":"30m","window_after":"10m","include_same_user":true,"include_same_file":true,"include_same_destination":true,"max_events_per_section":2}'
$requestPrecedencePath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/request_precedence.json' -Content '{"incident_id":"inc_precedence","main_event_id":"evt_main","window_before":"30m","window_after":"10m","include_same_user":false,"include_same_file":false,"include_same_destination":false,"max_events_per_section":3}'
$factorsPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/factors.yaml' -Content "factors:`n  - factor_id: factor_equals`n    title: Equals operator`n    condition:`n      field: destination_type`n      equals: external`n  - factor_id: factor_in`n    title: In operator`n    condition:`n      field: severity`n      in: [high, critical]`n  - factor_id: factor_contains`n    title: Contains operator`n    condition:`n      field: content_classes`n      contains: client_data`n  - factor_id: factor_exists`n    title: Exists operator`n    condition:`n      field: destination_id`n      exists: true`n  - factor_id: factor_negative`n    title: Negative operator`n    condition:`n      field: destination_type`n      equals: internal`n"

$tool = Join-Path $ctx.OutputsDir 'incident-card.exe'
$cardMd = Join-Path $ctx.OutputsDir 'card.md'
$cardJson = Join-Path $ctx.OutputsDir 'card.json'
$cardDot = Join-Path $ctx.OutputsDir 'card.dot'
$cardFlagsMd = Join-Path $ctx.OutputsDir 'card_flags.md'
$cardFlagsJson = Join-Path $ctx.OutputsDir 'card_flags.json'
$cardLimitMd = Join-Path $ctx.OutputsDir 'card_limit.md'
$cardLimitJson = Join-Path $ctx.OutputsDir 'card_limit.json'
$cardPrecedenceJson = Join-Path $ctx.OutputsDir 'card_precedence.json'
$genA = Join-Path $ctx.OutputsDir 'generated_a.jsonl'
$genB = Join-Path $ctx.OutputsDir 'generated_b.jsonl'
$targetedJsonPath = Join-Path $ctx.OutputsDir 'targeted_go_test.jsonl'

Invoke-CheckCommand -Ctx $ctx -Name 'go_test_all' -Command "& '$($ctx.GoCmd)' test ./..."
Invoke-CheckCommand -Ctx $ctx -Name 'make_test' -Command 'make test'
Invoke-CheckCommand -Ctx $ctx -Name 'make_bench' -Command 'make bench'
Invoke-CheckCommand -Ctx $ctx -Name 'make_demo' -Command 'make demo'
Invoke-CheckCommand -Ctx $ctx -Name 'build_cli' -Command "& '$($ctx.GoCmd)' build -o '$tool' ./cmd/incident-card"
Invoke-CheckCommand -Ctx $ctx -Name 'targeted_go_test' -Command "& '$($ctx.GoCmd)' test -json -run 'TestFindAndSearchRelationships|TestTimelineSortDedup' ./cmd/incident-card | Set-Content -LiteralPath '$targetedJsonPath' -Encoding UTF8"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_request' -Command "& '$tool' build --events '$eventsPath' --request '$requestPath' --factors '$factorsPath' --out '$cardMd' --json '$cardJson' --dot '$cardDot'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_flags' -Command "& '$tool' build --events '$eventsPath' --event-id evt_main --before 30m --after 10m --include-same-user=true --include-same-file=true --include-same-destination=true --max-events-per-section 50 --out '$cardFlagsMd' --json '$cardFlagsJson'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_limit2' -Command "& '$tool' build --events '$eventsPath' --request '$requestLimitPath' --out '$cardLimitMd' --json '$cardLimitJson'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_precedence' -Command "& '$tool' build --events '$eventsPath' --request '$requestPrecedencePath' --include-same-user=true --max-events-per-section 2 --json '$cardPrecedenceJson'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_malformed' -Command "& '$tool' build --events '$badEventsPath' --event-id evt_main --out '$($ctx.OutputsDir)\invalid_malformed.md'; if (`$LASTEXITCODE -ne 0) { exit 0 } else { Write-Error 'expected malformed rejection'; exit 1 }"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_unknown_main' -Command "& '$tool' build --events '$eventsPath' --event-id evt_unknown --out '$($ctx.OutputsDir)\invalid_unknown.md'; if (`$LASTEXITCODE -ne 0) { exit 0 } else { Write-Error 'expected unknown main rejection'; exit 1 }"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_limit0' -Command "& '$tool' build --events '$eventsPath' --event-id evt_main --max-events-per-section 0 --out '$($ctx.OutputsDir)\invalid_limit0.md'; if (`$LASTEXITCODE -ne 0) { exit 0 } else { Write-Error 'expected limit0 rejection'; exit 1 }"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_limit1001' -Command "& '$tool' build --events '$eventsPath' --event-id evt_main --max-events-per-section 1001 --out '$($ctx.OutputsDir)\invalid_limit1001.md'; if (`$LASTEXITCODE -ne 0) { exit 0 } else { Write-Error 'expected limit1001 rejection'; exit 1 }"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_generate_25_a' -Command "& '$tool' generate --count 25 --scenario external_send --seed 42 --out '$genA'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_generate_25_b' -Command "& '$tool' generate --count 25 --scenario external_send --seed 42 --out '$genB'"

$card = Read-CheckJson -Path $cardJson
$cardLimit = Read-CheckJson -Path $cardLimitJson
$cardPrecedence = Read-CheckJson -Path $cardPrecedenceJson
$cardMdText = if (Test-Path -LiteralPath $cardMd) { Get-Content -LiteralPath $cardMd -Raw } else { '' }
$cardDotText = if (Test-Path -LiteralPath $cardDot) { Get-Content -LiteralPath $cardDot -Raw } else { '' }
$targetedText = if (Test-Path -LiteralPath $targetedJsonPath) { Get-Content -LiteralPath $targetedJsonPath -Raw } else { '' }
$expectedBefore = @('evt_before_boundary', 'evt_before_mid', 'evt_before_near')
$expectedAfter = @('evt_after_mid', 'evt_after_near', 'evt_after_boundary')
$expectedSameUser = @('evt_same_user_user_1', 'evt_same_user_user_2', 'evt_same_overlap')
$expectedSameFile = @('evt_same_file_file_1', 'evt_same_file_file_2', 'evt_same_overlap')
$expectedSameDestination = @('evt_same_destination_dest_1', 'evt_same_destination_dest_2', 'evt_same_overlap')
$expectedFactors = @('factor_contains', 'factor_equals', 'factor_exists', 'factor_in')

$arraysOk = $false
$timelineOk = $false
$summaryOk = $false
$jsonConsistencyOk = $false
$sameFileOk = $false
$sameDestinationOk = $false
if ($null -ne $card) {
    $arraysOk = (Test-ArrayExact -Actual @($card.before_context_events) -Expected $expectedBefore) -and (Test-ArrayExact -Actual @($card.after_context_events) -Expected $expectedAfter) -and (Test-ArrayExact -Actual @($card.same_user_events) -Expected $expectedSameUser)
    $sameFileOk = Test-ArrayExact -Actual @($card.same_file_events) -Expected $expectedSameFile
    $sameDestinationOk = Test-ArrayExact -Actual @($card.same_destination_events) -Expected $expectedSameDestination
    $timelineOk = Test-TimelineSortedDedup -Timeline @($card.timeline)
    $summaryOk = ([string]$card.summary).Contains('evt_main') -and ([string]$card.summary).Contains('показано')
    $jsonConsistencyOk = Test-ArrayExact -Actual @($card.suspicious_factors) -Expected $expectedFactors
}
$markdownOk = ($cardMdText -match '^# ') -and ($cardMdText -match '\|---\|---\|---\|') -and ($cardMdText -match 'evt_main')
$dotOk = ($cardDotText -match 'digraph incident') -and ($cardDotText -match '"evt_main"') -and ($cardDotText -match 'same_user|same_file|same_destination')
$targetedTestsOk = ($targetedText -match '"Action":"pass".*"Test":"TestFindAndSearchRelationships"') -and ($targetedText -match '"Action":"pass".*"Test":"TestTimelineSortDedup"')
$malformedLog = Get-CommandLogText -Ctx $ctx -Name 'cli_build_malformed'
$unknownMainLog = Get-CommandLogText -Ctx $ctx -Name 'cli_build_unknown_main'
$limit0Log = Get-CommandLogText -Ctx $ctx -Name 'cli_build_limit0'
$limit1001Log = Get-CommandLogText -Ctx $ctx -Name 'cli_build_limit1001'
$malformedRejected = ($ctx.CommandResults['cli_build_malformed'].exit_code -eq 0) -and ($malformedLog -match 'events_bad\.jsonl:2 malformed JSONL')
$unknownMainRejected = ($ctx.CommandResults['cli_build_unknown_main'].exit_code -eq 0) -and ($unknownMainLog -match 'unknown main event id')
$limitZeroRejected = ($ctx.CommandResults['cli_build_limit0'].exit_code -eq 0) -and ($limit0Log -match 'max_events_per_section must be in \[1,1000\]')
$limit1001Rejected = ($ctx.CommandResults['cli_build_limit1001'].exit_code -eq 0) -and ($limit1001Log -match 'max_events_per_section must be in \[1,1000\]')
$limitsOk = $false
if ($null -ne $cardLimit) {
    $limitsOk = ($cardLimit.before_context.total -eq 3 -and $cardLimit.before_context.shown -eq 2 -and $cardLimit.before_context.truncated) -and ($cardLimit.after_context.total -eq 3 -and $cardLimit.after_context.shown -eq 2 -and $cardLimit.after_context.truncated) -and ($cardLimit.same_user.total -eq 3 -and $cardLimit.same_user.shown -eq 2 -and $cardLimit.same_user.truncated) -and ($cardLimit.same_file.total -eq 3 -and $cardLimit.same_file.shown -eq 2 -and $cardLimit.same_file.truncated) -and ($cardLimit.same_destination.total -eq 3 -and $cardLimit.same_destination.shown -eq 2 -and $cardLimit.same_destination.truncated)
}
$limitMdText = if (Test-Path -LiteralPath $cardLimitMd) { Get-Content -LiteralPath $cardLimitMd -Raw } else { '' }
$limitsOk = $limitsOk -and ($limitMdText -match '2') -and ($limitMdText -match '3')
$precedenceOk = $false
if ($null -ne $cardPrecedence) {
    $precedenceOk = ($cardPrecedence.same_user.shown -eq 2) -and ($cardPrecedence.same_file.shown -eq 0) -and ($cardPrecedence.same_destination.shown -eq 0) -and ($cardPrecedence.max_events_per_section -eq 2)
}
$generatorOk = $false
if ((Test-Path -LiteralPath $genA) -and (Test-Path -LiteralPath $genB)) {
    $genAStats = Get-FileStatsUniqueIDs -Path $genA
    $rawA = Get-Content -LiteralPath $genA -Raw
    $rawB = Get-Content -LiteralPath $genB -Raw
    $scenarioOk = $true
    $lines = @((Get-Content -LiteralPath $genA) | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    foreach ($line in $lines) {
        $obj = $line | ConvertFrom-Json
        [void][DateTime]::ParseExact($obj.timestamp, 'yyyy-MM-ddTHH:mm:ssZ', [System.Globalization.CultureInfo]::InvariantCulture, [System.Globalization.DateTimeStyles]::AdjustToUniversal)
        if ($obj.action -ne 'external_send' -or $obj.destination_type -ne 'external') { $scenarioOk = $false; break }
    }
    $generatorOk = ($genAStats.lines -eq 25) -and ($genAStats.duplicates -eq 0) -and ($rawA -eq $rawB) -and $scenarioOk
}
Save-CheckJson -Path (Join-Path $ctx.OutputsDir 'runtime_validation.json') -Value ([ordered]@{
    arrays_ok = ($arraysOk -and $sameFileOk -and $sameDestinationOk)
    timeline_sorted_dedup = $timelineOk
    summary_dynamic = $summaryOk
    markdown_required_sections = $markdownOk
    dot_required_markers = $dotOk
    targeted_tests_ok = $targetedTestsOk
    malformed_rejected = $malformedRejected
    unknown_main_rejected = $unknownMainRejected
    limit_zero_rejected = $limitZeroRejected
    limit_1001_rejected = $limit1001Rejected
    limits_metadata_ok = $limitsOk
    precedence_ok = $precedenceOk
    generator_ok = $generatorOk
    json_consistency_ok = $jsonConsistencyOk
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
if ($diskPreflightOk) {
    Invoke-CheckCommand -Ctx $ctx -Name 'cli_generate_1m' -Command "& '$tool' generate --count 1000000 --scenario external_send --seed 42 --out '$millionEventsPath'" -TimeoutSec 600
    Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_1m' -Command "& '$tool' build --events '$millionEventsPath' --event-id evt_0500000 --before 30m --after 10m --include-same-user=false --include-same-file=false --include-same-destination=false --include-links=false --max-events-per-section 2 --out '$millionMdPath' --json '$millionCardPath' --dot '$millionDotPath'" -TimeoutSec 600
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
$benchmarkOk = ($benchLog -match 'BenchmarkIncidentCard1000000') -and ($benchLog -match 'ns/op')
$millionOk = $diskPreflightOk -and ($gen1m.exit_code -eq 0) -and (-not [bool]$gen1m.timed_out) -and ($build1m.exit_code -eq 0) -and (-not [bool]$build1m.timed_out) -and ($millionStats.lines -eq 1000000) -and ($millionStats.duplicates -eq 0) -and $millionMemoryOk -and $benchmarkOk
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
    benchmark_ok = $benchmarkOk
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
    generated_a = Test-Path -LiteralPath $genA
    generated_b = Test-Path -LiteralPath $genB
})

Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.jsonl_reader' -Level 'minimum' -Category 'input' -Requirement 'Read JSONL events and reject malformed input with line number' -CommandName 'cli_build_flags' -RequiredArtifacts @($cardFlagsJson, $cardFlagsMd) -ExtraConformant $malformedRejected -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.main_event' -Level 'minimum' -Category 'algorithm' -Requirement 'Find exact main event and reject unknown event_id' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant $unknownMainRejected -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.time_context' -Level 'minimum' -Category 'algorithm' -Requirement 'Before/after windows are inclusive and correctly sorted' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant ($arraysOk -and $sameFileOk -and $sameDestinationOk) -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.markdown_card' -Level 'minimum' -Category 'format' -Requirement 'Markdown incident card has required headings/table and dynamic summary' -CommandName 'cli_build_request' -RequiredArtifacts @($cardMd) -ExtraConformant $markdownOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.search_sort_tests' -Level 'minimum' -Category 'tests' -Requirement 'Targeted Go tests prove Find/Search and Timeline Sort/Dedup' -CommandName 'targeted_go_test' -RequiredArtifacts @($targetedJsonPath) -ExtraConformant $targetedTestsOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.same_file' -Level 'good' -Category 'algorithm' -Requirement 'Related events by same file are correct' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant $sameFileOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.same_destination' -Level 'good' -Category 'algorithm' -Requirement 'Related events by same destination are correct' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant $sameDestinationOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.json_card' -Level 'good' -Category 'format' -Requirement 'JSON card is consistent with arrays and timeline' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant ($jsonConsistencyOk -and $timelineOk) -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.suspicious_factors' -Level 'good' -Category 'algorithm' -Requirement 'Suspicious factors from YAML operators equals/in/contains/exists' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson) -ExtraConformant $jsonConsistencyOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.generator' -Level 'good' -Category 'cli' -Requirement 'Generator produces 25 valid unique deterministic external_send events' -CommandName 'cli_generate_25_a' -RequiredArtifacts @($genA, $genB) -ExtraConformant $generatorOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.dot_graph' -Level 'excellent' -Category 'format' -Requirement 'DOT graph has main node, relations and escaping' -CommandName 'cli_build_request' -RequiredArtifacts @($cardDot) -ExtraConformant $dotOk -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.section_limits' -Level 'excellent' -Category 'report' -Requirement 'Section limit metadata/truncation and invalid limit rejection are correct' -CommandName 'cli_build_limit2' -RequiredArtifacts @($cardLimitJson, $cardLimitMd) -ExtraConformant ($limitsOk -and $limitZeroRejected -and $limit1001Rejected -and $precedenceOk) -ExtraEvidence @('outputs/runtime_validation.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.million_benchmark' -Level 'excellent' -Category 'performance' -Requirement 'Real 1M run and benchmark pass with memory/time constraints' -CommandName 'cli_build_1m' -RequiredArtifacts @((Join-Path $ctx.OutputsDir 'million_metrics.json')) -ExtraConformant $millionOk -ExtraEvidence @('outputs/million_metrics.json')
Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.polished_report' -Level 'excellent' -Category 'report' -Requirement 'Polished markdown+json report is sorted, deduped and consistent' -CommandName 'cli_build_request' -RequiredArtifacts @($cardMd, $cardJson) -ExtraConformant ($timelineOk -and $summaryOk -and $markdownOk -and $jsonConsistencyOk) -ExtraEvidence @('outputs/runtime_validation.json')

$notes.cleanup_ok = $cleanupOk
$notes.expected_score = [ordered]@{ minimum = 5; good = 5; excellent = 4; engineering = 12; total = 26 }
$notes.runtime_validation = 'outputs/runtime_validation.json'
$notes.million_metrics = 'outputs/million_metrics.json'

Complete-Check -Ctx $ctx -Extra $notes


