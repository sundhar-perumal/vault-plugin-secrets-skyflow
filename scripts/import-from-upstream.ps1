<#
.SYNOPSIS
    Import and transform code from internal repository to public repository.

.DESCRIPTION
    This script:
    1. Copies source files from the internal (private) repository
    2. Transforms goerr.New() calls to fmt.Errorf()
    3. Updates go.mod with public module path
    4. Removes private dependencies

.NOTES
    Run from the root of the public repository:
    .\scripts\import-from-upstream.ps1
#>

# ============================================================================
# CONFIGURATION - Update these paths for your environment
# ============================================================================

$INTERNAL_REPO = "D:\aone\github.com\angelone\go-skyflow-harshicorp-plugin"
$INTERNAL_MODULE = "github.com/angel-one/go-skyflow-harshicorp-plugin"
$PUBLIC_MODULE = "github.com/sundhar-perumal/vault-plugin-secrets-skyflow"

# ============================================================================
# SCRIPT START
# ============================================================================

$ErrorActionPreference = "Stop"
$PUBLIC_REPO = $PSScriptRoot | Split-Path -Parent

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Skyflow Vault Plugin - Import Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Internal Repo: $INTERNAL_REPO" -ForegroundColor Yellow
Write-Host "Public Repo:   $PUBLIC_REPO" -ForegroundColor Yellow
Write-Host ""

# Verify internal repo exists
if (-not (Test-Path $INTERNAL_REPO)) {
    Write-Host "ERROR: Internal repository not found at $INTERNAL_REPO" -ForegroundColor Red
    exit 1
}

# ============================================================================
# STEP 1: Copy files from internal repo
# ============================================================================

Write-Host "Step 1: Copying files from internal repository..." -ForegroundColor Green

# Directories to copy
$dirsToCopy = @("backend", "cmd", "docs", "examples", "test", ".github")

# Files to copy from root
$filesToCopy = @("go.mod", "go.sum", "Makefile", "Dockerfile", ".golangci.yml", ".gitignore")

# Clean existing directories (except scripts, LICENSE, and docs we want to keep)
foreach ($dir in $dirsToCopy) {
    $targetDir = Join-Path $PUBLIC_REPO $dir
    if (Test-Path $targetDir) {
        Remove-Item -Recurse -Force $targetDir
        Write-Host "  Cleaned: $dir" -ForegroundColor Gray
    }
}

# Copy directories
foreach ($dir in $dirsToCopy) {
    $sourceDir = Join-Path $INTERNAL_REPO $dir
    $targetDir = Join-Path $PUBLIC_REPO $dir
    if (Test-Path $sourceDir) {
        Copy-Item -Recurse -Force $sourceDir $targetDir
        Write-Host "  Copied: $dir" -ForegroundColor Gray
    }
}

# Copy root files
foreach ($file in $filesToCopy) {
    $sourceFile = Join-Path $INTERNAL_REPO $file
    $targetFile = Join-Path $PUBLIC_REPO $file
    if (Test-Path $sourceFile) {
        Copy-Item -Force $sourceFile $targetFile
        Write-Host "  Copied: $file" -ForegroundColor Gray
    }
}

Write-Host "  Done!" -ForegroundColor Green
Write-Host ""

# ============================================================================
# STEP 2: Transform goerr to fmt.Errorf
# ============================================================================

Write-Host "Step 2: Transforming goerr to fmt.Errorf..." -ForegroundColor Green

# Find all Go files
$goFiles = Get-ChildItem -Path $PUBLIC_REPO -Filter "*.go" -Recurse | Where-Object { $_.FullName -notlike "*vendor*" }

$transformedCount = 0

