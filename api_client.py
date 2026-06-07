"""Edonish API Client - handles all API communication"""
import requests
import json
import time
import logging
from typing import Optional, List, Dict, Any, Tuple
from config import (
    API_BASE, API_LOGIN, API_REFRESH, API_HEADER_INFO,
    API_PREFIXES, JOURNAL_OPTIONS, JOURNAL_DATES, JOURNAL_STUDENTS,
    JOURNAL_MARK_CREATE, JOURNAL_MARK_DELETE, JOURNAL_QUARTER_CREATE,
    JOURNAL_SEMESTER_CREATE, JOURNAL_YEAR_CREATE, JOURNAL_DATES_FINAL,
    JOURNAL_STUDENTS_FINAL, JOURNAL_ASSIGNMENT_UPDATE, GROUPS_LIST,
    PERIOD_QUARTERS, TEACHER_SUBJECT,
    SUBGROUPS, LANG_RU, MIN_GRADE, MAX_GRADE
)

logger = logging.getLogger("edonish_auto")


class EdonishAPI:
    """Client for the eDonish API - handles authentication and data operations."""

    def __init__(self):
        self.session = requests.Session()
        self.jwt_token: Optional[str] = None
        self.refresh_token: Optional[str] = None
        self.user_info: Optional[Dict] = None
        self.school_id: Optional[int] = None
        self.role: Optional[str] = None
        self.role_prefix: Optional[str] = None
        self.uid: Optional[str] = None
        self.all_roles: List[Dict] = []  # All roles from header/info: [{name, schoolId, ...}, ...]
        self._roles_raw: List[Dict] = []  # Raw roles data from API

    @property
    def _headers(self) -> Dict[str, str]:
        return {
            "Authorization": f"Bearer {self.jwt_token}",
            "Content-Type": "application/json",
        }

    def _url(self, endpoint: str, use_role_prefix: bool = True) -> str:
        prefix = self.role_prefix if use_role_prefix else ""
        return f"{API_BASE}{prefix}{endpoint}"

    def login(self, login_id: str, password: str) -> Dict[str, Any]:
        """Authenticate with the eDonish API."""
        try:
            resp = self.session.post(
                API_LOGIN,
                json={"login": login_id, "password": password},
                headers={"Content-Type": "application/json"},
                timeout=15,
            )
            resp.raise_for_status()
            data = resp.json()

            if data.get("status_code") != 0:
                raise AuthenticationError(
                    f"Login failed: status_code={data.get('status_code')}"
                )

            self.jwt_token = data["jwt_token"]
            self.refresh_token = data["refresh_token"]
            self.uid = data["uid"]

            # Get user roles and school info
            self.user_info = {
                "uid": data["uid"],
                "first_name": data.get("first_name", ""),
                "last_name": data.get("last_name", ""),
                "roles": data.get("roles", []),
            }

            # Determine role and get school_id
            self._resolve_role_and_school()

            logger.info(
                f"Logged in as {self.user_info['last_name']} {self.user_info['first_name']} "
                f"(role: {self.role}, school: {self.school_id})"
            )
            return self.user_info

        except requests.exceptions.RequestException as e:
            raise AuthenticationError(f"Network error during login: {e}")

    def _resolve_role_and_school(self):
        """Determine user role and school ID from header info.
        
        Stores ALL available roles so the user can switch between them.
        By default, picks the most capable role that allows grade modification.
        """
        try:
            resp = self.session.get(
                API_HEADER_INFO,
                params={"lang": LANG_RU},
                headers=self._headers,
                timeout=10,
            )
            resp.raise_for_status()
            roles_data = resp.json()

            if isinstance(roles_data, list) and len(roles_data) > 0:
                # Store all roles
                self._roles_raw = roles_data
                self.all_roles = [
                    {"name": r.get("name", "unknown"), "schoolId": r.get("schoolId")}
                    for r in roles_data
                ]

                # Priority order: school_admin > director > headteacher > classroom-teacher > teacher
                # Pick the most capable role that supports grade operations
                role_priority = [
                    "school_admin",
                    "director",
                    "headteacher",
                    "chief_curator",
                    "regional_curator",
                    "classroom-teacher",
                    "teacher",
                ]
                best_role = None
                for candidate in role_priority:
                    for r in roles_data:
                        if r.get("name") == candidate:
                            best_role = r
                            break
                    if best_role:
                        break

                if best_role:
                    self.role = best_role.get("name", "teacher")
                    self.role_prefix = API_PREFIXES.get(self.role, "/teacher/v1")
                    self.school_id = best_role.get("schoolId", self.school_id)
                else:
                    # Fallback: use first role
                    primary = roles_data[0]
                    self.school_id = primary.get("schoolId")
                    self.role = primary.get("name", "teacher")
                    self.role_prefix = API_PREFIXES.get(self.role, "/teacher/v1")
            else:
                self.role = "teacher"
                self.role_prefix = API_PREFIXES["teacher"]
                self.all_roles = [{"name": "teacher", "schoolId": None}]
                self._roles_raw = []

        except Exception as e:
            logger.warning(f"Could not resolve role: {e}")
            self.role = "teacher"
            self.role_prefix = API_PREFIXES["teacher"]
            self.all_roles = [{"name": "teacher", "schoolId": None}]
            self._roles_raw = []

    def switch_role(self, role_name: str) -> bool:
        """Switch the active role to a different one from the user's available roles.
        
        Returns True if the role was found and switched, False otherwise.
        """
        for r in self._roles_raw:
            if r.get("name") == role_name:
                self.role = role_name
                self.role_prefix = API_PREFIXES.get(role_name, "/teacher/v1")
                school_id = r.get("schoolId")
                if school_id:
                    self.school_id = school_id
                logger.info(f"Switched role to: {role_name} (prefix: {self.role_prefix}, school: {self.school_id})")
                return True
        logger.warning(f"Role '{role_name}' not found in available roles: {[r.get('name') for r in self._roles_raw]}")
        return False

    @property
    def has_school_admin_rights(self) -> bool:
        """Check if user has school_admin capabilities.
        
        Returns True if user has school_admin role (can access /school_admin/v1 endpoints).
        """
        return self.role == "school_admin" or any(
            r.get("name") == "school_admin" for r in self.all_roles
        )

    @property
    def can_modify_grades(self) -> bool:
        """Check if the current role allows modifying grades.
        
        Grade modification is allowed if the user has at least one of:
        teacher, classroom-teacher, school_admin
        
        Returns True if any of these roles is present in the user's available roles.
        """
        grade_roles = {"teacher", "classroom-teacher", "school_admin"}
        # Check if current role is one of the grade-modification roles
        if self.role in grade_roles:
            return True
        # Check if ANY available role is a grade-modification role
        return any(r.get("name") in grade_roles for r in self.all_roles)

    @property
    def available_role_names(self) -> List[str]:
        """Return list of all available role names for the current user."""
        return [r.get("name", "unknown") for r in self.all_roles]

    def _refresh_auth(self) -> bool:
        """Refresh the JWT token using the refresh token."""
        if not self.refresh_token:
            return False
        try:
            resp = self.session.get(
                API_REFRESH,
                headers={"Authorization": f"Bearer {self.refresh_token}"},
                timeout=10,
            )
            if resp.status_code == 200:
                data = resp.json()
                self.jwt_token = data["jwt_token"]
                self.refresh_token = data["refresh_token"]
                return True
        except Exception as e:
            logger.error(f"Token refresh failed: {e}")
        return False

    def _request(self, method: str, url: str, **kwargs) -> Any:
        """Make an authenticated API request with auto-refresh."""
        kwargs.setdefault("headers", self._headers)
        kwargs.setdefault("timeout", 15)

        resp = self.session.request(method, url, **kwargs)

        if resp.status_code == 403:
            # Try refreshing token first
            if self._refresh_auth():
                kwargs["headers"] = self._headers
                resp = self.session.request(method, url, **kwargs)

            # If still 403 after refresh — check if it's a date-lock issue
            if resp.status_code == 403:
                try:
                    error_body = resp.json()
                    error_msg = error_body.get("message", "") if isinstance(error_body, dict) else ""
                except Exception:
                    error_msg = resp.text[:200] if resp.text else ""

                # Detect date-lock patterns from the server
                date_lock_keywords = [
                    "date", "past", "expired", "locked", "closed",
                    "нельзя", "недоступно", "изменить", "прошл",
                    "cannot", "forbidden", "not allowed",
                    "status_code",
                ]
                msg_lower = error_msg.lower()
                is_date_lock = any(kw in msg_lower for kw in date_lock_keywords)

                if is_date_lock:
                    raise DateLockedError(
                        f"Дата заблокирована сервером: невозможно изменить оценку за прошлую дату. "
                        f"({error_msg or f'HTTP {resp.status_code}'})",
                        status_code=resp.status_code,
                    )

                # Generic 403 — could still be date lock
                raise DateLockedError(
                    f"Доступ запрещён (403). Возможно дата заблокирована. "
                    f"({error_msg or f'HTTP {resp.status_code}'})",
                    status_code=resp.status_code,
                )

        if resp.status_code == 404:
            return None

        resp.raise_for_status()
        return resp.json()

    def get_journal_options(self, lang: int = LANG_RU) -> List[Dict]:
        """Get available classes, subjects, and subgroups for the teacher."""
        return self._request(
            "OPTIONS",
            self._url(JOURNAL_OPTIONS),
            params={"lang": lang, "school_id": self.school_id},
        )

    def get_groups(self, lang: int = LANG_RU) -> List[Dict]:
        """Get all groups/classes in the school."""
        return self._request(
            "GET",
            self._url(GROUPS_LIST, use_role_prefix=False),
            params={"school_id": self.school_id, "lang": lang},
        )

    def get_quarters(self, school_id: int = None, rank_id: int = 2, lang: int = LANG_RU) -> List[Dict]:
        """Get quarter periods for the school (uses school_admin prefix)."""
        result = self._request(
            "GET",
            f"{API_BASE}/school_admin/v1{PERIOD_QUARTERS}",
            params={
                "school_id": school_id or self.school_id,
                "rank_id": rank_id,
                "lang": lang,
            },
        )
        return result if result else []

    def get_journal_dates(
        self, group_id: int, subject_id: int, quarter_property_id: int, lang: int = LANG_RU
    ) -> List[Dict]:
        """Get dates for a specific group/subject/quarter."""
        return self._request(
            "GET",
            self._url(JOURNAL_DATES),
            params={
                "group_id": group_id,
                "subject_id": subject_id,
                "quarter_property_id": quarter_property_id,
                "school_id": self.school_id,
                "lang": lang,
            },
        )

    def get_journal_students(
        self, group_id: int, subject_id: int, quarter_property_id: int, lang: int = LANG_RU
    ) -> List[Dict]:
        """Get students with their marks for a specific group/subject/quarter."""
        result = self._request(
            "GET",
            self._url(JOURNAL_STUDENTS),
            params={
                "group_id": group_id,
                "subject_id": subject_id,
                "quarter_property_id": quarter_property_id,
                "school_id": self.school_id,
                "lang": lang,
            },
        )
        # Debug: log what keys the first student has (for diagnosing mark detection)
        if result and isinstance(result, list) and len(result) > 0:
            first = result[0]
            mark_keys = [k for k in first.keys() if 'mark' in k.lower() or 'Mark' in k]
            total_marks = sum(len(first.get(k, []) or []) for k in mark_keys)
            logger.debug(f"get_journal_students: {len(result)} students, mark_keys={mark_keys}, first_student_marks={total_marks}")
        return result

    def get_journal_dates_final(
        self, group_id: int, curriculum_property_id: int, lang: int = LANG_RU
    ) -> List[Dict]:
        """Get final dates for a group."""
        return self._request(
            "GET",
            self._url(JOURNAL_DATES_FINAL),
            params={
                "group_id": group_id,
                "curriculum_property_id": curriculum_property_id,
                "school_id": self.school_id,
                "lang": lang,
            },
        )

    def create_mark(
        self,
        student_id: int,
        assignment_date_id: str,
        mark: int,
        mark_type_id: int = None,
        quarter_property_id: int = None,
    ) -> Optional[Dict]:
        """Create a mark for a student on a specific date.

        In the 10-point system, mark_type_id must equal the actual grade value
        (e.g. mark=7 → mark_type_id=7), similar to how quarter marks use mark_id=mark.
        
        For Н/А (Не аттестован), mark=0 and mark_type_id=1 (edonish convention).
        """
        # Н/А: mark=0, mark_type_id=1 (edonish uses 1/2 shortName pattern for Н/А)
        if mark == 0:
            effective_mark_type_id = 1  # Н/А mark type in edonish
        else:
            effective_mark_type_id = mark_type_id if mark_type_id is not None else mark
        body = {
            "mark_type_id": effective_mark_type_id,
            "group_subgroup_student_id": student_id,
            "schedule_date_id": assignment_date_id,
            "quarter_property_id": quarter_property_id or 0,
            "mark": mark,
        }
        return self._request(
            "POST",
            self._url(JOURNAL_MARK_CREATE),
            params={"school_id": self.school_id, "lang": LANG_RU},
            json=body,
        )

    def delete_mark(self, mark_id: str) -> Optional[Dict]:
        """Delete a mark by its ID."""
        return self._request(
            "POST",
            self._url(JOURNAL_MARK_DELETE),
            params={"mark_id": mark_id, "school_id": self.school_id},
        )

    def create_quarter_mark(
        self, student_id: int, quarter_property_id: int, mark: int,
        subject_id: int = 0, curriculum_property_id: int = 0,
    ) -> Optional[Dict]:
        """Create a quarter (четвертная) mark."""
        # mark_id is the ID from marks_ten_point matching the mark value
        mark_id = mark  # For 10-point system, mark_id equals the mark value
        body = {
            "group_subgroup_student_id": student_id,
            "quarter_property_id": quarter_property_id,
            "mark": mark,
            "mark_id": mark_id,
            "subject_id": subject_id,
            "curriculum_property_id": curriculum_property_id,
        }
        return self._request(
            "POST",
            self._url(JOURNAL_QUARTER_CREATE),
            params={"school_id": self.school_id},
            json=body,
        )

    def create_semester_mark(
        self, student_id: int, semester_property_id: int, mark: int
    ) -> Optional[Dict]:
        """Create a semester (полугодовая) mark."""
        body = {
            "group_subgroup_student_id": student_id,
            "semester_property_id": semester_property_id,
            "mark": mark,
        }
        return self._request(
            "POST",
            self._url(JOURNAL_SEMESTER_CREATE),
            params={"school_id": self.school_id},
            json=body,
        )

    def create_year_mark(
        self, student_id: int, year_property_id: int, mark: int
    ) -> Optional[Dict]:
        """Create a year (годовая) mark."""
        body = {
            "group_subgroup_student_id": student_id,
            "year_property_id": year_property_id,
            "mark": mark,
        }
        return self._request(
            "POST",
            self._url(JOURNAL_YEAR_CREATE),
            params={"school_id": self.school_id},
            json=body,
        )

    def get_teacher_subjects(self, lang: int = LANG_RU) -> List[Dict]:
        """Get all subjects for teachers in the school."""
        return self._request(
            "GET",
            self._url(TEACHER_SUBJECT, use_role_prefix=False),
            params={"school_id": self.school_id, "lang": lang},
        )

    def get_all_school_subjects(self, lang: int = LANG_RU) -> List[Dict]:
        """Get all subjects available in the school (school_admin capability)."""
        if not self.has_school_admin_rights:
            return self.get_teacher_subjects(lang)
        return self._request(
            "GET",
            f"{API_BASE}/school_admin/v1/subjects",
            params={"school_id": self.school_id, "lang": lang},
        )

    def create_subject(self, name: str, lang: int = LANG_RU) -> Optional[Dict]:
        """Create a new subject (school_admin capability)."""
        if not self.has_school_admin_rights:
            logger.warning("No school_admin rights to create subject")
            return None
        body = {"name": name, "school_id": self.school_id}
        return self._request(
            "POST",
            f"{API_BASE}/school_admin/v1/subjects",
            params={"lang": lang},
            json=body,
        )

    def create_group(self, name: str, number: str, lang: int = LANG_RU) -> Optional[Dict]:
        """Create a new class/group (school_admin capability)."""
        if not self.has_school_admin_rights:
            logger.warning("No school_admin rights to create group")
            return None
        body = {"name": name, "number": number, "school_id": self.school_id}
        return self._request(
            "POST",
            f"{API_BASE}/school_admin/v1/groups",
            params={"lang": lang},
            json=body,
        )

    def delete_group(self, group_id: int, lang: int = LANG_RU) -> Optional[Dict]:
        """Delete a class/group (school_admin capability)."""
        if not self.has_school_admin_rights:
            logger.warning("No school_admin rights to delete group")
            return None
        return self._request(
            "DELETE",
            f"{API_BASE}/school_admin/v1/groups/{group_id}",
            params={"lang": lang},
        )

    def create_comment(
        self,
        student_id: int,
        assignment_date_id: str,
        comment: str,
        quarter_property_id: int = None,
    ) -> Optional[Dict]:
        """Create a comment/signature for a student on a specific date."""
        body = {
            "group_subgroup_student_id": student_id,
            "schedule_date_id": assignment_date_id,
            "quarter_property_id": quarter_property_id or 0,
            "comment": comment,
        }
        return self._request(
            "POST",
            self._url(JOURNAL_COMMENT),
            params={"school_id": self.school_id},
            json=body,
        )

    def get_subgroups(self, group_id: int, lang: int = LANG_RU) -> List[Dict]:
        """Get subgroups for a specific group."""
        return self._request(
            "GET",
            self._url(SUBGROUPS, use_role_prefix=False),
            params={"school_id": self.school_id, "group_id": group_id, "lang": lang},
        )

    def update_assignment(
        self,
        schedule_date_id: str,
        topic: str = "",
        home_work: str = "",
    ) -> Optional[Dict]:
        """Update the topic and/or homework for a lesson date.
        
        Args:
            schedule_date_id: The assignmentDateId from get_journal_dates
            topic: Lesson topic text
            home_work: Homework assignment text
        """
        body = {
            "schedule_date_id": schedule_date_id,
        }
        if topic:
            body["topic"] = topic
        if home_work:
            body["homeWork"] = home_work
        return self._request(
            "POST",
            self._url(JOURNAL_ASSIGNMENT_UPDATE),
            params={"school_id": self.school_id},
            json=body,
        )


class AuthenticationError(Exception):
    """Raised when authentication fails."""
    pass


class APIError(Exception):
    """Raised when an API call fails."""
    pass


class DateLockedError(APIError):
    """Raised when trying to modify a grade on a date that is locked by the server.
    
    This happens when the date has passed (e.g. trying to change a grade
    from yesterday or an earlier date). The edonish.tj server blocks
    modifications to past dates based on role permissions.
    """
    def __init__(self, message: str = "", status_code: int = 0):
        self.status_code = status_code
        super().__init__(message)
