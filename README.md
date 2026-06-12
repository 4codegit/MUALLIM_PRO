# eDonish Auto 📚

**Автоматизация электронного журнала edonish.tj**

Десктопная программа для автоматического заполнения оценок в электронном журнале eDonish (г. Душанбе, Таджикистан).

## 🆕 Что нового в версии 3.24.0

### 🎯 Выбор конкретного ученика
Теперь можно генерировать оценки только для выбранного ученика:
1. Выберите класс, предмет и четверть
2. Укажите минимальную и максимальную оценку (по умолчанию 5-10)
3. Нажмите **"Анализировать"** (F5)
4. После анализа появится список всех учеников
5. Включите **"Только выбранного ученика"** и выберите нужного
6. Нажмите **"Запустить"**

## Возможности

- 🔐 **Автоматический вход** — логин через API, получение всех данных (школа, классы, предметы, четверти)
- 📋 **Автоматический анализ** — программа сама находит все классы, предметы, четверти, даты и студентов
- 🎯 **Авто-оценки** — заполнение оценок (8-10) на все пустые ячейки журнала
- ⚡ **Параллельные воркеры** — многопоточная обработка (4+ воркеров одновременно)
- 📊 **Просмотр журнала** — удобный просмотр оценок по классам/предметам/четвертям
- 🔄 **Четвертные оценки** — автоматическое заполнение четвертных, семестровых и годовых оценок
- 📝 **Логирование** — подробные логи всех операций
- 📦 **Установщики** — .exe (Windows), .rpm/.deb (Linux), .dmg (macOS)

## Установка

### 🪟 Windows

Скачайте `edonish-auto-2.0.0-setup.exe` и запустите установщик.

Или используйте CLI без установки:
```
edonish-auto-cli.exe --login YOUR_LOGIN --password YOUR_PASSWORD
```

### 🐧 Linux (Ubuntu/Debian)

```bash
# Установите .deb пакет
sudo dpkg -i edonish-auto_2.0.0_amd64.deb

# Или запустите бинарник напрямую
chmod +x edonish-auto-linux-x64
./edonish-auto-linux-x64
```

### 🐧 Linux (Fedora/RHEL)

```bash
sudo rpm -i edonish-auto-2.0.0-1.x86_64.rpm
```

### 🍎 macOS

Откройте `edonish-auto-2.0.0.dmg` и перетащите приложение в Applications.

## Быстрый старт

### Способ 1: Скачанный установщик

1. Установите программу (см. выше)
2. Запустите `eDonish Auto`
3. Введите логин и пароль от edonish.tj
4. Выберите класс, предмет, четверть
5. Нажмите "Анализировать", затем "Запустить"

### Способ 2: Python (ручная установка)

```bash
pip install customtkinter requests
python3 main.py          # GUI режим
python3 main_cli.py --login 200117707 --password test123  # CLI режим
```

### Способ 3: CLI режим

```bash
# Заполнить все пустые оценки
edonish-auto-cli --login 200117707 --password test123

# Только конкретный класс/предмет/четверть
edonish-auto-cli --login 200117707 --password test123 \
  --class "8Б" --subject "Технологияи иттилоотӣ" --quarter "Чоряки 4"

# Только анализ (без записи)
edonish-auto-cli --login 200117707 --password test123 --analyze-only

# Просмотр журнала
edonish-auto-cli --login 200117707 --password test123 \
  --view-journal --class "8Б" --subject "Технологияи иттилоотӣ" --quarter "Чоряки 4"

# С сохранением отчёта в JSON
edonish-auto-cli --login 200117707 --password test123 --save-report --json-output
```

## Сборка из исходников

### Требования для сборки

- Python 3.12+
- PyInstaller: `pip install pyinstaller`
- Для GUI: `pip install customtkinter`
- Для RPM: `sudo apt install rpm` или `sudo dnf install rpm-build`
- Для DEB: `dpkg-deb` (входит в Ubuntu/Debian)
- Для Windows .exe: NSIS (на Windows)
- Для macOS .dmg: `hdiutil` (на macOS)

### Сборка

```bash
# Скомпилировать для текущей платформы
bash build.sh linux     # Linux бинарники
bash build.sh rpm       # Linux + RPM пакет (требует rpm-build, fpm)
bash build.sh windows   # Windows (только на Windows)
bash build.sh macos     # macOS (только на macOS)
bash build.sh all       # Всё для текущей платформы

# Альтернативная сборка RPM/DEB
bash package.sh rpm     # RPM пакет
bash package.sh deb     # DEB пакет

# Или через Makefile
make native-gui         # GUI напрямую
make native-cli         # CLI напрямую
```

### Сборка для всех платформ

