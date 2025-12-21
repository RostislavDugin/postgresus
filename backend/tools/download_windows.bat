@echo off
setlocal enabledelayedexpansion

echo Downloading and installing PostgreSQL and MySQL client tools for Windows...
echo.

:: Create directories if they don't exist
if not exist "downloads" mkdir downloads
if not exist "postgresql" mkdir postgresql
if not exist "mysql" mkdir mysql
if not exist "mariadb" mkdir mariadb

:: Get the absolute paths
set "POSTGRES_DIR=%cd%\postgresql"
set "MYSQL_DIR=%cd%\mysql"
set "MARIADB_DIR=%cd%\mariadb"

echo PostgreSQL will be installed to: %POSTGRES_DIR%
echo MySQL will be installed to: %MYSQL_DIR%
echo MariaDB will be installed to: %MARIADB_DIR%
echo.

cd downloads

:: ========== PostgreSQL Installation ==========
echo ========================================
echo Installing PostgreSQL client tools (versions 12-18)...
echo ========================================
echo.

:: PostgreSQL download URLs for Windows x64
set "BASE_URL=https://get.enterprisedb.com/postgresql"

:: Define PostgreSQL versions and their corresponding download URLs
set "PG12_URL=%BASE_URL%/postgresql-12.20-1-windows-x64.exe"
set "PG13_URL=%BASE_URL%/postgresql-13.16-1-windows-x64.exe"
set "PG14_URL=%BASE_URL%/postgresql-14.13-1-windows-x64.exe"
set "PG15_URL=%BASE_URL%/postgresql-15.8-1-windows-x64.exe"
set "PG16_URL=%BASE_URL%/postgresql-16.4-1-windows-x64.exe"
set "PG17_URL=%BASE_URL%/postgresql-17.0-1-windows-x64.exe"
set "PG18_URL=%BASE_URL%/postgresql-18.0-1-windows-x64.exe"

:: PostgreSQL versions
set "pg_versions=12 13 14 15 16 17 18"

:: Download and install each PostgreSQL version
for %%v in (%pg_versions%) do (
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
                goto :next_pg_version
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
    
    :next_pg_version
    echo.
)

:: ========== MySQL Installation ==========
echo ========================================
echo Installing MySQL client tools (versions 5.7, 8.0, 8.4)...
echo ========================================
echo.

:: MySQL download URLs for Windows x64 (ZIP archives) - using CDN
:: Note: 5.7 is in Downloads, 8.0 and 8.4 specific versions are in archives
set "MYSQL57_URL=https://cdn.mysql.com/Downloads/MySQL-5.7/mysql-5.7.44-winx64.zip"
set "MYSQL80_URL=https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.40-winx64.zip"
set "MYSQL84_URL=https://cdn.mysql.com/archives/mysql-8.4/mysql-8.4.3-winx64.zip"

:: MySQL versions
set "mysql_versions=5.7 8.0 8.4"

:: Download and install each MySQL version
for %%v in (%mysql_versions%) do (
    echo Processing MySQL %%v...
    set "version_underscore=%%v"
    set "version_underscore=!version_underscore:.=!"
    set "filename=mysql-%%v-winx64.zip"
    set "install_dir=%MYSQL_DIR%\mysql-%%v"
    
    :: Build the URL variable name and get its value
    call set "current_url=%%MYSQL!version_underscore!_URL%%"
    
    :: Check if already installed
    if exist "!install_dir!\bin\mysqldump.exe" (
        echo MySQL %%v already installed, skipping...
    ) else (
        :: Download if not exists
        if not exist "!filename!" (
            echo Downloading MySQL %%v...
            echo Downloading from: !current_url!
            curl -L -o "!filename!" -A "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" "!current_url!"
            if !errorlevel! neq 0 (
                echo ERROR: Download request failed
                goto :next_mysql_version
            )
            if not exist "!filename!" (
                echo ERROR: Download failed - file not created
                goto :next_mysql_version
            )
            for %%s in ("!filename!") do if %%~zs LSS 1000000 (
                echo ERROR: Download failed - file too small, likely error page
                del "!filename!" 2>nul
                goto :next_mysql_version
            )
            echo MySQL %%v downloaded successfully
        ) else (
            echo MySQL %%v already downloaded
        )
        
        :: Verify file exists before extraction
        if not exist "!filename!" (
            echo Download file not found, skipping extraction...
            goto :next_mysql_version
        )
        
        :: Extract MySQL
        echo Extracting MySQL %%v...
        mkdir "!install_dir!" 2>nul
        
        powershell -Command "Expand-Archive -Path '!filename!' -DestinationPath '!install_dir!_temp' -Force"
        
        :: Move files from nested directory to install_dir
        for /d %%d in ("!install_dir!_temp\mysql-*") do (
            if exist "%%d\bin\mysqldump.exe" (
                mkdir "!install_dir!\bin" 2>nul
                copy "%%d\bin\mysql.exe" "!install_dir!\bin\" >nul 2>&1
                copy "%%d\bin\mysqldump.exe" "!install_dir!\bin\" >nul 2>&1
            )
        )
        
        :: Cleanup temp directory
        rmdir /s /q "!install_dir!_temp" 2>nul
        
        :: Verify installation
        if exist "!install_dir!\bin\mysqldump.exe" (
            echo MySQL %%v client tools installed successfully
        ) else (
            echo Failed to install MySQL %%v - mysqldump.exe not found
        )
    )
    
    :next_mysql_version
    echo.
)

