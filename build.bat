@echo off

:: Force utf-8 encoding (support emoji's)
chcp 65001 >nul

setlocal enabledelayedexpansion

:: Set module path (adjust if it ever changes)
set MODULE=github.com/Jaeiya/koshime/version

:: Get latest version tag
for /f "delims=" %%i in ('git describe --tags --abbrev=0 2^>nul') do set VERSION=%%i

:: Get commit hash (short)
for /f %%i in ('git rev-parse --short HEAD') do set COMMIT=%%i

:: Get UTC build date
for /f %%i in ('powershell -NoProfile -Command "Get-Date -Format yyyy-MM-ddTHH:mm:ssZ"') do set DATE=%%i

:: Decide what to use as build tag
if "%VERSION%"=="" (
    set BUILD_TAG=%COMMIT%
    set VERSION_STR=
) else (
    set BUILD_TAG=%VERSION%
    set VERSION_STR=%VERSION%
)

:: Set output binary name
set OUTPUT=dist\koshime-%BUILD_TAG%

if exist "%OUTPUT%.exe" (
    echo.
    echo  ⛔ Current version already built
    goto end
)

:: Run the build with ldflags
go build -ldflags="-s -w -X %MODULE%.Version=%VERSION_STR% -X %MODULE%.Commit=%COMMIT% -X %MODULE%.Date=%DATE%" -trimpath -o %OUTPUT%.exe
if errorlevel 1 (
    echo ❌ Build failed.
    exit /b 1
)

:: Path to 7z executable (assumed in PATH)
where 7z >nul 2>&1
if errorlevel 1 (
    echo ⚠️ 7z not found in PATH. Skipping packaging.
    goto binarysuccess
)

:: In case we build the same version again
if exist "%OUTPUT%.7z" del "%OUTPUT%.7z"

:: Package binary into .7z archive
echo Creating 7z archive %OUTPUT%.7z ...
7z a -t7z -mx=7 "%OUTPUT%.7z" "%OUTPUT%.exe"
if errorlevel 1 (
    echo ❌ 7z packaging failed.
    exit /b 1
) else (
    echo.
    echo  ✅ Built Archive: %OUTPUT%.7z
)

@REM :: Check if UPX is available
@REM where upx >nul 2>&1
@REM if errorlevel 1 (
@REM     echo ⚠️ UPX not found in PATH. Skipping compression.
@REM     goto end
@REM )

@REM :: Compress binary with UPX
@REM echo Compressing %OUTPUT% with UPX...
@REM upx --best --lzma %OUTPUT%
@REM if errorlevel 1 (
@REM     echo ❌ UPX compression failed.
@REM     exit /b 1
@REM )


:binarysuccess
echo.
echo  ✅ Built binary: %OUTPUT%

:end

endlocal
