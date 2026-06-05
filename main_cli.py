#!/usr/bin/env python3
"""
eDonish Auto CLI — Headless command-line interface for Docker/servers.
No GUI required. Works fully in terminal.

Usage:
  python3 main_cli.py --login 200117707 --password test123
  python3 main_cli.py --login 200117707 --password test123 --class "8Б" --subject "Технологияи иттилоотӣ" --quarter "Чоряки 4"
  python3 main_cli.py --login 200117707 --password test123 --class all --quarter all --min-grade 8 --max-grade 10
"""
import sys
import os
import argparse
import json
import logging
import time
from datetime import datetime
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from config import (
    APP_NAME, APP_VERSION, MIN_GRADE, MAX_GRADE, DEFAULT_WORKERS,
    LANG_RU, API_BASE, API_LOGIN, API_REFRESH, API_HEADER_INFO,
    API_PREFIXES, JOURNAL_OPTIONS, JOURNAL_DATES, JOURNAL_STUDENTS,
    JOURNAL_MARK_CREATE, JOURNAL_MARK_DELETE, JOURNAL_QUARTER_CREATE,
    JOURNAL_SEMESTER_CREATE, JOURNAL_YEAR_CREATE, GROUPS_LIST,
    PERIOD_QUARTERS, TEACHER_SUBJECT, SUBGROUPS,
)
from api_client import EdonishAPI, AuthenticationError
from grade_engine import GradeEngine, GradePlan

# ── Logging ──────────────────────────────────────────────────────────

LOG_DIR = Path("/app/logs" if os.path.exists("/app/logs") else os.path.expanduser("~"))
LOG_DIR.mkdir(parents=True, exist_ok=True)
LOG_FILE = LOG_DIR / f"edonish_auto_{datetime.now().strftime('%Y%m%d_%H%M%S')}.log"
OUTPUT_DIR = Path("/app/output" if os.path.exists("/app/output") else os.path.expanduser("~"))
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[
        logging.FileHandler(str(LOG_FILE)),
        logging.StreamHandler(sys.stdout),
    ],
)
logger = logging.getLogger("edonish_cli")


# ── ANSI Colors ──────────────────────────────────────────────────────

class C:
    """ANSI color codes for terminal output."""
    RESET = "\033[0m"
    BOLD = "\033[1m"
    RED = "\033[91m"
    GREEN = "\033[92m"
    YELLOW = "\033[93m"
    BLUE = "\033[94m"
    MAGENTA = "\033[95m"
    CYAN = "\033[96m"
    WHITE = "\033[97m"

    @staticmethod
    def ok(msg):    return f"{C.GREEN}{msg}{C.RESET}"
    @staticmethod
    def err(msg):   return f"{C.RED}{msg}{C.RESET}"
    @staticmethod
    def warn(msg):  return f"{C.YELLOW}{msg}{C.RESET}"
    @staticmethod
    def info(msg):  return f"{C.CYAN}{msg}{C.RESET}"
    @staticmethod
    def bold(msg):  return f"{C.BOLD}{msg}{C.RESET}"


# ── Banner ───────────────────────────────────────────────────────────

BANNER = f"""
{C.BOLD}{C.BLUE}╔══════════════════════════════════════════════╗
║           eDonish Auto CLI v{APP_VERSION}            ║
║     Автоматизация электронного журнала       ║
║              edonish.tj                       ║
╚══════════════════════════════════════════════╝{C.RESET}
"""


# ── CLI Callbacks ────────────────────────────────────────────────────

