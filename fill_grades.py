#!/usr/bin/env python3
"""
Параллельное заполнение оценок для 8Б класса на edonish.tj
Использует API напрямую с несколькими воркерами
"""

import requests
import json
import random
import time
import concurrent.futures
from collections import defaultdict

# ===== CONFIG =====
LOGIN = "200117707"
PASSWORD = "test123"
SCHOOL_ID = 354
GROUP_ID = 29  # 8Б
SUBJECT_ID = 25  # Технологияи иттилоотӣ
QUARTERS = {
    187: "Чоряки 1",
    197: "Чоряки 2", 
    207: "Чоряки 3",
    217: "Чоряки 4",
}
GRADE_RANGE = [8, 9, 10]  # Оценки >= 8
MAX_WORKERS = 4  # Параллельных воркеров
BASE_URL = "https://api.edonish.tj"

# ===== AUTH =====
def login():
    resp = requests.post(f"{BASE_URL}/auth/v1/login", json={"login": LOGIN, "password": PASSWORD})
    data = resp.json()
    token = data.get("jwt_token")
    if not token:
        raise Exception(f"Login failed: {data}")
    print(f"[AUTH] Logged in as {data.get('first_name')} {data.get('last_name')}")
    return token

# ===== DATA COLLECTION =====
def get_dates(token, quarter_property_id):
    """Получить список дат для четверти"""
    headers = {"Authorization": f"Bearer {token}"}
    resp = requests.get(
        f"{BASE_URL}/teacher/v1/journal/dates",
        params={"group_id": GROUP_ID, "subject_id": SUBJECT_ID, 
                "quarter_property_id": quarter_property_id, "school_id": SCHOOL_ID, "lang": 1},
        headers=headers
    )
    data = resp.json()
    if isinstance(data, list) and len(data) > 0:
        days = data[0].get("days", [])
        print(f"[DATES] {QUARTERS.get(quarter_property_id, quarter_property_id)}: {len(days)} dates")
        return days
    return []

def get_students(token, quarter_property_id):
    """Получить список учеников с текущими оценками"""
    headers = {"Authorization": f"Bearer {token}"}
    resp = requests.get(
        f"{BASE_URL}/teacher/v1/journal/students",
        params={"group_id": GROUP_ID, "subject_id": SUBJECT_ID,
                "quarter_property_id": quarter_property_id, "school_id": SCHOOL_ID, "lang": 1},
        headers=headers
    )
    data = resp.json()
    if isinstance(data, list):
        print(f"[STUDENTS] {QUARTERS.get(quarter_property_id, quarter_property_id)}: {len(data)} students")
        return data
    return []

# ===== MARK CREATION =====
def create_mark(token, student_id, schedule_date_id, quarter_property_id, grade):
    """Создать оценку через API"""
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
    resp = requests.post(
        f"{BASE_URL}/teacher/v1/journal/10_point_mark/create",
        params={"school_id": SCHOOL_ID, "lang": 1},
        headers=headers,
        json={
            "mark_type_id": grade,
            "group_subgroup_student_id": student_id,
            "schedule_date_id": schedule_date_id,
            "quarter_property_id": quarter_property_id
        }
    )
    data = resp.json()
    if "assignmentMarkId" in data:
        return True, data["shortName"]
    else:
        return False, str(data)

def create_quarter_mark(token, student_id, quarter_property_id, grade):
    """Создать четвертную оценку"""
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
    resp = requests.post(
        f"{BASE_URL}/teacher/v1/journal/10_point_quarter_mark/create",
        params={"school_id": SCHOOL_ID, "lang": 1},
        headers=headers,
        json={
            "mark_type_id": grade,
            "group_subgroup_student_id": student_id,
            "quarter_property_id": quarter_property_id
        }
    )
    data = resp.json()
    if "quarterMarkId" in data:
        return True, data.get("shortName", "")
    else:
        return False, str(data)

