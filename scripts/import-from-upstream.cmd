@echo off
REM ============================================================================
REM import-from-upstream.cmd - Wrapper for PowerShell import script
REM ============================================================================
REM
REM This script imports and transforms code from the internal (private) 
REM repository to the public repository for HashiCorp submission.
REM
REM Usage:
REM   cd D:\peru\onedrive\works\vault-plugin-secrets-skyflow
REM   scripts\import-from-upstream.cmd
REM
REM ============================================================================

echo.
echo ========================================
echo Skyflow Vault Plugin - Import Script
echo ========================================
echo.

REM Get the directory where this script is located
set "SCRIPT_DIR=%~dp0"

REM Check if PowerShell script exists
if not exist "%SCRIPT_DIR%import-from-upstream.ps1" (
    echo ERROR: PowerShell script not found at:
    echo   %SCRIPT_DIR%import-from-upstream.ps1
    echo.
    pause
    exit /b 1
)

REM Run the PowerShell script with bypass execution policy
echo Running PowerShell import script...
echo.

powershell -ExecutionPolicy Bypass -File "%SCRIPT_DIR%import-from-upstream.ps1"

set "EXIT_CODE=%ERRORLEVEL%"

if %EXIT_CODE% NEQ 0 (
    echo.
    echo ========================================
    echo ERROR: Import failed with code %EXIT_CODE%
    echo ========================================
    echo.
    pause
    exit /b %EXIT_CODE%
)

echo.
pause

