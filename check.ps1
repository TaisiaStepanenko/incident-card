param(
    [string]$RepoRoot = (Get-Location).Path,
    [string]$OutRoot = ''
)


# Embedded common helpers. This file is standalone and can be run from the repository root.

Set-StrictMode -Version 2.0

function Get-CheckGoCommand {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) {
        return $go.Source
    }

    throw 'go executable was not found in PATH. Install Go and make sure go is available in PATH.'
}

function New-CheckContext {
    param(
        [Parameter(Mandatory=$true)][string]$Student,
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [string]$OutRoot = ''
    )

    $repo = (Resolve-Path -LiteralPath $RepoRoot).Path
    if ($OutRoot -eq '') {
        $OutRoot = Join-Path $repo '.check-results'
    }

    $timestamp = Get-Date -Format 'yyyyMMdd_HHmmss'
    $safeStudent = $Student -replace '[^A-Za-z0-9_.-]', '_'
    $resultDir = Join-Path $OutRoot "${safeStudent}_${timestamp}"
    $logsDir = Join-Path $resultDir 'logs'
    $inputsDir = Join-Path $resultDir 'inputs'
    $outputsDir = Join-Path $resultDir 'outputs'
    $metaDir = Join-Path $resultDir 'meta'
    $tmpDir = Join-Path $resultDir 'tmp'

    foreach ($dir in @($resultDir, $logsDir, $inputsDir, $outputsDir, $metaDir, $tmpDir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
    }

    $ctx = [ordered]@{
        Student = $Student
        RepoRoot = $repo
        ResultDir = $resultDir
        LogsDir = $logsDir
        InputsDir = $inputsDir
        OutputsDir = $outputsDir
        MetaDir = $metaDir
        TmpDir = $tmpDir
        CommandsPath = Join-Path $resultDir 'commands.jsonl'
        GoCmd = Get-CheckGoCommand
        StartedAt = (Get-Date).ToString('o')
        CommandResults = @{}
        Assessments = New-Object System.Collections.ArrayList
    }

    '' | Set-Content -LiteralPath $ctx.CommandsPath -Encoding UTF8
    return $ctx
}

function Write-CheckText {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$RelativePath,
        [Parameter(Mandatory=$true)][string]$Content
    )

    $path = Join-Path $Ctx.ResultDir $RelativePath
    $parent = Split-Path -Parent $path
    if ($parent) {
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
    }
    Set-Content -LiteralPath $path -Value $Content -Encoding UTF8
    return $path
}

function Save-CheckJson {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)]$Value
    )

    $json = $Value | ConvertTo-Json -Depth 30
    Set-Content -LiteralPath $Path -Value $json -Encoding UTF8
}

function Invoke-CheckCommand {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Name,
        [Parameter(Mandatory=$true)][string]$Command,
        [string]$WorkingDirectory = ''
    )

    if ($WorkingDirectory -eq '') {
        $WorkingDirectory = $Ctx.RepoRoot
    }

    $safeName = $Name -replace '[^A-Za-z0-9_.-]', '_'
    $logPath = Join-Path $Ctx.LogsDir "$safeName.log"
    $runnerPath = Join-Path $Ctx.TmpDir "$safeName.ps1"
    $started = Get-Date

    $runner = @"
`$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath '$($WorkingDirectory.Replace("'", "''"))'
try {
    `$global:LASTEXITCODE = `$null
    `$Error.Clear()
    $Command
    `$success = `$?
    `$exitCode = `$global:LASTEXITCODE
    if (`$null -eq `$exitCode) {
        if (`$success -and `$Error.Count -eq 0) {
            `$exitCode = 0
        } else {
            `$exitCode = 1
        }
    }
    exit `$exitCode
} catch {
    Write-Error `$_
    exit 1
}
"@

    Set-Content -LiteralPath $runnerPath -Value $runner -Encoding UTF8

    $output = & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $runnerPath 2>&1
    $exitCode = $LASTEXITCODE
    $ended = Get-Date

    @(
        "name: $Name"
        "working_directory: $WorkingDirectory"
        "command:"
        $Command
        "exit_code: $exitCode"
        "started_at: $($started.ToString('o'))"
        "ended_at: $($ended.ToString('o'))"
        ""
        "output:"
        ($output | Out-String)
    ) | Set-Content -LiteralPath $logPath -Encoding UTF8

    $record = [ordered]@{
        name = $Name
        command = $Command
        working_directory = $WorkingDirectory
        exit_code = $exitCode
        started_at = $started.ToString('o')
        ended_at = $ended.ToString('o')
        duration_ms = [int](($ended - $started).TotalMilliseconds)
        log = "logs/$safeName.log"
    }
    ($record | ConvertTo-Json -Compress) | Add-Content -LiteralPath $Ctx.CommandsPath -Encoding UTF8
    $Ctx.CommandResults[$Name] = $record
    $script:LAST_CHECK_EXIT_CODE = $exitCode
}