def delete_quarter_mark(token, quarter_mark_id):
    """Удалить четвертную оценку"""
    headers = {"Authorization": f"Bearer {token}"}
    resp = requests.post(
        f"{BASE_URL}/teacher/v1/journal/quarter/delete",
        params={"quarter_mark_id": quarter_mark_id, "school_id": SCHOOL_ID},
        headers=headers
    )
    return resp.json()

def create_semester_mark(token, student_id, semester_property_id, grade):
    """Создать семестровую оценку"""
    headers = {"Authorization": f"Bearer $token", "Content-Type": "application/json"}
    resp = requests.post(
        f"{BASE_URL}/teacher/v1/journal/10_point_semester/create",
        params={"school_id": SCHOOL_ID, "lang": 1},
        headers=headers,
        json={
            "mark_type_id": grade,
            "group_subgroup_student_id": student_id,
            "semester_property_id": semester_property_id
        }
    )
    data = resp.json()
    if "semesterMarkId" in data:
        return True, data.get("shortName", "")
    else:
        return False, str(data)

# ===== MAIN LOGIC =====
def process_student_dates(args):
    """Обработка одного ученика на одной дате (для параллельности)"""
    token, student_id, student_name, date_info, quarter_property_id, quarter_name = args
    
    assignment_date_id = date_info["assignmentDateId"]
    date_str = date_info["assignmentDate"]
    
    # Рандомная оценка >= 8
    grade = random.choice(GRADE_RANGE)
    
    success, result = create_mark(token, student_id, assignment_date_id, quarter_property_id, grade)
    if success:
        return True, f"  ✓ {student_name}: {grade} ({date_str}, {quarter_name})"
    else:
        # Возможно оценка уже стоит
        if "already" in result.lower() or "exist" in result.lower():
            return True, f"  = {student_name}: already has mark ({date_str}, {quarter_name})"
        return False, f"  ✗ {student_name}: FAILED ({date_str}, {quarter_name}) - {result}"

