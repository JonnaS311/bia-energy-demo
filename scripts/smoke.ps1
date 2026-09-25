# Smoke test del flujo principal contra una instancia levantada (spec de pruebas RF-T-12).
# Uso: pwsh -File scripts/smoke.ps1  (o powershell -File ...). Variables: API_URL, WEB_URL.
$ErrorActionPreference = 'Stop'
$api = if ($env:API_URL) { $env:API_URL } else { 'http://localhost:8080' }
$web = if ($env:WEB_URL) { $env:WEB_URL } else { 'http://localhost:5173' }
$step = 0
function Fail($msg) { Write-Host "FAIL step ${script:step}: $msg"; exit 1 }
function Step($name, [scriptblock]$body) {
  $script:step++
  try { & $body; Write-Host "ok  $($script:step). $name" } catch { Fail $_.Exception.Message }
}
$start = Get-Date

Step 'GET /health' {
  $h = Invoke-RestMethod "$api/health"
  if ($h.db -ne 'ok' -or $h.ingest.status -ne 'COMPLETED' -or $h.ingest.readings -ne 4032) { throw "health $($h | ConvertTo-Json -Compress)" }
}
Step 'POST /auth/login' {
  $body = @{ email = 'analista@energy.local'; password = 'Demo1234!' } | ConvertTo-Json
  $r = Invoke-RestMethod "$api/auth/login" -Method Post -ContentType 'application/json' -Body $body
  if (-not $r.token) { throw 'empty token' }
  $script:auth = @{ Authorization = "Bearer $($r.token)" }
}
Step 'GET /meters' {
  $m = Invoke-RestMethod "$api/meters" -Headers $auth
  if ($m.total -ne 12) { throw "total $($m.total)" }
}
Step 'POST /ai/analyze' {
  try {
    $r = Invoke-RestMethod "$api/ai/analyze" -Method Post -Headers $auth
    $script:aid = $r.analysis_id
  } catch {
    $resp = $_.ErrorDetails.Message | ConvertFrom-Json
    if ($resp.error.code -ne 'ANALYSIS_IN_PROGRESS') { throw $_ }
    $script:aid = $resp.error.details.analysis_id
  }
  if (-not $aid) { throw 'no analysis id' }
}
Step 'GET /ai/analysis/:id hasta COMPLETED' {
  $deadline = (Get-Date).AddSeconds(90)
  do {
    Start-Sleep -Milliseconds 1000
    $a = Invoke-RestMethod "$api/ai/analysis/$aid" -Headers $auth
  } while ($a.status -in @('QUEUED', 'RUNNING') -and (Get-Date) -lt $deadline)
  if ($a.status -ne 'COMPLETED' -or $a.summary.anomalies_detected -ne 4 -or $a.summary.high_priority -ne 2) { throw "analysis $($a | ConvertTo-Json -Compress -Depth 5)" }
}
Step 'GET /anomalies orden' {
  $an = Invoke-RestMethod "$api/anomalies" -Headers $auth
  $order = ($an.items | Select-Object -First 4 | ForEach-Object { $_.meter_id }) -join ','
  if ($order -ne 'M-109,M-112,M-104,M-106') { throw "order $order" }
}
Step 'GET /meters/M-109' {
  $m = Invoke-RestMethod "$api/meters/M-109" -Headers $auth
  if ($m.status -ne 'CRITICAL' -or $m.variation_pct -lt 110.4 -or $m.variation_pct -gt 110.6) { throw "M-109 $($m.status) $($m.variation_pct)" }
}
Step 'GET /dashboard/summary' {
  $d = Invoke-RestMethod "$api/dashboard/summary" -Headers $auth
  if ($d.anomalies_total -ne 4 -or $d.high_priority_total -ne 2 -or $d.ai_confidence_avg -ne 0.93) { throw "dashboard $($d | ConvertTo-Json -Compress -Depth 3)" }
}
Step 'GET web /login' {
  $w = Invoke-WebRequest "$web/login" -UseBasicParsing
  if ($w.StatusCode -ne 200 -or $w.Content -notmatch '<div id="root">') { throw "web $($w.StatusCode)" }
}
Write-Host ("SMOKE OK (9/9) en {0:N1} s" -f ((Get-Date) - $start).TotalSeconds)
