
$ErrorActionPreference = "Stop"

$file = ".\services\api\cmd\server\main.go"
$path = (Resolve-Path -LiteralPath $file).Path
$content = [System.IO.File]::ReadAllText($path)

$startMarker = "dbConfig, err := database.LoadConfig()"
$endMarker = "redisConfig, err := queue.LoadConfig()"

$start = $content.IndexOf($startMarker)
$end = $content.IndexOf($endMarker)

if ($start -lt 0 -or $end -le $start) {
    throw "Database startup boundaries not found."
}

if ($content.Contains("database.LoadMigratorConfig()")) {
    throw "Migration configuration already exists."
}

$backup = ".\backups\main-db-pools-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
New-Item -ItemType Directory -Path $backup -Force | Out-Null
Copy-Item -LiteralPath $file -Destination (Join-Path $backup "main.go")

$newBlock = @'
migrationConfig, err := database.LoadMigratorConfig()
        if err != nil {
                log.Fatalf("migration database configuration error: %v", err)
        }

        migrationPool, err := database.Connect(startupCtx, migrationConfig)
        if err != nil {
                log.Fatalf("migration database connection failed: %v", err)
        }

        if err := database.RunMigrations(startupCtx, migrationPool); err != nil {
                migrationPool.Close()
                log.Fatalf("database migration failed: %v", err)
        }
        migrationPool.Close()

        dbConfig, err := database.LoadRuntimeConfig()
        if err != nil {
                log.Fatalf("runtime database configuration error: %v", err)
        }

        pool, err := database.Connect(startupCtx, dbConfig)
        if err != nil {
                log.Fatalf("runtime database connection failed: %v", err)
        }
        defer pool.Close()

'@

$content = $content.Substring(0, $start) + $newBlock + $content.Substring($end)

[System.IO.File]::WriteAllText(
    $path,
    $content,
    [System.Text.UTF8Encoding]::new($false)
)

Write-Host "PATCH WRITTEN. Backup: $backup" -ForegroundColor Green

Push-Location ".\services\api"
try {
    gofmt -w ".\cmd\server\main.go"
    if ($LASTEXITCODE -ne 0) { throw "gofmt failed" }

    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "Go tests failed" }

    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed" }

    Write-Host "API DATABASE POOL PATCH: PASS" -ForegroundColor Green
}
finally {
    Pop-Location
}