class CLICallbacks:
    """Callback handler for grade engine — prints to terminal."""

    def __init__(self):
        self.last_update = 0

    def on_progress(self, plan: GradePlan):
        """Print progress bar to terminal."""
        total = plan.total_tasks - plan.skipped
        if total <= 0:
            return
        
        now = time.time()
        if now - self.last_update < 0.5 and plan.completed + plan.failed < total:
            return  # Throttle updates
        self.last_update = now

        done = plan.completed + plan.failed
        pct = done / total * 100
        bar_len = 40
        filled = int(bar_len * done / total)
        bar = "█" * filled + "░" * (bar_len - filled)

        sys.stdout.write(
            f"\r  {C.info('Прогресс')}: [{bar}] {pct:.1f}% "
            f"({done}/{total}) "
            f"{C.ok(f'✅{plan.completed}')} {C.err(f'❌{plan.failed}')} "
            f"{C.warn(f'⏭️{plan.skipped}')}"
        )
        sys.stdout.flush()

        if done >= total:
            sys.stdout.write("\n")

    def on_log(self, message: str, level: str = "info"):
        """Print log message to terminal."""
        ts = datetime.now().strftime("%H:%M:%S")
        prefix = {
            "info": C.info("ℹ️"),
            "warning": C.warn("⚠️"),
            "error": C.err("❌"),
        }.get(level, "ℹ️")
        print(f"  [{ts}] {prefix} {message}")


# ── Main Logic ───────────────────────────────────────────────────────