foreach ($file in $goFiles) {
    $content = Get-Content -Path $file.FullName -Raw
    $originalContent = $content
    
    # Check if file uses goerr
    if ($content -match 'goerr') {
        
        # Remove goerr import
        # Handle single import: import "github.com/angel-one/goerr"
        $content = $content -replace 'import\s+"github\.com/angel-one/goerr"\s*\n', ''
        
        # Handle import block: "github.com/angel-one/goerr"
        $content = $content -replace '\s*"github\.com/angel-one/goerr"\s*\n', "`n"
        
        # ====================================================================
        # Pattern 1: goerr.New(nil, fmt.Sprintf("format", args)) 
        #         -> fmt.Errorf("format", args)
        # ====================================================================
        $content = $content -replace 'goerr\.New\(nil,\s*fmt\.Sprintf\(("[^"]+"),\s*([^)]+)\)\)', 'fmt.Errorf($1, $2)'
        
        # ====================================================================
        # Pattern 2: goerr.New(err, fmt.Sprintf("format", args))
        #         -> fmt.Errorf("format: %w", args, err)
        # Handle expressions like ctx.Err(), lastErr, etc.
        # ====================================================================
        # Match: goerr.New(EXPR, fmt.Sprintf("FORMAT", ARGS))
        # Where EXPR can be: word, word.Method(), word.field
        $content = $content -replace 'goerr\.New\((\w+(?:\.\w+\(\))?),\s*fmt\.Sprintf\("([^"]+)",\s*([^)]+)\)\)', 'fmt.Errorf("$2: %w", $3, $1)'
        
        # ====================================================================
        # Pattern 3: goerr.New(nil, "message") -> fmt.Errorf("message")
        # ====================================================================
        $content = $content -replace 'goerr\.New\(nil,\s*"([^"]+)"\)', 'fmt.Errorf("$1")'
        
        # ====================================================================
        # Pattern 4: goerr.New(err, "message") -> fmt.Errorf("message: %w", err)
        # Where err is a simple variable name
        # ====================================================================
        $content = $content -replace 'goerr\.New\((\w+),\s*"([^"]+)"\)', 'fmt.Errorf("$2: %w", $1)'
        
        # Ensure fmt is imported if we made changes and it's not already imported
        if ($content -ne $originalContent) {
            # Check if fmt is already imported
            if ($content -notmatch '"fmt"') {
                # Add fmt to imports - find import block and add fmt
                if ($content -match 'import \(') {
                    # Multi-line import block - add fmt after opening paren
                    $content = $content -replace '(import \(\s*\n)', "`$1`t`"fmt`"`n"
                } elseif ($content -match 'import "') {
                    # Single import - convert to block
                    $content = $content -replace 'import "([^"]+)"', "import (`n`t`"fmt`"`n`t`"`$1`"`n)"
                }
            }
            
            $transformedCount++
            Write-Host "  Transformed: $($file.Name)" -ForegroundColor Gray
        }
    }
    
    # Write back if changed
    if ($content -ne $originalContent) {
        Set-Content -Path $file.FullName -Value $content -NoNewline
    }
}

Write-Host "  Transformed $transformedCount files" -ForegroundColor Green
Write-Host ""

# ============================================================================
# STEP 3: Update go.mod
# ============================================================================

Write-Host "Step 3: Updating go.mod..." -ForegroundColor Green

$goModPath = Join-Path $PUBLIC_REPO "go.mod"
$goModContent = Get-Content -Path $goModPath -Raw

# Update module path
$goModContent = $goModContent -replace [regex]::Escape($INTERNAL_MODULE), $PUBLIC_MODULE

# Remove goerr dependency
$goModContent = $goModContent -replace '\s*github\.com/angel-one/goerr\s+v[^\n]+\n', "`n"

Set-Content -Path $goModPath -Value $goModContent -NoNewline

Write-Host "  Updated module path to: $PUBLIC_MODULE" -ForegroundColor Gray
Write-Host "  Removed goerr dependency" -ForegroundColor Gray
Write-Host ""

# ============================================================================
# STEP 4: Update internal imports in Go files
# ============================================================================

Write-Host "Step 4: Updating internal imports..." -ForegroundColor Green

foreach ($file in $goFiles) {
    $content = Get-Content -Path $file.FullName -Raw
    $originalContent = $content
    
    # Replace internal module references
    $content = $content -replace [regex]::Escape($INTERNAL_MODULE), $PUBLIC_MODULE
    
    if ($content -ne $originalContent) {
        Set-Content -Path $file.FullName -Value $content -NoNewline
        Write-Host "  Updated imports: $($file.Name)" -ForegroundColor Gray
    }
}

Write-Host ""

# ============================================================================
# STEP 5: Update telemetry tracer name
# ============================================================================

Write-Host "Step 5: Updating telemetry references..." -ForegroundColor Green

$telemetryFile = Join-Path $PUBLIC_REPO "backend\telemetry\telemetry.go"
if (Test-Path $telemetryFile) {
    $content = Get-Content -Path $telemetryFile -Raw
    $content = $content -replace [regex]::Escape($INTERNAL_MODULE), $PUBLIC_MODULE
    Set-Content -Path $telemetryFile -Value $content -NoNewline
    Write-Host "  Updated TracerName constant" -ForegroundColor Gray
}

Write-Host ""

# ============================================================================
# STEP 6: Run go mod tidy
# ============================================================================

Write-Host "Step 6: Running go mod tidy..." -ForegroundColor Green

Push-Location $PUBLIC_REPO
try {
    & go mod tidy 2>&1 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
    Write-Host "  Done!" -ForegroundColor Green
} catch {
    Write-Host "  Warning: go mod tidy failed. You may need to run it manually." -ForegroundColor Yellow
}
Pop-Location

Write-Host ""

# ============================================================================
# SUMMARY
# ============================================================================

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Import Complete!" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Review the changes: git status" -ForegroundColor White
Write-Host "  2. Test the build:     go build ./..." -ForegroundColor White
Write-Host "  3. Run unit tests:     go test ./..." -ForegroundColor White
Write-Host "  4. Run integration:    go test -tags=integration ./..." -ForegroundColor White
Write-Host "  5. Commit changes:     git add -A && git commit -m 'Import from upstream'" -ForegroundColor White
Write-Host "  6. Push to GitHub:     git push origin main" -ForegroundColor White