:: ========== MariaDB Installation ==========
echo ========================================
echo Installing MariaDB client tools (versions 10.6 and 12.1)...
echo ========================================
echo.

:: MariaDB uses two client versions:
:: - 10.6 (legacy): For older servers (5.5, 10.1) that don't have generation_expression column
:: - 12.1 (modern): For newer servers (10.2+)

:: MariaDB download URLs
set "MARIADB106_URL=https://archive.mariadb.org/mariadb-10.6.21/winx64-packages/mariadb-10.6.21-winx64.zip"
set "MARIADB121_URL=https://archive.mariadb.org/mariadb-12.1.2/winx64-packages/mariadb-12.1.2-winx64.zip"

:: MariaDB versions to install
set "mariadb_versions=10.6 12.1"

:: Download and install each MariaDB version
for %%v in (%mariadb_versions%) do (
    echo Processing MariaDB %%v...
    set "version_underscore=%%v"
    set "version_underscore=!version_underscore:.=!"
    set "mariadb_install_dir=%MARIADB_DIR%\mariadb-%%v"
    
    :: Build the URL variable name and get its value
    call set "current_url=%%MARIADB!version_underscore!_URL%%"
    
    :: Check if already installed
    if exist "!mariadb_install_dir!\bin\mariadb-dump.exe" (
        echo MariaDB %%v already installed, skipping...
    ) else (
        :: Extract version number from URL for filename
        for %%u in ("!current_url!") do set "mariadb_filename=%%~nxu"
        
        if not exist "!mariadb_filename!" (
            echo Downloading MariaDB %%v...
            echo Downloading from: !current_url!
            curl -L -o "!mariadb_filename!" -A "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" "!current_url!"
            if !errorlevel! neq 0 (
                echo ERROR: Download request failed
                goto :next_mariadb_version
            )
            if not exist "!mariadb_filename!" (
                echo ERROR: Download failed - file not created
                goto :next_mariadb_version
            )
            for %%s in ("!mariadb_filename!") do if %%~zs LSS 1000000 (
                echo ERROR: Download failed - file too small, likely error page
                del "!mariadb_filename!" 2>nul
                goto :next_mariadb_version
            )
            echo MariaDB %%v downloaded successfully
        ) else (
            echo MariaDB %%v already downloaded
        )
        
        :: Verify file exists before extraction
        if not exist "!mariadb_filename!" (
            echo Download file not found, skipping extraction...
            goto :next_mariadb_version
        )
        
        :: Extract MariaDB
        echo Extracting MariaDB %%v...
        mkdir "!mariadb_install_dir!" 2>nul
        mkdir "!mariadb_install_dir!\bin" 2>nul
        
        powershell -Command "Expand-Archive -Path '!mariadb_filename!' -DestinationPath '!mariadb_install_dir!_temp' -Force"
        
        :: Move files from nested directory to install_dir
        for /d %%d in ("!mariadb_install_dir!_temp\mariadb-*") do (
            if exist "%%d\bin\mariadb-dump.exe" (
                copy "%%d\bin\mariadb.exe" "!mariadb_install_dir!\bin\" >nul 2>&1
                copy "%%d\bin\mariadb-dump.exe" "!mariadb_install_dir!\bin\" >nul 2>&1
            )
        )
        
        :: Cleanup temp directory
        rmdir /s /q "!mariadb_install_dir!_temp" 2>nul
        
        :: Verify installation
        if exist "!mariadb_install_dir!\bin\mariadb-dump.exe" (
            echo MariaDB %%v client tools installed successfully
        ) else (
            echo Failed to install MariaDB %%v - mariadb-dump.exe not found
        )
    )
    
    :next_mariadb_version
    echo.
)

:skip_mariadb
echo.

cd ..

echo.
echo ========================================
echo Installation process completed!
echo ========================================
echo.
echo PostgreSQL versions are installed in: %POSTGRES_DIR%
echo MySQL versions are installed in: %MYSQL_DIR%
echo MariaDB is installed in: %MARIADB_DIR%
echo.

:: List installed PostgreSQL versions
echo Installed PostgreSQL client versions:
for %%v in (%pg_versions%) do (
    set "version_dir=%POSTGRES_DIR%\postgresql-%%v"
    if exist "!version_dir!\bin\pg_dump.exe" (
        echo   postgresql-%%v: !version_dir!\bin\
    )
)

echo.
echo Installed MySQL client versions:
for %%v in (%mysql_versions%) do (
    set "version_dir=%MYSQL_DIR%\mysql-%%v"
    if exist "!version_dir!\bin\mysqldump.exe" (
        echo   mysql-%%v: !version_dir!\bin\
    )
)

echo.
echo Installed MariaDB client versions:
for %%v in (%mariadb_versions%) do (
    set "version_dir=%MARIADB_DIR%\mariadb-%%v"
    if exist "!version_dir!\bin\mariadb-dump.exe" (
        echo   mariadb-%%v: !version_dir!\bin\
    )
)

echo.
echo Usage examples:
echo   %POSTGRES_DIR%\postgresql-15\bin\pg_dump.exe --version
echo   %MYSQL_DIR%\mysql-8.0\bin\mysqldump.exe --version
echo   %MARIADB_DIR%\mariadb-12.1\bin\mariadb-dump.exe --version
echo.

pause