function Add-FeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][ValidateSet('not_implemented','partial','full')][string]$Implementation,
        [Parameter(Mandatory=$true)][ValidateSet('not_tested','nonconformant','conformant')][string]$Conformance,
        [string[]]$Evidence = @(),
        [string]$Details = ''
    )

    $item = [ordered]@{
        id = $Id
        level = $Level
        category = $Category
        requirement = $Requirement
        implementation = $Implementation
        conformance = $Conformance
        evidence = @($Evidence)
        details = $Details
    }
    $Ctx.Assessments.Add($item) | Out-Null
}

function Add-CommandFeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][string]$CommandName,
        [string[]]$RequiredArtifacts = @(),
        [string]$Details = ''
    )

    $hasCommand = $Ctx.CommandResults.ContainsKey($CommandName)
    $exitCode = if ($hasCommand) { [int]$Ctx.CommandResults[$CommandName].exit_code } else { -999 }
    $missingArtifacts = @($RequiredArtifacts | Where-Object { -not (Test-Path -LiteralPath $_) })
    $implementation = 'not_implemented'
    $conformance = 'not_tested'

    if ($hasCommand) {
        $implementation = if ($exitCode -eq 0 -and $missingArtifacts.Count -eq 0) { 'full' } else { 'partial' }
        $conformance = if ($exitCode -eq 0 -and $missingArtifacts.Count -eq 0) { 'conformant' } else { 'nonconformant' }
    }

    $evidence = @()
    if ($hasCommand) {
        $evidence += [string]$Ctx.CommandResults[$CommandName].log
    }
    foreach ($artifact in $RequiredArtifacts) {
        if (Test-Path -LiteralPath $artifact) {
            $evidence += $artifact.Replace($Ctx.ResultDir, '').TrimStart('\')
        }
    }

    $detailParts = @()
    if ($Details) {
        $detailParts += $Details
    }
    if ($hasCommand) {
        $detailParts += "exit_code=$exitCode"
    } else {
        $detailParts += 'command was not executed'
    }
    if ($missingArtifacts.Count -gt 0) {
        $detailParts += "missing artifacts: $($missingArtifacts -join ', ')"
    }

    Add-FeatureAssessment -Ctx $Ctx -Id $Id -Level $Level -Category $Category -Requirement $Requirement -Implementation $implementation -Conformance $conformance -Evidence $evidence -Details ($detailParts -join '; ')
}

function Add-BooleanFeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][bool]$Implemented,
        [Parameter(Mandatory=$true)][bool]$Conformant,
        [string[]]$Evidence = @(),
        [string]$Details = ''
    )

    $implementation = if ($Implemented) { 'full' } else { 'not_implemented' }
    $conformance = if (-not $Implemented) { 'not_tested' } elseif ($Conformant) { 'conformant' } else { 'nonconformant' }
    Add-FeatureAssessment -Ctx $Ctx -Id $Id -Level $Level -Category $Category -Requirement $Requirement -Implementation $implementation -Conformance $conformance -Evidence $Evidence -Details $Details
}

function Add-SourceFeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][string[]]$Patterns,
        [ValidateSet('any','all')][string]$Match = 'all',
        [string]$Details = ''
    )

    $files = @(Get-ChildItem -LiteralPath $Ctx.RepoRoot -Recurse -File -ErrorAction SilentlyContinue | Where-Object {
        $_.FullName -notlike '*\.check-results\*' -and ($_.Extension -in @('.go', '.md') -or $_.Name -eq 'Makefile')
    })
    $matchedPatterns = @()
    $evidence = @()
    foreach ($pattern in $Patterns) {
        $hits = @($files | Select-String -Pattern $pattern -ErrorAction SilentlyContinue)
        if ($hits.Count -gt 0) {
            $matchedPatterns += $pattern
            $evidence += @($hits | Select-Object -First 5 | ForEach-Object {
                "$($_.Path):$($_.LineNumber)"
            })
        }
    }

    $implemented = if ($Match -eq 'all') {
        $matchedPatterns.Count -eq $Patterns.Count
    } else {
        $matchedPatterns.Count -gt 0
    }
    $implementation = if ($implemented) { 'partial' } else { 'not_implemented' }
    $detailText = "source-only check; matched=$($matchedPatterns.Count)/$($Patterns.Count)"
    if ($Details) {
        $detailText = "$Details; $detailText"
    }
    Add-FeatureAssessment -Ctx $Ctx -Id $Id -Level $Level -Category $Category -Requirement $Requirement -Implementation $implementation -Conformance 'not_tested' -Evidence ($evidence | Select-Object -Unique) -Details $detailText
}