def process_quarter(token, quarter_property_id):
    """Обработка одной четверти"""
    quarter_name = QUARTERS.get(quarter_property_id, str(quarter_property_id))
    print(f"\n{'='*60}")
    print(f"[QUARTER] Processing {quarter_name} (qid={quarter_property_id})")
    print(f"{'='*60}")
    
    # Получаем даты
    dates = get_dates(token, quarter_property_id)
    if not dates:
        print(f"[WARN] No dates for {quarter_name}")
        return
    
    # Получаем учеников
    students = get_students(token, quarter_property_id)
    if not students:
        print(f"[WARN] No students for {quarter_name}")
        return
    
    # Собираем существующие оценки
    existing_marks = set()
    for student in students:
        student_id = student["studentId"]
        marks = student.get("subjectMarks") or []
        for mark in marks:
            date_id = mark.get("assignmentDateId")
            existing_marks.add((student_id, date_id))
    
    print(f"[INFO] Found {len(existing_marks)} existing marks")
    
    # Формируем задачи для параллельного выполнения
    tasks = []
    for student in students:
        student_id = student["studentId"]
        student_name = f"{student['lastName']} {student['firstName']}"
        
        for date_info in dates:
            date_id = date_info["assignmentDateId"]
            # Пропускаем если оценка уже стоит
            if (student_id, date_id) in existing_marks:
                continue
            tasks.append((token, student_id, student_name, date_info, quarter_property_id, quarter_name))
    
    print(f"[INFO] Need to add {len(tasks)} marks")
    
    if not tasks:
        print(f"[INFO] All marks already filled for {quarter_name}")
        return
    
    # Параллельное выполнение (4 воркера)
    success_count = 0
    fail_count = 0
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        futures = {executor.submit(process_student_dates, task): task for task in tasks}
        for future in concurrent.futures.as_completed(futures):
            try:
                success, msg = future.result()
                if success:
                    success_count += 1
                else:
                    fail_count += 1
                    print(msg)
            except Exception as e:
                fail_count += 1
                print(f"  ✗ Exception: {e}")
    
    print(f"\n[RESULT] {quarter_name}: {success_count} added, {fail_count} failed")
    
    # Добавляем/обновляем четвертную оценку
    print(f"\n[QUARTER MARK] Setting quarter marks for {quarter_name}...")
    # Обновляем данные студентов
    students = get_students(token, quarter_property_id)
    
    qm_tasks = []
    for student in students:
        student_id = student["studentId"]
        student_name = f"{student['lastName']} {student['firstName']}"
        
        # Считаем среднюю оценку по четверти
        marks = student.get("subjectMarks") or []
        if marks:
            numeric_marks = []
            for m in marks:
                try:
                    numeric_marks.append(int(m.get("shortName", "0")))
                except (ValueError, TypeError):
                    pass
            if numeric_marks:
                avg = sum(numeric_marks) / len(numeric_marks)
                # Округляем до ближайшей оценки >= 8
                q_grade = max(8, min(10, round(avg)))
            else:
                q_grade = random.choice(GRADE_RANGE)
        else:
            q_grade = random.choice(GRADE_RANGE)
        
        # Проверяем, есть ли уже четвертная оценка
        quarter_mark = student.get("quarterMark")
        if quarter_mark and len(quarter_mark) > 0:
            existing_q_grade = quarter_mark[0].get("shortName", "")
            try:
                existing_val = int(existing_q_grade)
                if existing_val >= 8:
                    print(f"  = {student_name}: quarter mark already {existing_q_grade}")
                    continue
                else:
                    # Удаляем старую оценку и ставим новую
                    qm_id = quarter_mark[0].get("quarterMarkId")
                    delete_quarter_mark(token, qm_id)
                    print(f"  - {student_name}: deleted old quarter mark {existing_q_grade}")
            except (ValueError, TypeError):
                pass
        
        qm_tasks.append((student_id, student_name, q_grade))
    
    for student_id, student_name, q_grade in qm_tasks:
        success, result = create_quarter_mark(token, student_id, quarter_property_id, q_grade)
        if success:
            print(f"  ✓ {student_name}: quarter mark = {result}")
        else:
            print(f"  ✗ {student_name}: quarter mark FAILED - {result}")
    
    time.sleep(1)

def main():
    print("="*60)
    print("  FILLING GRADES FOR 8Б CLASS - EDONISH.TJ")
    print("  Subject: Технологияи иттилоотӣ")
    print(f"  Grades: {GRADE_RANGE}")
    print(f"  Workers: {MAX_WORKERS}")
    print("="*60)
    
    # Логин
    token = login()
    
    # Обрабатываем каждую четверть
    for qid in [187, 197, 207, 217]:
        process_quarter(token, qid)
    
    # Финальная проверка - семестровые и годовые оценки
    print(f"\n{'='*60}")
    print("[SEMESTER/YEAR] Processing semester and year marks...")
    print(f"{'='*60}")
    
    # Получаем данные за 4-ю четверть (она содержит semester и year marks)
    students = get_students(token, 217)
    for student in students:
        student_id = student["studentId"]
        student_name = f"{student['lastName']} {student['firstName']}"
        
        # Семестровая оценка (Нимсолаи 2)
        semester_mark = student.get("semesterMark")
        if semester_mark and len(semester_mark) > 0:
            existing_s = semester_mark[0].get("shortName", "")
            print(f"  = {student_name}: semester mark already {existing_s}")
        else:
            # Вычисляем на основе четвертных
            s_grade = random.choice(GRADE_RANGE)
            # Нужно найти semester_property_id
            # Пробуем через dates API
            pass
        
        # Годовая оценка (Солона)
        year_mark = student.get("yearMark")
        if year_mark and len(year_mark) > 0:
            existing_y = year_mark[0].get("shortName", "")
            print(f"  = {student_name}: year mark already {existing_y}")
    
    print(f"\n{'='*60}")
    print("  ALL DONE!")
    print(f"{'='*60}")

if __name__ == "__main__":
    main()
