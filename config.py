"""Configuration for Edonish Auto"""
import os
import sys


def _get_app_dir():
    """Return a writable directory for app data (log, session).

    On Android, os.path.expanduser('~') resolves to /data/ which is not
    writable, causing PermissionError.  This helper detects the Android
    environment and falls back to the app's internal storage or temp dir.
    """
    # Detect Android environment specifically
    _is_android = (
        'ANDROID_ARGUMENT' in os.environ
        or 'ANDROID_ROOT' in os.environ
        or os.path.exists('/system/bin/app_process')
    )
    if _is_android:
        # On Android the app runs from /data/user/0/tj.edonish.auto/files/flet/app/
        # The writable parent directory is /data/user/0/tj.edonish.auto/files/
        for candidate in [
            os.environ.get('FLET_APP_DATA'),
            os.environ.get('APP_DATA'),
            '/data/data/tj.edonish.auto/files',
            '/data/user/0/tj.edonish.auto/files',
        ]:
            if candidate and os.path.isdir(candidate):
                return candidate
        # Try to derive from the script's own location
        try:
            script_dir = os.path.dirname(os.path.abspath(__file__))
            # Walk up: .../files/flet/app -> .../files
            parent = os.path.dirname(os.path.dirname(os.path.dirname(script_dir)))
            if os.path.isdir(parent) and os.access(parent, os.W_OK):
                return parent
        except Exception:
            pass
        # Last resort: try home, then temp
        home = os.path.expanduser("~")
        if os.access(home, os.W_OK):
            return home
        import tempfile
        return tempfile.gettempdir()
    return os.path.expanduser("~")

# API Configuration
API_BASE = "https://api.edonish.tj"
API_LOGIN = f"{API_BASE}/auth/v1/login"
API_REFRESH = f"{API_BASE}/auth/v1/refresh_token"
API_HEADER_INFO = f"{API_BASE}/auth/v1/header/info"

# Role-based API prefixes
API_PREFIXES = {
    "teacher": "/teacher/v1",
    "classroom-teacher": "/teacher/v1",
    "school_admin": "/school_admin/v1",
    "director": "/director/v1",
    "headteacher": "/headteacher/v1",
    "chief_curator": "/chief_curator/v1",
    "regional_curator": "/regional_curator/v1",
    "parent": "/parent/v1",
    "student": "/student/v1",
}

# Journal API endpoints (relative to role prefix)
JOURNAL_OPTIONS = "/journal"
JOURNAL_DATES = "/journal/dates"
JOURNAL_STUDENTS = "/journal/students"
JOURNAL_STUDENTS_FINAL = "/journal/students/final"
JOURNAL_DATES_FINAL = "/journal/dates/final"
JOURNAL_MARK_CREATE = "/journal/10_point_mark/create"
JOURNAL_MARK_DELETE = "/journal/mark/delete"
JOURNAL_QUARTER_CREATE = "/journal/10_point_quarter_mark/create"
JOURNAL_SEMESTER_CREATE = "/journal/10_point_semester/create"
JOURNAL_YEAR_CREATE = "/journal/10_point_year/create"
JOURNAL_COMMENT = "/journal/comment"
JOURNAL_ASSIGNMENT_UPDATE = "/journal/assignment/update"

# Other endpoints
GROUPS_LIST = "/groups/list"
PERIOD_QUARTERS = "/period/quaters"
TEACHER_SUBJECT = "/teacher/subject"
SCHOOL_SUBJECTS = "/school/subjects"
SUBGROUPS = "/subgroups"

# Language codes
LANG_TJ = 1  # Тоҷикӣ
LANG_RU = 2  # Русский
LANG_EN = 3  # English

# Grade settings
MIN_GRADE = 5
MAX_GRADE = 10
MAX_GRADE_ALLOW = 11  # Allow manual entry of 11 but not in random
DEFAULT_GRADE_RANGE = (5, 10)

# Worker settings
DEFAULT_WORKERS = 4
DEFAULT_BATCH_SIZE = 4

# App settings
APP_NAME = "eDonish Auto"
APP_VERSION = "3.21.2"
APP_AUTHOR = "Edonish Auto Team"

# Session file — uses mobile-safe directory
SESSION_FILE = os.path.join(_get_app_dir(), ".edonish_session.json")

# Colors
COLOR_PRIMARY = "#1a73e8"
COLOR_SUCCESS = "#34a853"
COLOR_WARNING = "#fbbc04"
COLOR_ERROR = "#ea4335"
COLOR_BG = "#ffffff"
COLOR_BG_DARK = "#1a1a2e"
COLOR_TEXT = "#333333"
COLOR_TEXT_LIGHT = "#ffffff"