def parse_args():
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(
        description=f"{APP_NAME} CLI — автоматизация edonish.tj",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Примеры:
  # Заполнить все пустые оценки во всех классах/предметах/четвертях
  python3 main_cli.py --login 200117707 --password test123

  # Заполнить только 8Б класс, конкретный предмет, 4-ю четверть
  python3 main_cli.py --login 200117707 --password test123 --class "8Б" --subject "Технологияи иттилоотӣ" --quarter "Чоряки 4"

  # Заполнить с оценками 9-10, 8 воркеров
  python3 main_cli.py --login 200117707 --password test123 --min-grade 9 --max-grade 10 --workers 8

  # Только анализ (без записи оценок)
  python3 main_cli.py --login 200117707 --password test123 --analyze-only

  # Просмотр журнала (вывод в терминал)
  python3 main_cli.py --login 200117707 --password test123 --view-journal --class "8Б" --subject "Технологияи иттилоотӣ" --quarter "Чоряки 4"
""",
    )

    # Auth
    parser.add_argument("--login", required=True, help="Логин (ID) от edonish.tj")
    parser.add_argument("--password", required=True, help="Пароль от edonish.tj")

    # Filters
    parser.add_argument("--class", dest="class_name", default="all",
                        help="Класс (напр. '8Б') или 'all' для всех")
    parser.add_argument("--subject", default="all",
                        help="Предмет или 'all' для всех")
    parser.add_argument("--quarter", default="all",
                        help="Четверть (напр. 'Чоряки 1') или 'all' для всех")

    # Grade settings
    parser.add_argument("--min-grade", type=int, default=MIN_GRADE,
                        help=f"Минимальная оценка (default: {MIN_GRADE})")
    parser.add_argument("--max-grade", type=int, default=MAX_GRADE,
                        help=f"Максимальная оценка (default: {MAX_GRADE})")

    # Execution
    parser.add_argument("--workers", type=int, default=DEFAULT_WORKERS,
                        help=f"Количество параллельных воркеров (default: {DEFAULT_WORKERS})")
    parser.add_argument("--fill-empty", type=bool, default=True,
                        help="Заполнять только пустые ячейки (default: True)")
    parser.add_argument("--quarter-marks", type=bool, default=True,
                        help="Заполнять четвертные оценки (default: True)")
    parser.add_argument("--analyze-only", action="store_true",
                        help="Только анализ, без записи оценок")
    parser.add_argument("--view-journal", action="store_true",
                        help="Просмотр журнала (таблица оценок)")

    # Output
    parser.add_argument("--json-output", action="store_true",
                        help="Вывести результат в JSON формате")
    parser.add_argument("--save-report", action="store_true",
                        help="Сохранить отчёт в файл")

    # Environment variables fallback
    args = parser.parse_args()

    # Override from env vars if not provided via CLI
    if args.login is None:
        args.login = os.environ.get("EDONISH_LOGIN", "")
    if args.password is None:
        args.password = os.environ.get("EDONISH_PASSWORD", "")
    if args.class_name == "all":
        args.class_name = os.environ.get("EDONISH_CLASS", "all")
    if args.subject == "all":
        args.subject = os.environ.get("EDONISH_SUBJECT", "all")
    if args.quarter == "all":
        args.quarter = os.environ.get("EDONISH_QUARTER", "all")

    env_min = os.environ.get("EDONISH_MIN_GRADE")
    if env_min and args.min_grade == MIN_GRADE:
        args.min_grade = int(env_min)
    env_max = os.environ.get("EDONISH_MAX_GRADE")
    if env_max and args.max_grade == MAX_GRADE:
        args.max_grade = int(env_max)
    env_workers = os.environ.get("EDONISH_WORKERS")
    if env_workers and args.workers == DEFAULT_WORKERS:
        args.workers = int(env_workers)

    return args


def print_journal_table(students, dates_data):
    """Print journal as formatted table in terminal."""
    if not students:
        print(C.warn("  Нет данных для отображения"))
        return

    dates = []
    if dates_data and dates_data[0].get("days"):
        dates = dates_data[0]["days"]

    # Header
    header = f"  {'№':<3} {'Ученик':<30}"
    for d in dates:
        date_str = d.get("assignmentDate", "")[5:]
        header += f" {date_str:>6}"
    header += f"  {'Чтв':>4}  {'Смст':>4}  {'Год':>4}"

    print(C.bold(header))
    print("  " + "─" * len(header))

    # Rows
    for i, s in enumerate(students, 1):
        name = f"{s.get('lastName', '')} {s.get('firstName', '')}"
        row = f"  {i:<3} {name:<30}"

        marks_by_date = {}
        for m in s.get("subjectMarks", []):
            marks_by_date[m["assignmentDateId"]] = m.get("shortName", "")

        for d in dates:
            mark = marks_by_date.get(d["assignmentDateId"], "")
            if mark:
                row += f" {C.ok(f'{mark:>6}')}"
            else:
                row += f" {'·':>6}"

        # Quarter/semester/year marks
        qm = s.get("quarterMark", [{}])
        qm_val = qm[0].get("shortName", "") if qm else ""
        sm = s.get("semesterMark", [{}])
        sm_val = sm[0].get("shortName", "") if sm else ""
        ym = s.get("yearMark", [{}])
        ym_val = ym[0].get("shortName", "") if ym else ""

        row += f"  {qm_val:>4}  {sm_val:>4}  {ym_val:>4}"
        print(row)


def save_report(plan: GradePlan, args, output_path: Path):
    """Save execution report to JSON file."""
    report = {
        "timestamp": datetime.now().isoformat(),
        "settings": {
            "class": args.class_name,
            "subject": args.subject,
            "quarter": args.quarter,
            "min_grade": args.min_grade,
            "max_grade": args.max_grade,
            "workers": args.workers,
        },
        "summary": {
            "total_tasks": plan.total_tasks,
            "completed": plan.completed,
            "failed": plan.failed,
            "skipped": plan.skipped,
        },
        "tasks": [
            {
                "student": t.student_name,
                "date": t.date_str,
                "grade": t.mark,
                "subject": t.subject_name,
                "group": t.group_name,
                "status": t.status,
                "error": t.error,
            }
            for t in plan.tasks
        ],
    }
    report_path = output_path / f"report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
    with open(report_path, "w", encoding="utf-8") as f:
        json.dump(report, f, ensure_ascii=False, indent=2)
    print(C.ok(f"\n  📄 Отчёт сохранён: {report_path}"))
    return report_path


def main():
    """Main CLI entry point."""
    args = parse_args()

    print(BANNER)

    # ── Login ────────────────────────────────────────────────────────
    print(C.bold("  🔐 Подключение к edonish.tj..."))
    api = EdonishAPI()

    try:
        user_info = api.login(args.login, args.password)
    except AuthenticationError as e:
        print(C.err(f"\n  ❌ Ошибка авторизации: {e}"))
        sys.exit(1)
    except Exception as e:
        print(C.err(f"\n  ❌ Ошибка сети: {e}"))
        sys.exit(1)

    name = f"{user_info.get('last_name', '')} {user_info.get('first_name', '')}"
    print(C.ok(f"  ✅ Вход выполнен: {name}"))
    print(C.info(f"  🏫 Школа ID: {api.school_id} | Роль: {api.role}"))

    # ── Load Data ────────────────────────────────────────────────────
    print(C.bold("\n  📋 Загрузка данных журнала..."))

    try:
        journal_options = api.get_journal_options()
    except Exception as e:
        print(C.err(f"  ❌ Ошибка загрузки: {e}"))
        sys.exit(1)

    # Parse groups and subjects
    groups = []
    subjects_set = set()

    if journal_options and "groups" in journal_options:
        for g in journal_options["groups"]:
            group_name = f"{g.get('number', '')}{g.get('name', '')}"
            groups.append({
                "id": g["id"],
                "name": group_name,
                "number": g.get("number", ""),
                "group": g.get("name", ""),
                "edit": g.get("edit", False),
                "myClass": g.get("myClass", False),
            })
            for s in g.get("subjects", []):
                subjects_set.add((s["subjectId"], s["subjectName"]))

    teacher_subjects = [{"id": sid, "name": sname} for sid, sname in subjects_set]

    # Get quarters
    try:
        quarters_data = api.get_quarters() or []
    except Exception:
        quarters_data = []

    print(C.ok(f"  ✅ Загружено: {len(groups)} классов, {len(teacher_subjects)} предметов, {len(quarters_data)} четвертей"))

    # Show available data
    print(C.bold("\n  📊 Доступные классы:"))
    for g in groups:
        my_class = " (мой класс)" if g.get("myClass") else ""
        print(f"    • {g['name']}{C.warn(my_class)}")

    print(C.bold("\n  📚 Доступные предметы:"))
    for s in teacher_subjects:
        print(f"    • {s['name']}")

    print(C.bold("\n  📅 Доступные четверти:"))
    for q in quarters_data:
        print(f"    • {q.get('name', '')}")

    # ── Filter Data ──────────────────────────────────────────────────
    selected_groups = groups
    if args.class_name != "all":
        selected_groups = [g for g in groups if g["name"] == args.class_name]
        if not selected_groups:
            print(C.err(f"\n  ❌ Класс '{args.class_name}' не найден!"))
            print(C.info(f"  Доступные: {', '.join(g['name'] for g in groups)}"))
            sys.exit(1)

    selected_subjects = teacher_subjects
    if args.subject != "all":
        selected_subjects = [s for s in teacher_subjects if s["name"] == args.subject]
        if not selected_subjects:
            print(C.err(f"\n  ❌ Предмет '{args.subject}' не найден!"))
            print(C.info(f"  Доступные: {', '.join(s['name'] for s in teacher_subjects)}"))
            sys.exit(1)

    selected_quarters = quarters_data
    if args.quarter != "all":
        selected_quarters = [q for q in quarters_data if q.get("name") == args.quarter]
        if not selected_quarters:
            print(C.err(f"\n  ❌ Четверть '{args.quarter}' не найдена!"))
            print(C.info(f"  Доступные: {', '.join(q.get('name', '') for q in quarters_data)}"))
            sys.exit(1)

    # ── View Journal Mode ────────────────────────────────────────────
    if args.view_journal:
        print(C.bold("\n  📋 Просмотр журнала"))
        print("=" * 80)

        for group in selected_groups:
            for subject in selected_subjects:
                for quarter in selected_quarters:
                    qprop_id = quarter["qpropId"]
                    qname = quarter.get("name", "")
                    print(C.bold(f"\n  📚 {group['name']} | {subject['name']} | {qname}"))
                    print("  " + "─" * 60)

                    try:
                        students = api.get_journal_students(
                            group_id=group["id"],
                            subject_id=subject["id"],
                            quarter_property_id=qprop_id,
                        )
                        dates_data = api.get_journal_dates(
                            group_id=group["id"],
                            subject_id=subject["id"],
                            quarter_property_id=qprop_id,
                        )
                        print_journal_table(students, dates_data)
                    except Exception as e:
                        print(C.err(f"  ❌ Ошибка: {e}"))

        return

    # ── Build Grade Plan ─────────────────────────────────────────────
    callbacks = CLICallbacks()
    engine = GradeEngine(api)
    engine.set_callbacks(
        progress_cb=callbacks.on_progress,
        log_cb=callbacks.on_log,
    )

    print(C.bold("\n  🔍 Анализ журнала..."))
    print("=" * 80)

    # Convert subjects for engine format
    engine_subjects = [
        {"subjectId": s["id"], "subjectName": s["name"]}
        for s in selected_subjects
    ]

    plan = engine.build_grade_plan(
        groups=selected_groups,
        subjects=engine_subjects,
        quarters=selected_quarters,
        min_grade=args.min_grade,
        max_grade=args.max_grade,
        fill_empty_only=args.fill_empty,
    )

    to_execute = sum(1 for t in plan.tasks if t.status == "pending")

    # Print summary
    print(C.bold(f"\n  📊 Результаты анализа:"))
    print(f"    Всего задач:     {plan.total_tasks}")
    print(f"    Будет выполнено: {C.ok(str(to_execute))}")
    print(f"    Пропущено:       {C.warn(str(plan.skipped))}")

    if to_execute == 0:
        print(C.warn("\n  ⚠️ Нет оценок для добавления. Все ячейки уже заполнены!"))
        if args.save_report:
            save_report(plan, args, OUTPUT_DIR)
        return

    # Show breakdown
    from collections import defaultdict
    by_group = defaultdict(list)
    for task in plan.tasks:
        if task.status == "pending":
            key = f"{task.group_name} | {task.subject_name}"
            by_group[key].append(task)

    print(C.bold("\n  📚 Детализация:"))
    for key, tasks in sorted(by_group.items()):
        print(f"    • {key}: {len(tasks)} оценок")
        for t in tasks[:3]:
            print(f"      - {t.student_name} → {t.mark} ({t.date_str})")
        if len(tasks) > 3:
            print(f"      ... и ещё {len(tasks) - 3}")

    if args.analyze_only:
        print(C.info("\n  ℹ️ Режим анализа — оценки НЕ записаны."))
        if args.save_report:
            save_report(plan, args, OUTPUT_DIR)
        if args.json_output:
            print(json.dumps({
                "total": plan.total_tasks,
                "to_execute": to_execute,
                "skipped": plan.skipped,
            }, indent=2))
        return

    # ── Execute ──────────────────────────────────────────────────────
    print(C.bold(f"\n  🚀 Запуск: {to_execute} оценок, {args.workers} воркеров..."))
    print(f"  Диапазон оценок: {args.min_grade}-{args.max_grade}")
    print("=" * 80)

    start_time = time.time()

    engine.execute_plan(
        plan=plan,
        num_workers=args.workers,
    )

    # Quarter marks
    if args.quarter_marks:
        print(C.bold("\n  📋 Заполнение четвертных оценок..."))
        qplan = engine.build_grade_plan_for_quarter_marks(
            groups=selected_groups,
            subjects=engine_subjects,
            quarters=selected_quarters,
            min_grade=args.min_grade,
            max_grade=args.max_grade,
            fill_empty_only=args.fill_empty,
        )
        if qplan.total_tasks > 0:
            engine.execute_quarter_marks(qplan)

    elapsed = time.time() - start_time

    # ── Final Summary ────────────────────────────────────────────────
    print("\n" + "=" * 80)
    print(C.bold("  📊 ИТОГИ:"))
    print(f"    ✅ Успешно:   {C.ok(str(plan.completed))}")
    print(f"    ❌ Ошибки:    {C.err(str(plan.failed))}")
    print(f"    ⏭️  Пропущено: {C.warn(str(plan.skipped))}")
    print(f"    ⏱️  Время:     {elapsed:.1f} сек")
    print("=" * 80)

    # Save report
    if args.save_report:
        save_report(plan, args, OUTPUT_DIR)

    # JSON output
    if args.json_output:
        print(json.dumps({
            "completed": plan.completed,
            "failed": plan.failed,
            "skipped": plan.skipped,
            "elapsed_seconds": round(elapsed, 1),
        }, indent=2))


if __name__ == "__main__":
    main()
