# eDonish Auto — Makefile

.PHONY: build run run-cli run-gui analyze view test clean help

# Default target
help:
	@echo "eDonish Auto — команды:"
	@echo ""
	@echo "  make build        — Собрать Docker образ"
	@echo "  make run          — Запустить CLI (docker compose up)"
	@echo "  make run-cli      — Запустить CLI с аргументами"
	@echo "  make run-gui      — Запустить GUI (только Linux/X11)"
	@echo "  make analyze      — Только анализ (без записи)"
	@echo "  make view         — Просмотр журнала"
	@echo "  make clean        — Очистить логи и отчёты"
	@echo ""
	@echo "Нативный запуск (без Docker):"
	@echo "  make native-gui   — Запустить GUI"
	@echo "  make native-cli   — Запустить CLI"

# Docker commands
build:
	docker compose build

run:
	docker compose up

run-cli:
	docker compose run --rm edonish-cli $(ARGS)

run-gui:
	xhost +local:docker
	docker compose --profile gui up edonish-gui
	xhost -local:docker

analyze:
	docker compose run --rm edonish-cli --analyze-only --save-report

view:
	docker compose run --rm edonish-cli --view-journal

clean:
	rm -rf logs/ output/
	docker compose down --rmi local 2>/dev/null || true

# Native (non-Docker) commands
native-gui:
	python3 main.py

native-cli:
	python3 main_cli.py $(ARGS)

native-install:
	pip install -r requirements.txt

native-analyze:
	python3 main_cli.py $(ARGS) --analyze-only --save-report
