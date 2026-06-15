# ИНСТРУКЦИЯ ПО ЗАПУСКУ

## Шаг 1: Проверка установки Go

```bash
go version
```

Если Go не установлен, скачайте с https://go.dev/dl/

## Шаг 2: Установка системных зависимостей

### Ubuntu/Debian:
```bash
sudo apt-get update
sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev
```

### macOS:
```bash
brew install go
brew install gcc
```

### Windows:
```bash
# Установите MinGW через Chocolatey
choco install mingw
```

## Шаг 3: Инициализация проекта

```bash
# Если вы в корне проекта, создайте новую директорию
mkdir edonish-go
cd edonish-go

# Скопируйте файлы из текущей директории
cp ../go.mod .
cp ../main.go .
cp ../controller.go .
mkdir client && cp -r ../client/* client/
mkdir ui && cp -r ../ui/* ui/
```

## Шаг 4: Установка зависимостей

```bash
go mod init edonish-app
go mod tidy
```

## Шаг 5: Запуск приложения

```bash
go run .
```

Или с явным указанием всех файлов:
```bash
go run main.go controller.go client/client.go client/html_parser.go ui/login_screen.go ui/dashboard.go
```

## Шаг 6: Компиляция (опционально)

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o edonish-app .

# Windows
GOOS=windows GOARCH=amd64 go build -o edonish-app.exe .

# macOS
GOOS=darwin GOARCH=amd64 go build -o edonish-app .
```

## Шаг 7: Настройка API Endpoints

Откройте `controller.go` и измените URL-адреса:

```go
c.client, err = client.NewEdonishClient(
    "https://edonish.tj",              // baseURL
    "https://edonish.tj/auth/v1/login", // authURL - ЗАМЕНИТЕ на реальный
)
```

## Шаг 8: Изучение реального API

1. Откройте https://edonish.tj в браузере
2. Нажмите F12 для открытия DevTools
3. Перейдите на вкладку **Network**
4. Выполните вход
5. Найдите запросы к API
6. Скопируйте URL-адреса и форматы JSON
7. Обновите структуры в `client/client.go`

## Ваши учётные данные

Для тестирования используйте:
- **Логин**: 200117707
- **Пароль**: test123

**ВАЖНО**: Никогда не коммитьте реальные учётные данные в Git!

## Если возникли проблемы

### Ошибка "go: command not found"
Установите Go с https://go.dev/dl/

### Ошибка компиляции Fyne
```bash
go clean -modcache
go mod tidy
```

### Ошибка соединения
Проверьте интернет-соединение и firewall

### Ошибка парсинга JSON
Сравните формат ответа API со структурами в `client/client.go`