function Add-StandardEngineeringAssessments {
    param(
        [Parameter(Mandatory=$true)]$Ctx
    )

    $testFiles = @(Get-ChildItem -LiteralPath $Ctx.RepoRoot -Recurse -File -Filter '*_test.go' -ErrorAction SilentlyContinue | Where-Object { $_.FullName -notlike '*\.check-results\*' })
    $testFunctions = @($testFiles | Select-String -Pattern '^\s*func\s+Test[A-Za-z0-9_]+\s*\(' -ErrorAction SilentlyContinue)
    $benchmarkFunctions = @($testFiles | Select-String -Pattern '^\s*func\s+Benchmark[A-Za-z0-9_]+\s*\(' -ErrorAction SilentlyContinue)

    $testFileEvidence = @($testFiles | ForEach-Object { $_.FullName })
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.unit_tests_present' -Level 'engineering' -Category 'tests' -Requirement 'Go unit tests are present' -Implemented ($testFunctions.Count -gt 0) -Conformant ($testFunctions.Count -gt 0) -Evidence $testFileEvidence -Details "test_files=$($testFiles.Count); test_functions=$($testFunctions.Count)"
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.benchmarks_present' -Level 'engineering' -Category 'benchmarks' -Requirement 'Go benchmark tests are present' -Implemented ($benchmarkFunctions.Count -gt 0) -Conformant ($benchmarkFunctions.Count -gt 0) -Evidence $testFileEvidence -Details "benchmark_functions=$($benchmarkFunctions.Count)"

    if ($Ctx.CommandResults.ContainsKey('go_test_all')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.go_test_passes' -Level 'engineering' -Category 'tests' -Requirement 'go test ./... passes' -CommandName 'go_test_all'
    }
    if ($Ctx.CommandResults.ContainsKey('go_test_race')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.race_test_passes' -Level 'engineering' -Category 'tests' -Requirement 'go test -race ./... passes' -CommandName 'go_test_race'
    }
    if ($Ctx.CommandResults.ContainsKey('make_test')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.make_test_runs' -Level 'engineering' -Category 'reproducibility' -Requirement 'make test passes' -CommandName 'make_test'
    }
    if ($Ctx.CommandResults.ContainsKey('make_bench')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.make_bench_runs' -Level 'engineering' -Category 'reproducibility' -Requirement 'make bench passes' -CommandName 'make_bench'
    }
    if ($Ctx.CommandResults.ContainsKey('make_demo')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.make_demo_runs' -Level 'engineering' -Category 'reproducibility' -Requirement 'make demo passes' -CommandName 'make_demo'
    }

    $readmePath = Join-Path $Ctx.RepoRoot 'README.md'
    $readmeOk = (Test-Path -LiteralPath $readmePath) -and ((Get-Item -LiteralPath $readmePath).Length -gt 100)
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.readme' -Level 'engineering' -Category 'documentation' -Requirement 'README.md exists and is not empty' -Implemented $readmeOk -Conformant $readmeOk -Evidence @('repo_snapshot/README.md')

    $makefilePath = Join-Path $Ctx.RepoRoot 'Makefile'
    $makefileText = if (Test-Path -LiteralPath $makefilePath) { Get-Content -LiteralPath $makefilePath -Raw } else { '' }
    foreach ($target in @('test','bench','demo')) {
        $targetOk = $makefileText -match "(?m)^\s*${target}\s*:"
        Add-BooleanFeatureAssessment -Ctx $Ctx -Id "engineering.make_$target" -Level 'engineering' -Category 'reproducibility' -Requirement "Makefile has target $target" -Implemented $targetOk -Conformant $targetOk -Evidence @('repo_snapshot/Makefile')
    }

    $controlPath = Join-Path $Ctx.RepoRoot 'testdata\control'
    $controlFiles = @()
    if (Test-Path -LiteralPath $controlPath) {
        $controlFiles = @(Get-ChildItem -LiteralPath $controlPath -Recurse -File -ErrorAction SilentlyContinue)
    }
    $controlEvidence = @($controlFiles | ForEach-Object { $_.FullName })
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.control_data' -Level 'engineering' -Category 'reproducibility' -Requirement 'Fixed testdata/control set exists' -Implemented ($controlFiles.Count -gt 0) -Conformant ($controlFiles.Count -gt 0) -Evidence $controlEvidence -Details "files=$($controlFiles.Count)"

    $solutionPath = Join-Path $Ctx.RepoRoot 'docs\reshenie.md'
    $solutionOk = (Test-Path -LiteralPath $solutionPath) -and ((Get-Item -LiteralPath $solutionPath).Length -gt 100)
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.solution_doc' -Level 'engineering' -Category 'documentation' -Requirement 'Non-empty docs/reshenie.md exists' -Implemented $solutionOk -Conformant $solutionOk -Evidence @('repo_snapshot/docs/reshenie.md')
}

