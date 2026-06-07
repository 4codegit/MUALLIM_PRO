"""Grade Engine - handles automated grade creation with parallel workers"""
import random
import math
import time
import logging
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List, Dict, Any, Optional, Callable
from dataclasses import dataclass, field
from config import MIN_GRADE, MAX_GRADE, DEFAULT_WORKERS, DEFAULT_BATCH_SIZE

logger = logging.getLogger("edonish_auto")


def weighted_random_grade(min_grade: int = MIN_GRADE, max_grade: int = MAX_GRADE, include_na: bool = True) -> int:
    """
    Generate a weighted random grade.
    
    Uses a bell-curve-like distribution centered around 7-9,
    with lower probability for very low (3-4) or very high (10) grades.
    This produces more realistic grade distributions.
    
    When include_na=True, grade value 0 represents Н/А (Не аттестован)
    with a small probability (~3%).
    
    Returns:
        int: Grade value (0 for Н/А, or min_grade..max_grade for numeric grades)
    """
    # Build grade options and weights
    grades = []
    weights = []
    
    # Н/А option (mark value 0)
    if include_na:
        grades.append(0)  # 0 = Н/А
        weights.append(1)  # ~3% probability
    
    for g in range(min_grade, max_grade + 1):
        grades.append(g)
        if g <= 4:
            w = 1   # rare low grades
        elif g == 5:
            w = 2
        elif g == 6:
            w = 3
        elif g == 7:
            w = 4
        elif g == 8:
            w = 5
        elif g == 9:
            w = 4
        else:  # 10
            w = 3  # rare perfect score
        weights.append(w)
    return random.choices(grades, weights=weights, k=1)[0]


@dataclass
class GradeTask:
    """Represents a single grade creation task."""
    student_id: int
    student_name: str
    assignment_date_id: str
    date_str: str
    quarter_property_id: int
    mark: int
    subject_name: str = ""
    group_name: str = ""
    status: str = "pending"  # pending, running, success, error, skipped
    error: str = ""
    result: Any = None
    existing_mark_id: str = ""  # ID of existing mark to delete before creating new one
    subject_id: int = 0  # subject_id for quarter mark creation
    curriculum_property_id: int = 0  # curriculumPropertyId for quarter mark creation


@dataclass
class GradePlan:
    """Complete plan for grade creation."""
    tasks: List[GradeTask] = field(default_factory=list)
    total_tasks: int = 0
    completed: int = 0
    failed: int = 0
    skipped: int = 0

    def add_task(self, task: GradeTask):
        self.tasks.append(task)
        self.total_tasks = len(self.tasks)


