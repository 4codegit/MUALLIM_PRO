@echo off
setlocal enabledelayedexpansion

set VERSION=%1
if "%VERSION%"=="" (
    for /f "tokens=*" %%i in ('git describe --tags 2^>^&1') do set VERSION=%%i
)
if "%VERSION%"=="" set VERSION=dev

echo 🔨 Building Edonish App for Windows (version: %VERSION%)

REM Проверка Go
where go >nul 2>nul
if %errorlevel% neq 0 (
    echo ❌ Go не установлен
    exit /b 1
)

REM Установка зависимостей
echo 📦 Установка зависимостей...
choco install mingw --no-progress -y
choco install nsis --no-progress -y

REM Сборка бинарника
echo 📦 Сборка Windows бинарника...
call go mod tidy
set GOOS=windows
set GOARCH=amd64
go build -o edonish-app-windows.exe .

REM Создание установщика NSIS
echo 📦 Создание установщика...
if exist "installer.nsi" (
    "C:\Program Files (x86)\NSIS\makensis.exe" /DVERSION=%VERSION% installer.nsi
)

REM Создание директории для релизов
if not exist "release\windows" mkdir release\windows

move edonish-app-windows.exe release\windows\
move edonish-app-windows-installer.exe release\windows\ 2>nul

echo ✅ Готово! Файлы в release/windows/
dir release\windows\

endlocal
