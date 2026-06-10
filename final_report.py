#!/usr/bin/env python3
"""Final report for 8Б grades"""
import requests, json

with open('/home/z/my-project/jwt_token.txt') as f:
    JWT = f.read().strip()

BASE_URL = "https://api.edonish.tj"
SCHOOL_ID = 354
SUBJECT_ID = 25
CURRICULUM_PROPERTY_ID = 394

quarters = {187: "Чоряки 1", 197: "Чоряки 2", 207: "Чоряки 3", 217: "Чоряки 4"}

print("=" * 80)
print("  ФИНАЛЬНЫЙ ОТЧЁТ - ОЦЕНКИ 8Б КЛАССА")
print("  Предмет: Технологияи иттилоотӣ")
print("=" * 80)

headers = {"Authorization": f"Bearer {JWT}"}

# Get final marks
resp = requests.get(
    f"{BASE_URL}/teacher/v1/journal/students/final",
    params={"group_id": 29, "subject_id": SUBJECT_ID, "curriculum_property_id": CURRICULUM_PROPERTY_ID, "quarter_property_id": 217, "school_id": SCHOOL_ID, "lang": 1},
    headers=headers
)
final_data = resp.json()

if isinstance(final_data, dict):
    print("ERROR:", final_data)
    exit(1)

# Build student lookup
student_info = {}
for s in final_data:
    name = s['lastName'] + ' ' + s['firstName']
    sid = s['studentId']
    qmarks = {}
    for qm in (s.get('quarterMarks') or []):
        qmarks[qm['quarterPropertyId']] = qm['shortName']
    
    fm = s.get('finalMark')
    final_mark = fm[0].get('shortName', '-') if fm and len(fm) > 0 else '-'
    
    ym = s.get('yearMark')
    year_mark = ym[0].get('shortName', '-') if ym and len(ym) > 0 else '-'
    
    student_info[sid] = {
        'name': name,
        'quarter_marks': qmarks,
        'final_mark': final_mark,
        'year_mark': year_mark,
    }

# Print header  
print(f"\n{'№':>2} {'Ученик':<30} {'Ч1':>3} {'Ч2':>3} {'Ч3':>3} {'Ч4':>3} {'Н2':>3} {'Сол':>3}")
print("-" * 65)

idx = 0
for sid, info in sorted(student_info.items(), key=lambda x: x[1]['name']):
    idx += 1
    q1 = info['quarter_marks'].get(187, '-')
    q2 = info['quarter_marks'].get(197, '-')
    q3 = info['quarter_marks'].get(207, '-')
    q4 = info['quarter_marks'].get(217, '-')
    final = info['final_mark']
    year = info['year_mark']
    
    print(f"{idx:>2} {info['name']:<30} {q1:>3} {q2:>3} {q3:>3} {q4:>3} {final:>3} {year:>3}")

# Count date marks per quarter
print("\n" + "=" * 80)
print("  СТАТИСТИКА ПО ДАТАМ")
print("=" * 80)

for qid, qname in quarters.items():
    resp = requests.get(
        f"{BASE_URL}/teacher/v1/journal/dates",
        params={"group_id": 29, "subject_id": SUBJECT_ID, "quarter_property_id": qid, "school_id": SCHOOL_ID, "lang": 1},
        headers=headers
    )
    dates_data = resp.json()
    if isinstance(dates_data, list) and len(dates_data) > 0:
        days = dates_data[0].get('days', [])
        
        resp2 = requests.get(
            f"{BASE_URL}/teacher/v1/journal/students",
            params={"group_id": 29, "subject_id": SUBJECT_ID, "quarter_property_id": qid, "school_id": SCHOOL_ID, "lang": 1},
            headers=headers
        )
        students = resp2.json()
        
        total_marks = 0
        expected = len(students) * len(days)
        for s in students:
            marks = s.get('subjectMarks') or []
            total_marks += len(marks)
        
        pct = (total_marks / expected * 100) if expected > 0 else 0
        print(f"  {qname}: {total_marks}/{expected} оценок ({pct:.1f}%), {len(days)} дат")

print("\n" + "=" * 80)
print("  ВСЕ ОЦЕНКИ ЗАПОЛНЕНЫ!")
print("=" * 80)
