---
Task ID: 1
Agent: main
Task: Add per-student random grade fill with min/max in journal page

Work Log:
- Read entire journal.go (2152 lines), app.go, auto.go to understand full codebase
- Added `currentStudentName` field to JournalPage struct to track selected student
- Added `trackSelectedStudent()` method called from OnSelected handler
- Added `fillRandomGradesForStudent()` method for single-student random fill
- Redesigned `showRandomFillDialog()` with:
  - Student selector dropdown at top (pre-selects currently highlighted student)
  - Min/Max entries for selected student that sync with saved limits
  - "Заполнить выбранного" button (fills only selected student)
  - "Заполнить всех" button (fills all students - existing behavior)
  - Full per-student list with "Установить всем" still available
- Verified brace/paren balance (474/474 braces, 859/859 parens)
- Existing logic preserved - only additions made

Stage Summary:
- Key change: random fill now supports per-student mode
- When user selects a student in the table and clicks "Рандом", the dialog pre-selects that student
- User can set individual min/max for each student
- "Заполнить выбранного" fills only that student's empty cells
- "Заполнить всех" fills all students (original behavior)
- Go compiler not available in this environment; syntax verified manually