#### 🐧 Linux (Fedora/RPM)
```bash
# Установите зависимости для сборки
sudo dnf install rpm-build python3-pip

# Скомпилируйте
./build.sh linux
./package.sh rpm

# Получите RPM в dist/rpm/
```

#### 🪟 Windows
```powershell
# На Windows:
pip install pyinstaller
pyinstaller edonish-auto.spec --clean
pyinstaller edonish-auto-cli.spec --clean

# Получите .exe в dist\windows\
```

#### 🍎 macOS
```bash
# На macOS:
pip3 install pyinstaller
./build.sh macos

# Получите DMG в dist/dmg/
```

#### 🤖 Android
```bash
# Требуется Flet для Android:
pip install flet
flet package android

# Откройте build/android/ в Android Studio и соберите APK
```

### GitHub Actions CI/CD

При пуше тега `v*` автоматически собираются все платформы и создаётся Release:

```bash
git tag v2.0.0
git push origin v2.0.0
```

Артефакты в Release:
- `edonish-auto-2.0.0-setup.exe` — Windows установщик
- `edonish-auto_2.0.0_amd64.deb` — Ubuntu/Debian
- `edonish-auto-2.0.0-1.x86_64.rpm` — Fedora/RHEL
- `edonish-auto-2.0.0.dmg` — macOS

## CLI параметры

```
edonish-auto-cli [OPTIONS]

Обязательные:
  --login LOGIN         Логин (ID) от edonish.tj
  --password PASSWORD   Пароль от edonish.tj

Фильтры:
  --class CLASS         Класс (напр. '8Б') или 'all'
  --subject SUBJECT     Предмет или 'all'
  --quarter QUARTER     Четверть или 'all'

Оценки:
  --min-grade N         Минимальная оценка (default: 8)
  --max-grade N         Максимальная оценка (default: 10)

Выполнение:
  --workers N           Воркеры (default: 4)
  --fill-empty BOOL     Только пустые ячейки (default: True)
  --quarter-marks BOOL  Четвертные оценки (default: True)
  --analyze-only        Только анализ, без записи
  --view-journal        Просмотр журнала

Вывод:
  --json-output         Результат в JSON
  --save-report         Сохранить отчёт в файл
```

## Структура проекта

```
edonish-auto/
├── main.py                 # GUI приложение (CustomTkinter)
├── main_cli.py             # CLI интерфейс (headless)
├── api_client.py           # API клиент edonish.tj
├── grade_engine.py         # Движок автоматизации оценок
├── config.py               # Конфигурация
├── requirements.txt        # Зависимости Python
├── edonish-auto.spec       # PyInstaller spec (GUI)
├── edonish-auto-cli.spec   # PyInstaller spec (CLI)
├── edonish-auto.spec.rpm   # RPM spec file
├── installer.nsi           # NSIS installer (Windows)
├── build.sh                # Скрипт сборки всех платформ
├── package.sh              # DEB/RPM упаковка
├── .github/workflows/      # CI/CD (Windows/Linux/macOS)
├── .env.example            # Пример переменных окружения
├── Makefile                # Удобные команды
├── LICENSE.txt             # MIT лицензия
└── run.sh                  # Скрипт запуска
```

## API Endpoints (обнаружено)

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/auth/v1/login` | POST | Авторизация |
| `/auth/v1/refresh_token` | GET | Обновление токена |
| `/auth/v1/header/info` | GET | Информация о пользователе |
| `/teacher/v1/journal` | OPTIONS | Доступные классы/предметы |
| `/teacher/v1/journal/dates` | GET | Даты журнала |
| `/teacher/v1/journal/students` | GET | Студенты с оценками |
| `/teacher/v1/journal/10_point_mark/create` | POST | Создание оценки |
| `/teacher/v1/journal/10_point_quarter_mark/create` | POST | Четвертная оценка |
| `/teacher/v1/journal/10_point_semester/create` | POST | Семестровая оценка |
| `/teacher/v1/journal/10_point_year/create` | POST | Годовая оценка |
| `/teacher/v1/journal/mark/delete` | POST | Удаление оценки |
| `/school_admin/v1/period/quaters` | GET | Список четвертей |
| `/groups/list` | GET | Список классов школы |
| `/teacher/subject` | GET | Предметы учителя |
| `/subgroups` | GET | Подгруппы класса |

## Поддерживаемые роли

| Роль | Префикс API |
|------|-------------|
| teacher | /teacher/v1 |
| classroom-teacher | /teacher/v1 |
| school_admin | /school_admin/v1 |
| director | /director/v1 |
| headteacher | /headteacher/v1 |
| parent | /parent/v1 |
| student | /student/v1 |

## Безопасность

- Токены JWT хранятся только в памяти
- Пароли не сохраняются
- Все запросы идут через HTTPS
- `.env` файлы исключены из Git

## Лицензия

MIT License — см. [LICENSE.txt](LICENSE.txt)
