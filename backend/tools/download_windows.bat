@echo off
setlocal enabledelayedexpansion

echo Downloading and installing PostgreSQL versions 12-18 for Windows...
echo.

:: Create downloads and postgresql directories if they don't exist
if not exist "downloads" mkdir downloads
if not exist "postgresql" mkdir postgresql

:: Get the absolute path to the postgresql directory
set "POSTGRES_DIR=%cd%\postgresql"

cd downloads

:: PostgreSQL download URLs for Windows x64
set "BASE_URL=https://get.enterprisedb.com/postgresql"

:: Define versions and their corresponding download URLs
set "PG12_URL=%BASE_URL%/postgresql-12.20-1-windows-x64.exe"
set "PG13_URL=%BASE_URL%/postgresql-13.16-1-windows-x64.exe"
set "PG14_URL=%BASE_URL%/postgresql-14.13-1-windows-x64.exe"
set "PG15_URL=%BASE_URL%/postgresql-15.8-1-windows-x64.exe"
set "PG16_URL=%BASE_URL%/postgresql-16.4-1-windows-x64.exe"
set "PG17_URL=%BASE_URL%/postgresql-17.0-1-windows-x64.exe"
set "PG18_URL=%BASE_URL%/postgresql-18.0-1-windows-x64.exe"

:: Array of versions
set "versions=12 13 14 15 16 17 18"

:: Download and install each version
for %%v in (%versions%) do (
    echo Processing PostgreSQL %%v...
    set "filename=postgresql-%%v-windows-x64.exe"
    set "install_dir=%POSTGRES_DIR%\postgresql-%%v"
    
    :: Check if already installed
    if exist "!install_dir!" (
        echo PostgreSQL %%v already installed, skipping...
    ) else (
        :: Download if not exists
        if not exist "!filename!" (
            echo Downloading PostgreSQL %%v...
            powershell -Command "& {$ProgressPreference = 'SilentlyContinue'; [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; $uri = '!PG%%v_URL!'; $file = '!filename!'; $client = New-Object System.Net.WebClient; $client.Headers.Add('User-Agent', 'Mozilla/5.0'); $client.add_DownloadProgressChanged({param($s,$e) $pct = $e.ProgressPercentage; $recv = [math]::Round($e.BytesReceived/1MB,1); $total = [math]::Round($e.TotalBytesToReceive/1MB,1); Write-Host ('{0}%% - {1} MB / {2} MB' -f $pct, $recv, $total) -NoNewline; Write-Host ('`r') -NoNewline}); try {Write-Host 'Starting download...'; $client.DownloadFile($uri, $file); Write-Host ''; Write-Host 'Download completed!'} finally {$client.Dispose()}}"
            
            if !errorlevel! neq 0 (
                echo Failed to download PostgreSQL %%v
                goto :next_version
            )
            echo PostgreSQL %%v downloaded successfully
        ) else (
            echo PostgreSQL %%v already downloaded
        )
        
        :: Install PostgreSQL client tools only
        echo Installing PostgreSQL %%v client tools to !install_dir!...
        echo This may take up to 10 minutes even on powerful machines, please wait...
        
        :: First try: Install with component selection
        start /wait "" "!filename!" --mode unattended --unattendedmodeui none --prefix "!install_dir!" --disable-components server,pgAdmin,stackbuilder --enable-components commandlinetools
        
        :: Check if installation actually worked by looking for pg_dump.exe
        if exist "!install_dir!\bin\pg_dump.exe" (
            echo PostgreSQL %%v client tools installed successfully
        ) else (
            echo Component selection failed, trying full installation...
            echo This may take up to 10 minutes even on powerful machines, please wait...
            :: Fallback: Install everything but without starting services
            start /wait "" "!filename!" --mode unattended --unattendedmodeui none --prefix "!install_dir!" --datadir "!install_dir!\data" --servicename "postgresql-%%v" --serviceaccount "NetworkService" --superpassword "postgres" --serverport 543%%v --extract-only 1
            
            :: Check again
            if exist "!install_dir!\bin\pg_dump.exe" (
                echo PostgreSQL %%v installed successfully
            ) else (
                echo Failed to install PostgreSQL %%v - No files found in installation directory
                echo Checking what was created:
                if exist "!install_dir!" (
                    powershell -Command "Get-ChildItem '!install_dir!' -Recurse | Select-Object -First 10 | ForEach-Object { $_.FullName }"
                ) else (
                    echo Installation directory was not created
                )
            )
        )
    )
    
    :next_version
    echo.
)

echo.
echo Installation process completed!
echo PostgreSQL versions are installed in: %POSTGRES_DIR%
echo.

pause