function Copy-CheckPath {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Source,
        [Parameter(Mandatory=$true)][string]$RelativeDestination
    )

    if (-not (Test-Path -LiteralPath $Source)) {
        return
    }

    $destination = Join-Path $Ctx.ResultDir $RelativeDestination
    $parent = Split-Path -Parent $destination
    if ($parent) {
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
    }
    Copy-Item -LiteralPath $Source -Destination $destination -Recurse -Force
}

function Complete-Check {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [hashtable]$Extra = @{}
    )

    Add-StandardEngineeringAssessments -Ctx $Ctx

    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_git_head' -Command "git rev-parse HEAD | Set-Content -LiteralPath '$($Ctx.MetaDir)\git_head.txt' -Encoding UTF8" | Out-Null
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_git_status' -Command "git status --short | Set-Content -LiteralPath '$($Ctx.MetaDir)\git_status_short.txt' -Encoding UTF8" | Out-Null
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_go_version' -Command "& '$($Ctx.GoCmd)' version | Set-Content -LiteralPath '$($Ctx.MetaDir)\go_version.txt' -Encoding UTF8" | Out-Null
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_go_env' -Command "& '$($Ctx.GoCmd)' env GOVERSION GOOS GOARCH | Set-Content -LiteralPath '$($Ctx.MetaDir)\go_env.txt' -Encoding UTF8" | Out-Null

    foreach ($name in @('README.md', 'Makefile', 'go.mod', 'docs')) {
        $path = Join-Path $Ctx.RepoRoot $name
        Copy-CheckPath -Ctx $Ctx -Source $path -RelativeDestination "repo_snapshot/$name"
    }

    $assessmentItems = @($Ctx.Assessments)
    $assessmentSummary = [ordered]@{}
    foreach ($level in @('minimum','good','excellent','engineering')) {
        $items = @($assessmentItems | Where-Object { $_.level -eq $level })
        $assessmentSummary[$level] = [ordered]@{
            total = $items.Count
            full = @($items | Where-Object { $_.implementation -eq 'full' }).Count
            partial = @($items | Where-Object { $_.implementation -eq 'partial' }).Count
            not_implemented = @($items | Where-Object { $_.implementation -eq 'not_implemented' }).Count
            conformant = @($items | Where-Object { $_.conformance -eq 'conformant' }).Count
            nonconformant = @($items | Where-Object { $_.conformance -eq 'nonconformant' }).Count
            not_tested = @($items | Where-Object { $_.conformance -eq 'not_tested' }).Count
        }
    }
    Save-CheckJson -Path (Join-Path $Ctx.ResultDir 'assessment.json') -Value ([ordered]@{
        schema_version = 1
        statuses = [ordered]@{
            implementation = @('not_implemented','partial','full')
            conformance = @('not_tested','nonconformant','conformant')
        }
        summary = $assessmentSummary
        features = $assessmentItems
    })

    $manifest = [ordered]@{
        student = $Ctx.Student
        repo_root = $Ctx.RepoRoot
        started_at = $Ctx.StartedAt
        completed_at = (Get-Date).ToString('o')
        machine = [ordered]@{
            computer_name = $env:COMPUTERNAME
            user_name = $env:USERNAME
            os = (Get-CimInstance Win32_OperatingSystem).Caption
            powershell = $PSVersionTable.PSVersion.ToString()
        }
        result_dir = $Ctx.ResultDir
        commands_file = 'commands.jsonl'
        assessment_file = 'assessment.json'
        notes = $Extra
    }
    Save-CheckJson -Path (Join-Path $Ctx.ResultDir 'manifest.json') -Value $manifest

    $zipPath = "$($Ctx.ResultDir).zip"
    if (Test-Path -LiteralPath $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }
    Compress-Archive -Path (Join-Path $Ctx.ResultDir '*') -DestinationPath $zipPath -Force

    Write-Host "CHECK_RESULT_DIR=$($Ctx.ResultDir)"
    Write-Host "CHECK_RESULT_ZIP=$zipPath"
    return $zipPath
}