class GradeEngine:
    """Engine for automated grade creation with parallel processing."""

    def __init__(self, api_client):
        self.api = api_client
        self._stop_event = threading.Event()
        self._progress_callback: Optional[Callable] = None
        self._log_callback: Optional[Callable] = None
        self._running = False

    @property
    def is_running(self) -> bool:
        return self._running

    def set_callbacks(self, progress_cb: Callable = None, log_cb: Callable = None):
        """Set callback functions for progress and logging."""
        self._progress_callback = progress_cb
        self._log_callback = log_cb

    def _log(self, message: str, level: str = "info"):
        """Send a log message through the callback."""
        if self._log_callback:
            self._log_callback(message, level)
        getattr(logger, level)(message)

    def _update_progress(self, plan: GradePlan):
        """Update progress through the callback."""
        if self._progress_callback:
            self._progress_callback(plan)

    def stop(self):
        """Stop the grade engine."""
        self._stop_event.set()
        self._log("Остановка двигателя оценок...", "warning")

    def build_grade_plan(
        self,
        groups: List[Dict],
        subjects: List[Dict],
        quarters: List[Dict],
        min_grade: int = MIN_GRADE,
        max_grade: int = MAX_GRADE,
        fill_empty_only: bool = True,
        grades_per_date: int = 1,
        include_na: bool = True,
    ) -> GradePlan:
        """
        Build a complete plan of grades to create.
        
        For each group/subject/quarter combination:
        - Get available dates
        - Get students list
        - For each student, check if they already have a mark on each date
        - If not (or fill_empty_only=False), plan a random grade
        """
        plan = GradePlan()
        self._log("📋 Построение плана оценок...")

        for group in groups:
            group_id = group["id"]
            group_name = f"{group.get('number', group.get('class', ''))}{group.get('group', group.get('name', ''))}"

            # Use group-specific quarters if available, otherwise use global quarters
            group_quarters = group.get("quarters", quarters)
            # Use group-specific subjects if available, otherwise use global subjects
            group_subjects = group.get("subjects", subjects)

            # Filter subjects to match the selected ones
            if group_subjects is not subjects:  # group has its own subjects
                selected_subject_ids = {s.get("subjectId") for s in subjects}
                effective_subjects = [s for s in group_subjects if s.get("subjectId") in selected_subject_ids] if selected_subject_ids else group_subjects
            else:
                effective_subjects = subjects

            # Filter quarters to match the selected ones
            if group_quarters is not quarters:  # group has its own quarters
                selected_quarter_names = {q.get("name") for q in quarters}
                effective_quarters = [q for q in group_quarters if q.get("name") in selected_quarter_names] if selected_quarter_names else group_quarters
            else:
                effective_quarters = quarters

            for subject in effective_subjects:
                subject_id = subject.get("subjectId", subject.get("id"))
                subject_name = subject.get("subjectName", subject.get("name", ""))

                for quarter in effective_quarters:
                    qprop_id = quarter["qpropId"]
                    quarter_name = quarter.get("name", f"Чоряки {qprop_id}")

                    self._log(f"📚 {group_name} | {subject_name} | {quarter_name}")

                    # Get dates for this combination
                    try:
                        dates_data = self.api.get_journal_dates(
                            group_id=group_id,
                            subject_id=subject_id,
                            quarter_property_id=qprop_id,
                        )
                    except Exception as e:
                        self._log(f"  ❌ Ошибка получения дат: {e}", "error")
                        continue

                    if not dates_data or not dates_data[0].get("days"):
                        self._log(f"  ⏭️ Нет дат для этой комбинации")
                        continue

                    days = dates_data[0].get("days", [])

                    # Get students
                    try:
                        students = self.api.get_journal_students(
                            group_id=group_id,
                            subject_id=subject_id,
                            quarter_property_id=qprop_id,
                        )
                    except Exception as e:
                        self._log(f"  ❌ Ошибка получения студентов: {e}", "error")
                        continue

                    if not students:
                        self._log(f"  ⏭️ Нет студентов")
                        continue

                    self._log(f"  📊 Найдено {len(students)} студентов, {len(days)} дат")

                    # Build a map of student marks by date
                    marks_found = 0
                    for student in students:
                        student_id = student["studentId"]
                        student_name = f"{student.get('lastName', '')} {student.get('firstName', '')}"

                        # Get existing marks indexed by assignmentDateId
                        existing_marks = {}
                        for mark in (student.get("subjectMarks") or []):
                            date_id = mark.get("assignmentDateId")
                            existing_marks[date_id] = mark
                            marks_found += 1

                        # Plan grades for each date
                        for day in days:
                            date_id = day["assignmentDateId"]
                            date_str = day.get("assignmentDate", "")

                            if fill_empty_only and date_id in existing_marks:
                                # Skip - already has a mark
                                task = GradeTask(
                                    student_id=student_id,
                                    student_name=student_name,
                                    assignment_date_id=date_id,
                                    date_str=date_str,
                                    quarter_property_id=qprop_id,
                                    mark=0,
                                    subject_name=subject_name,
                                    group_name=group_name,
                                    status="skipped",
                                )
                                plan.add_task(task)
                                plan.skipped += 1
                                continue

                            # If not fill_empty_only and there's an existing mark, store its ID for deletion
                            existing_mark_id = ""
                            if not fill_empty_only and date_id in existing_marks:
                                existing_mark_id = existing_marks[date_id].get("assignmentMarkId", "")

                            # Generate weighted random grade (0 = Н/А if include_na)
                            grade = weighted_random_grade(min_grade, max_grade, include_na=include_na)
                            task = GradeTask(
                                student_id=student_id,
                                student_name=student_name,
                                assignment_date_id=date_id,
                                date_str=date_str,
                                quarter_property_id=qprop_id,
                                mark=grade,
                                subject_name=subject_name,
                                group_name=group_name,
                                existing_mark_id=existing_mark_id,
                            )
                            plan.add_task(task)

                    if marks_found == 0:
                        self._log(f"  ⚠️ У студентов нет оценок (subjectMarks пустой)")
                    else:
                        self._log(f"  📊 Найдено {marks_found} существующих оценок у студентов")

        self._log(f"✅ План построен: {plan.total_tasks} задач ({plan.skipped} пропущено)")
        return plan

    def build_grade_plan_for_quarter_marks(
        self,
        groups: List[Dict],
        subjects: List[Dict],
        quarters: List[Dict],
        min_grade: int = MIN_GRADE,
        max_grade: int = MAX_GRADE,
        fill_empty_only: bool = True,
    ) -> GradePlan:
        """Build a plan for quarter/semester/year marks."""
        plan = GradePlan()
        self._log("📋 Построение плана четвертных/семестровых/годовых оценок...")

        for group in groups:
            group_id = group["id"]
            group_name = f"{group.get('number', group.get('class', ''))}{group.get('group', group.get('name', ''))}"

            # Use group-specific quarters/subjects if available
            group_quarters = group.get("quarters", quarters)
            group_subjects = group.get("subjects", subjects)
            if group_quarters is not quarters:
                selected_quarter_names = {q.get("name") for q in quarters}
                effective_quarters = [q for q in group_quarters if q.get("name") in selected_quarter_names] if selected_quarter_names else group_quarters
            else:
                effective_quarters = quarters
            if group_subjects is not subjects:
                selected_subject_ids = {s.get("subjectId") for s in subjects}
                effective_subjects = [s for s in group_subjects if s.get("subjectId") in selected_subject_ids] if selected_subject_ids else group_subjects
            else:
                effective_subjects = subjects

            for subject in effective_subjects:
                subject_id = subject.get("subjectId", subject.get("id"))
                subject_name = subject.get("subjectName", subject.get("name", ""))

                for quarter in effective_quarters:
                    qprop_id = quarter["qpropId"]
                    quarter_name = quarter.get("name", f"Чоряки {qprop_id}")

                    try:
                        students = self.api.get_journal_students(
                            group_id=group_id,
                            subject_id=subject_id,
                            quarter_property_id=qprop_id,
                        )
                    except Exception as e:
                        self._log(f"  ❌ Ошибка: {e}", "error")
                        continue

                    if not students:
                        continue

                    for student in students:
                        student_id = student["studentId"]
                        student_name = f"{student.get('lastName', '')} {student.get('firstName', '')}"

                        # Check quarter mark
                        quarter_marks = student.get("quarterMark", [])
                        if fill_empty_only and quarter_marks and quarter_marks[0].get("shortName"):
                            continue

                        # Calculate quarter grade as ceil(average of subject marks)
                        subject_marks = student.get("subjectMarks") or []
                        grade_values = []
                        for m in subject_marks:
                            sn = m.get("shortName", "")
                            # Parse grade: filter fractional format (0/X = Н/А, skip)
                            if sn and "/" in sn:
                                numerator = sn.split("/")[0]
                                if numerator == "0":
                                    continue  # Н/А — skip for average calculation
                                sn = numerator
                            if sn and sn.isdigit():
                                v = int(sn)
                                if MIN_GRADE <= v <= MAX_GRADE:
                                    grade_values.append(v)

                        if grade_values:
                            avg = sum(grade_values) / len(grade_values)
                            grade = min(max(int(math.ceil(avg)), min_grade), max_grade)
                            self._log(f"  📊 {student_name}: средний={avg:.2f} → ceil={grade} (из {len(grade_values)} оценок)")
                        else:
                            # No marks — fallback to weighted random
                            grade = weighted_random_grade(min_grade, max_grade, include_na=False)
                            self._log(f"  ⚠️ {student_name}: нет оценок, рандом={grade}")

                        curriculum_property_id = subject.get("curriculumPropertyId", 0)
                        task = GradeTask(
                            student_id=student_id,
                            student_name=student_name,
                            assignment_date_id="",
                            date_str=quarter_name,
                            quarter_property_id=qprop_id,
                            mark=grade,
                            subject_name=subject_name,
                            group_name=group_name,
                            subject_id=subject_id,
                            curriculum_property_id=curriculum_property_id,
                        )
                        plan.add_task(task)

        self._log(f"✅ План построен: {plan.total_tasks} четвертных оценок")
        return plan

    def execute_plan(
        self,
        plan: GradePlan,
        num_workers: int = DEFAULT_WORKERS,
        batch_delay: float = 0.3,
        task_delay: float = 0.15,
    ) -> GradePlan:
        """
        Execute the grade plan with parallel workers.
        
        Args:
            plan: The grade plan to execute
            num_workers: Number of parallel workers
            batch_delay: Delay between batches in seconds
            task_delay: Delay between individual tasks in seconds
        """
        self._running = True
        self._stop_event.clear()

        # Filter out skipped tasks
        tasks_to_execute = [t for t in plan.tasks if t.status == "pending"]
        total_to_execute = len(tasks_to_execute)

        if total_to_execute == 0:
            self._log("✅ Нет задач для выполнения")
            self._running = False
            return plan

        self._log(f"🚀 Запуск {total_to_execute} задач с {num_workers} воркерами...")

        completed = 0
        failed = 0

        # Split tasks into batches for each worker
        batches = [[] for _ in range(num_workers)]
        for i, task in enumerate(tasks_to_execute):
            batches[i % num_workers].append(task)

        def worker(worker_id: int, tasks: List[GradeTask]):
            nonlocal completed, failed
            for task in tasks:
                if self._stop_event.is_set():
                    task.status = "skipped"
                    continue

                task.status = "running"
                self._update_progress(plan)

                try:
                    # Delete existing mark first if overwriting
                    if task.existing_mark_id:
                        try:
                            self.api.delete_mark(mark_id=task.existing_mark_id)
                        except Exception:
                            pass  # Old mark may not exist or already deleted

                    result = self.api.create_mark(
                        student_id=task.student_id,
                        assignment_date_id=task.assignment_date_id,
                        mark=task.mark,
                        quarter_property_id=task.quarter_property_id,
                    )

                    if result and not (isinstance(result, dict) and result.get("error")):
                        task.status = "success"
                        task.result = result
                        completed += 1
                        plan.completed = completed
                        display_mark = "Н/А" if task.mark == 0 else task.mark
                        self._log(
                            f"  ✅ [{worker_id}] {task.student_name} -> {display_mark} "
                            f"({task.date_str})"
                        )
                    else:
                        task.status = "error"
                        task.error = str(result)
                        failed += 1
                        plan.failed = failed
                        self._log(
                            f"  ❌ [{worker_id}] {task.student_name}: {result}",
                            "error",
                        )

                except Exception as e:
                    task.status = "error"
                    task.error = str(e)
                    failed += 1
                    plan.failed = failed
                    self._log(f"  ❌ [{worker_id}] {task.student_name}: {e}", "error")

                self._update_progress(plan)

                # Delay between tasks
                if not self._stop_event.is_set():
                    time.sleep(task_delay)

        # Execute workers in parallel
        with ThreadPoolExecutor(max_workers=num_workers) as executor:
            futures = []
            for i, batch in enumerate(batches):
                if batch:
                    future = executor.submit(worker, i + 1, batch)
                    futures.append(future)

            for future in as_completed(futures):
                try:
                    future.result()
                except Exception as e:
                    self._log(f"Worker error: {e}", "error")

        self._running = False
        self._log(
            f"🏁 Завершено! ✅ {completed} успешно, ❌ {failed} ошибок, "
            f"⏭️ {plan.skipped} пропущено"
        )
        self._update_progress(plan)
        return plan

    def execute_quarter_marks(
        self,
        plan: GradePlan,
        num_workers: int = DEFAULT_WORKERS,
        task_delay: float = 0.2,
    ) -> GradePlan:
        """Execute quarter/semester/year marks plan."""
        self._running = True
        self._stop_event.clear()

        tasks = [t for t in plan.tasks if t.status == "pending"]
        completed = 0
        failed = 0

        for task in tasks:
            if self._stop_event.is_set():
                break

            task.status = "running"
            try:
                result = self.api.create_quarter_mark(
                    student_id=task.student_id,
                    quarter_property_id=task.quarter_property_id,
                    mark=task.mark,
                    subject_id=task.subject_id,
                    curriculum_property_id=task.curriculum_property_id,
                )
                if result:
                    task.status = "success"
                    completed += 1
                    plan.completed = completed
                    self._log(f"  ✅ {task.student_name} -> {'Н/А' if task.mark == 0 else task.mark} ({task.date_str})")
                else:
                    task.status = "error"
                    failed += 1
                    plan.failed = failed
            except Exception as e:
                task.status = "error"
                task.error = str(e)
                failed += 1
                plan.failed = failed
                self._log(f"  ❌ {task.student_name}: {e}", "error")

            self._update_progress(plan)
            time.sleep(task_delay)

        self._running = False
        self._log(f"🏁 Четвертные оценки: ✅ {completed}, ❌ {failed}")
        return plan

    def execute_signatures(
        self,
        groups: List[Dict],
        subjects: List[Dict],
        quarters: List[Dict],
        signature_text: str = "",
        fill_empty_only: bool = True,
        task_delay: float = 0.2,
    ) -> GradePlan:
        """Add signatures/comments to students in the journal."""
        self._running = True
        self._stop_event.clear()
        plan = GradePlan()
        self._log("📋 Построение плана подписей...")

        for group in groups:
            group_id = group["id"]
            group_name = f"{group.get('number', group.get('class', ''))}{group.get('group', group.get('name', ''))}"

            # Use group-specific quarters/subjects if available
            group_quarters = group.get("quarters", quarters)
            group_subjects = group.get("subjects", subjects)
            if group_quarters is not quarters:
                selected_quarter_names = {q.get("name") for q in quarters}
                effective_quarters = [q for q in group_quarters if q.get("name") in selected_quarter_names] if selected_quarter_names else group_quarters
            else:
                effective_quarters = quarters
            if group_subjects is not subjects:
                selected_subject_ids = {s.get("subjectId") for s in subjects}
                effective_subjects = [s for s in group_subjects if s.get("subjectId") in selected_subject_ids] if selected_subject_ids else group_subjects
            else:
                effective_subjects = subjects

            for subject in effective_subjects:
                subject_id = subject.get("subjectId", subject.get("id"))
                subject_name = subject.get("subjectName", subject.get("name", ""))

                for quarter in effective_quarters:
                    qprop_id = quarter["qpropId"]
                    quarter_name = quarter.get("name", f"Чоряки {qprop_id}")

                    self._log(f"✍️ {group_name} | {subject_name} | {quarter_name}")

                    try:
                        dates_data = self.api.get_journal_dates(
                            group_id=group_id,
                            subject_id=subject_id,
                            quarter_property_id=qprop_id,
                        )
                    except Exception as e:
                        self._log(f"  ❌ Ошибка получения дат: {e}", "error")
                        continue

                    if not dates_data or not dates_data[0].get("days"):
                        continue

                    days = dates_data[0].get("days", [])

                    try:
                        students = self.api.get_journal_students(
                            group_id=group_id,
                            subject_id=subject_id,
                            quarter_property_id=qprop_id,
                        )
                    except Exception as e:
                        self._log(f"  ❌ Ошибка получения студентов: {e}", "error")
                        continue

                    if not students:
                        continue

                    for student in students:
                        student_id = student["studentId"]
                        student_name = f"{student.get('lastName', '')} {student.get('firstName', '')}"

                        # Use the last available date for signature
                        if days:
                            last_day = days[-1]
                            date_id = last_day["assignmentDateId"]
                            date_str = last_day.get("assignmentDate", "")

                            # Check if student already has a mark on this date
                            if fill_empty_only:
                                existing_marks = {}
                                for mark in (student.get("subjectMarks") or []):
                                    existing_marks[mark.get("assignmentDateId")] = mark
                                if date_id in existing_marks:
                                    continue

                            task = GradeTask(
                                student_id=student_id,
                                student_name=student_name,
                                assignment_date_id=date_id,
                                date_str=date_str,
                                quarter_property_id=qprop_id,
                                mark=0,
                                subject_name=subject_name,
                                group_name=group_name,
                            )
                            plan.add_task(task)

        self._log(f"✅ План подписей: {plan.total_tasks} учеников")

        # Execute signatures
        completed = 0
        failed = 0
        comment = signature_text or "Подпись"

        for task in plan.tasks:
            if self._stop_event.is_set():
                break

            task.status = "running"
            try:
                result = self.api.create_comment(
                    student_id=task.student_id,
                    assignment_date_id=task.assignment_date_id,
                    comment=comment,
                    quarter_property_id=task.quarter_property_id,
                )
                if result:
                    task.status = "success"
                    completed += 1
                    plan.completed = completed
                    self._log(f"  ✍️ {task.student_name} — подпись добавлена")
                else:
                    task.status = "error"
                    failed += 1
                    plan.failed = failed
                    self._log(f"  ❌ {task.student_name}: ошибка подписи", "error")
            except Exception as e:
                task.status = "error"
                task.error = str(e)
                failed += 1
                plan.failed = failed
                self._log(f"  ❌ {task.student_name}: {e}", "error")

            self._update_progress(plan)
            time.sleep(task_delay)

        self._running = False
        self._log(f"🏁 Подписи: ✅ {completed}, ❌ {failed}")
        return plan

    def execute_delete_marks(
        self,
        groups: List[Dict],
        subjects: List[Dict],
        quarters: List[Dict],
        task_delay: float = 0.1,
    ) -> GradePlan:
        """Delete all marks for selected group/subject/quarter combinations."""
        self._running = True
        self._stop_event.clear()
        plan = GradePlan()
        self._log("📋 Построение плана удаления оценок...")

        total_to_delete = 0

        for group in groups:
            group_id = group["id"]
            group_name = f"{group.get('number', group.get('class', ''))}{group.get('group', group.get('name', ''))}"

            # Use group-specific quarters/subjects if available
            group_quarters = group.get("quarters", quarters)
            group_subjects = group.get("subjects", subjects)
            if group_quarters is not quarters:
                selected_quarter_names = {q.get("name") for q in quarters}
                effective_quarters = [q for q in group_quarters if q.get("name") in selected_quarter_names] if selected_quarter_names else group_quarters
            else:
                effective_quarters = quarters
            if group_subjects is not subjects:
                selected_subject_ids = {s.get("subjectId") for s in subjects}
                effective_subjects = [s for s in group_subjects if s.get("subjectId") in selected_subject_ids] if selected_subject_ids else group_subjects
            else:
                effective_subjects = subjects

            for subject in effective_subjects:
                subject_id = subject.get("subjectId", subject.get("id"))
                subject_name = subject.get("subjectName", subject.get("name", ""))

                for quarter in effective_quarters:
                    qprop_id = quarter["qpropId"]
                    quarter_name = quarter.get("name", f"Чоряки {qprop_id}")

                    self._log(f"🗑️ {group_name} | {subject_name} | {quarter_name}")

                    try:
                        students = self.api.get_journal_students(
                            group_id=group_id,
                            subject_id=subject_id,
                            quarter_property_id=qprop_id,
                        )
                    except Exception as e:
                        self._log(f"  ❌ Ошибка получения студентов: {e}", "error")
                        continue

                    if not students:
                        self._log(f"  ⏭️ Нет студентов")
                        continue

                    self._log(f"  📊 Найдено {len(students)} студентов")

                    for student in students:
                        student_id = student["studentId"]
                        student_name = f"{student.get('lastName', '')} {student.get('firstName', '')}"

                        # Collect all mark IDs to delete
                        marks = student.get("subjectMarks") or []
                        if not marks:
                            # Try alternate keys
                            marks = student.get("marks") or student.get("subject_marks") or []
                        
                        for mark in marks:
                            mark_id = mark.get("assignmentMarkId") or mark.get("id")
                            if mark_id:
                                task = GradeTask(
                                    student_id=student_id,
                                    student_name=student_name,
                                    assignment_date_id=mark.get("assignmentDateId", ""),
                                    date_str=mark.get("shortName", ""),
                                    quarter_property_id=qprop_id,
                                    mark=int(mark.get("shortName", "0")) if mark.get("shortName", "").isdigit() else 0,
                                    subject_name=subject_name,
                                    group_name=group_name,
                                    result=mark_id,  # store mark_id in result field
                                )
                                plan.add_task(task)
                                total_to_delete += 1

                        # Also collect quarter marks
                        for qm in (student.get("quarterMark") or []):
                            qm_id = qm.get("quarterMarkId") or qm.get("assignmentMarkId") or qm.get("id")
                            if qm_id and qm.get("shortName"):
                                task = GradeTask(
                                    student_id=student_id,
                                    student_name=student_name,
                                    assignment_date_id="",
                                    date_str=f"Чтв: {qm.get('shortName', '')}",
                                    quarter_property_id=qprop_id,
                                    mark=0,
                                    subject_name=subject_name,
                                    group_name=group_name,
                                    result=qm_id,
                                )
                                plan.add_task(task)
                                total_to_delete += 1

        self._log(f"📋 Найдено {total_to_delete} оценок для удаления")

        if total_to_delete == 0:
            self._log("✅ Нет оценок для удаления")
            self._running = False
            return plan

        # Execute deletions
        completed = 0
        failed = 0

        for task in plan.tasks:
            if self._stop_event.is_set():
                break

            task.status = "running"
            try:
                result = self.api.delete_mark(mark_id=task.result)
                if result:
                    task.status = "success"
                    completed += 1
                    plan.completed = completed
                    self._log(f"  🗑️ {task.student_name} — оценка удалена ({task.date_str})")
                else:
                    task.status = "error"
                    failed += 1
                    plan.failed = failed
            except Exception as e:
                task.status = "error"
                task.error = str(e)
                failed += 1
                plan.failed = failed
                self._log(f"  ❌ {task.student_name}: {e}", "error")

            self._update_progress(plan)
            time.sleep(task_delay)

        self._running = False
        self._log(f"🏁 Удаление: ✅ {completed} удалено, ❌ {failed} ошибок")
        return plan
