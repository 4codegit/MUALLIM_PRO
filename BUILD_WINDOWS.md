# Сборка Windows версии eDonish Auto

## 📋 Требования

- Windows 10/11
- Python 3.11+
- NSIS (Nullsoft Scriptable Install System)
- Git

## 🔧 Установка зависимостей

### 1. Python
Скачайте с: https://www.python.org/downloads/

### 2. NSIS
Вариант A - Chocolatey:
```powershell
choco install nsis
```

Вариант B - Ручная установка:
1. Скачайте с: https://nsis.sourceforge.io/Download
2. Установите в `C:\Program Files (x86)\NSIS`
3. Добавьте в PATH: `C:\Program Files (x86)\NSIS`

### 3. Зависимости проекта
```powershell
cd edonish-auto
pip install -r requirements.txt
pip install pyinstaller
```

## 🛠️ Сборка

### Шаг 1: Собрать бинарники
```powershell
# GUI версия
pyinstaller edonish-auto.spec --clean --noconfirm

# CLI версия
pyinstaller edonish-auto-cli.spec --clean --noconfirm
```

Результат: `dist\windows\edonish-auto.exe` и `dist\windows\edonish-auto-cli.exe`

### Шаг 2: Создать NSIS установщик
```powershell
makensis installer.nsi
```

Результат: `edonish-auto-3.24.0-setup.exe`

### Шаг 3: Создать портативную версию
```powershell
Compress-Archive -Path dist\windows\*.exe -DestinationPath edonish-auto-3.24.0-windows.zip -Force
```

## 📦 Файлы для релиза

- `edonish-auto-3.24.0-setup.exe` — установщик
- `edonish-auto-3.24.0-windows.zip` — портативная версия

## 🤖 Автоматическая сборка через GitHub Actions

Создайте тег:
```bash
git tag v3.24.1
git push origin v3.24.1
```

GitHub Actions автоматически:
1. Соберёт бинарники
2. Создаст ZIP пакет
3. Загрузит в релиз

## ⚙️ Конфигурация NSIS

Файл `installer.nsi` содержит:
- Название приложения
- Версию
- Путь установки
- Ярлыки (рабочий стол, меню Пуск)
- Удаление при деинсталляции

Для изменения версии отредактируйте:
```nsis
!define APPVERSION "3.24.0"
```