$ctx = New-CheckContext -Student 'incident_card_check' -RepoRoot $RepoRoot -OutRoot $OutRoot

$eventsPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/events.jsonl' -Content @'
{"event_id":"evt_001","timestamp":"2026-06-16T10:00:00Z","user_id":"user_017","machine_id":"pc_003","department":"sales","action":"open_file","channel":"local","file_id":"file_778","file_name":"client_base.xlsx","file_ext":"xlsx","content_classes":["client_data","personal_data"],"destination_type":"none","size_bytes":204800,"severity":"low"}
{"event_id":"evt_002","timestamp":"2026-06-16T10:10:00Z","user_id":"user_017","machine_id":"pc_003","department":"sales","action":"create_archive","channel":"local","file_id":"file_779","file_name":"client_base.zip","file_ext":"zip","content_classes":["client_data"],"destination_type":"none","size_bytes":409600,"severity":"medium"}
{"event_id":"evt_003","timestamp":"2026-06-16T10:15:00Z","user_id":"user_017","machine_id":"pc_003","department":"sales","action":"email_send","channel":"email","file_id":"file_778","file_name":"client_base.xlsx","file_ext":"xlsx","content_classes":["client_data","personal_data"],"destination_id":"dst_009","destination_type":"external","destination":"external_email_001","size_bytes":204800,"severity":"high"}
{"event_id":"evt_004","timestamp":"2026-06-16T10:25:00Z","user_id":"user_017","machine_id":"pc_003","department":"sales","action":"delete_file","channel":"local","file_id":"file_778","file_name":"client_base.xlsx","file_ext":"xlsx","content_classes":["client_data"],"destination_type":"none","size_bytes":204800,"severity":"high"}
'@

$requestPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/request.json' -Content @'
{
  "incident_id": "inc_001",
  "main_event_id": "evt_003",
  "window_before": "30m",
  "window_after": "10m",
  "include_same_user": true,
  "include_same_file": true,
  "include_same_destination": true,
  "max_events_per_section": 50
}
'@

$factorsPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/factors.yaml' -Content @'
factors:
  - factor_id: external_destination
    title: External destination
    condition:
      field: destination_type
      equals: external
  - factor_id: client_data
    title: Client data
    condition:
      field: content_classes
      contains: client_data
'@

Invoke-CheckCommand -Ctx $ctx -Name 'go_test_all' -Command "& '$($ctx.GoCmd)' test ./..."

if (Test-Path -LiteralPath (Join-Path $ctx.RepoRoot 'Makefile')) {
    Invoke-CheckCommand -Ctx $ctx -Name 'make_test' -Command 'make test'
    Invoke-CheckCommand -Ctx $ctx -Name 'make_bench' -Command 'make bench'
    Invoke-CheckCommand -Ctx $ctx -Name 'make_demo' -Command 'make demo'
}

$tool = Join-Path $ctx.OutputsDir 'incident-card.exe'
Invoke-CheckCommand -Ctx $ctx -Name 'build_cli' -Command "& '$($ctx.GoCmd)' build -o '$tool' ./cmd/incident-card"

$cardMd = Join-Path $ctx.OutputsDir 'card.md'
$cardJson = Join-Path $ctx.OutputsDir 'card.json'
$dot = Join-Path $ctx.OutputsDir 'graph.dot'
$cardFlagsMd = Join-Path $ctx.OutputsDir 'card_flags.md'
$cardFlagsJson = Join-Path $ctx.OutputsDir 'card_flags.json'
$generated = Join-Path $ctx.OutputsDir 'generated_events.jsonl'

Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_request' -Command "& '$tool' build --events '$eventsPath' --request '$requestPath' --factors '$factorsPath' --out '$cardMd' --json '$cardJson' --dot '$dot'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_build_flags' -Command "& '$tool' build --events '$eventsPath' --event-id evt_003 --before 30m --after 10m --out '$cardFlagsMd' --json '$cardFlagsJson'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_generate' -Command "& '$tool' generate --count 25 --scenario external_send --out '$generated' --seed 42"

$expectedArtifacts = [ordered]@{
    card_md = Test-Path -LiteralPath $cardMd
    card_json = Test-Path -LiteralPath $cardJson
    graph_dot = Test-Path -LiteralPath $dot
    card_flags_md = Test-Path -LiteralPath $cardFlagsMd
    card_flags_json = Test-Path -LiteralPath $cardFlagsJson
    generated_events = Test-Path -LiteralPath $generated
}
Save-CheckJson -Path (Join-Path $ctx.OutputsDir 'artifact_presence.json') -Value $expectedArtifacts

Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.jsonl_reader' -Level 'minimum' -Category 'input' -Requirement 'Read JSONL events' -CommandName 'cli_build_flags' -RequiredArtifacts @($cardFlagsMd)
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.main_event' -Level 'minimum' -Category 'algorithm' -Requirement 'Find main event by event_id' -CommandName 'cli_build_flags' -RequiredArtifacts @($cardFlagsMd)
Add-SourceFeatureAssessment -Ctx $ctx -Id 'minimum.time_context' -Level 'minimum' -Category 'algorithm' -Requirement 'Before and after time context' -Patterns @('Before|before','After|after','time\.Parse|ParseDuration') -Match 'all'
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.markdown_card' -Level 'minimum' -Category 'format' -Requirement 'Markdown incident card' -CommandName 'cli_build_flags' -RequiredArtifacts @($cardFlagsMd)
Add-SourceFeatureAssessment -Ctx $ctx -Id 'minimum.search_sort_tests' -Level 'minimum' -Category 'tests' -Requirement 'Event search and timeline sorting tests' -Patterns @('Test.*Find|Test.*Search|Test.*Event','Test.*Sort|Test.*Timeline') -Match 'all'

Add-SourceFeatureAssessment -Ctx $ctx -Id 'good.same_file' -Level 'good' -Category 'algorithm' -Requirement 'Related events by same file' -Patterns @('SameFile|same_file|file_id') -Match 'any'
Add-SourceFeatureAssessment -Ctx $ctx -Id 'good.same_destination' -Level 'good' -Category 'algorithm' -Requirement 'Related events by same destination' -Patterns @('SameDestination|same_destination|destination_id') -Match 'any'
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.json_card' -Level 'good' -Category 'format' -Requirement 'JSON incident card' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson)
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.suspicious_factors' -Level 'good' -Category 'algorithm' -Requirement 'Suspicious factors from YAML' -CommandName 'cli_build_request' -RequiredArtifacts @($cardJson)
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.generator' -Level 'good' -Category 'cli' -Requirement 'Event scenario generator' -CommandName 'cli_generate' -RequiredArtifacts @($generated)

Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.dot_graph' -Level 'excellent' -Category 'format' -Requirement 'DOT relation graph' -CommandName 'cli_build_request' -RequiredArtifacts @($dot)
Add-SourceFeatureAssessment -Ctx $ctx -Id 'excellent.section_limits' -Level 'excellent' -Category 'report' -Requirement 'max_events_per_section limit and truncation marker' -Patterns @('max_events_per_section|MaxEventsPerSection','truncat') -Match 'all'
Add-SourceFeatureAssessment -Ctx $ctx -Id 'excellent.million_benchmark' -Level 'excellent' -Category 'performance' -Requirement 'Benchmark for 1000000 events' -Patterns @('Benchmark','1000000') -Match 'all'
Add-SourceFeatureAssessment -Ctx $ctx -Id 'excellent.polished_report' -Level 'excellent' -Category 'report' -Requirement 'Polished report with tables and summary' -Patterns @('Timeline|\u0412\u0440\u0435\u043c\u0435\u043d\u043d\u0430\u044f \u0448\u043a\u0430\u043b\u0430','Summary|\u041a\u0440\u0430\u0442\u043a\u043e\u0435 \u0440\u0435\u0437\u044e\u043c\u0435|\u0420\u0435\u0437\u044e\u043c\u0435','\|.*\|') -Match 'all'

Complete-Check -Ctx $ctx -Extra @{
    expected_cli = 'incident-card generate/build'
    expected_outputs = @('card.md', 'card.json', 'graph.dot', 'card_flags.md', 'card_flags.json')
}


