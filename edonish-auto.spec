# -*- mode: python ; coding: utf-8 -*-
"""
eDonish Auto — PyInstaller spec file
Builds: GUI executable with Flet/Flutter UI

Fixes:
  - Includes flet + flet-desktop packages
  - Collects ALL flet data files (Flutter engine, assets)
  - Pre-caches and bundles the Flet View (Flutter engine) binary
  - Works on clean systems without Python installed

Usage:
  # Option 1 (recommended): Use flet pack
  flet pack main.py --onefile --name edonish-auto --windowed \
    --hidden-import=requests --hidden-import=certifi \
    --collect-all certifi

  # Option 2: Use this spec file (requires pre-cached Flet View)
  python -c "import flet_desktop; print('Flet View cached')"  # pre-cache
  python -m PyInstaller edonish-auto.spec --clean --noconfirm
"""
import sys
import os
from pathlib import Path

block_cipher = None

# ── Collect all data files ──────────────────────────────────────
datas_list = []

# Collect all Flet packages (includes Flutter engine, assets, etc.)
for pkg_name in ['flet', 'flet_core', 'flet_runtime', 'flet_desktop']:
    try:
        mod = __import__(pkg_name)
        pkg_path = Path(mod.__path__[0])
        if pkg_path.exists():
            datas_list.append((str(pkg_path), pkg_name))
            print(f"  Adding {pkg_name}: {pkg_path}")
    except ImportError:
        print(f"WARNING: {pkg_name} not found, GUI build may fail!")

# ── Collect Flet View (Flutter engine) from cache ──────────────
# The Flet View binary is NOT part of the flet_desktop Python package.
# It's cached in a platform-specific directory after the first run.
# We need to find and include it so the app doesn't try to download
# at runtime (which fails with SSL errors in PyInstaller bundles).
flet_view_cached = False

# Determine Flet View cache directory based on platform
if sys.platform == 'win32':
    flet_cache_base = os.path.join(
        os.environ.get('LOCALAPPDATA', os.path.expanduser('~')), 'flet')
elif sys.platform == 'darwin':
    flet_cache_base = os.path.join(
        os.path.expanduser('~'), 'Library', 'Caches', 'flet')
else:
    xdg_cache = os.environ.get(
        'XDG_CACHE_HOME', os.path.join(os.path.expanduser('~'), '.cache'))
    flet_cache_base = os.path.join(xdg_cache, 'flet')

if os.path.isdir(flet_cache_base):
    # Find the Flet View directory (e.g., flet-0.85.2)
    for item in sorted(os.listdir(flet_cache_base), reverse=True):
        cache_item = os.path.join(flet_cache_base, item)
        if os.path.isdir(cache_item) and item.startswith('flet-'):
            datas_list.append((cache_item, 'flet'))
            print(f"  Adding Flet View cache: {cache_item}")
            flet_view_cached = True
            break

if not flet_view_cached:
    print("WARNING: Flet View not found in cache!")
    print("  Run 'flet pack' or launch a Flet app once to cache the Flutter engine.")
    print(f"  Expected cache location: {flet_cache_base}")
    print("  The app will try to download the Flutter engine at runtime (may fail).")

# ── Collect certifi CA bundle ───────────────────────────────────
try:
    import certifi
    certifi_path = Path(certifi.__file__).parent
    datas_list.append((str(certifi_path), 'certifi'))
    print(f"  Adding certifi: {certifi_path}")
except ImportError:
    print("WARNING: certifi not found, SSL may not work!")

# ── Collect binaries (Windows DLLs) ────────────────────────────
binaries_list = []

# Windows: collect ALL necessary DLLs from Python installation
if sys.platform == 'win32':
    dll_search_dirs = [
        Path(sys.base_exec_prefix) / "DLLs",
        Path(sys.base_exec_prefix),
    ]
    for dll_dir in dll_search_dirs:
        if not dll_dir.exists():
            continue
        for dll_file in dll_dir.iterdir():
            name_lower = dll_file.name.lower()
            if any(name_lower.startswith(p) for p in [
                'tcl', 'tk', 'vcruntime', 'msvcp',
                'python3', 'libcrypto', 'libssl',
                'libffi', 'sqlite3', 'select',
                'unicodedata', '_ctypes', '_decimal',
                '_hashlib', '_socket', '_bz2',
                '_lzma', '_zlib',
            ]):
                binaries_list.append((str(dll_file), '.'))
                print(f"  Adding DLL: {dll_file.name}")

# ── Hidden imports (platform-safe) ─────────────────────────────
hidden_imports = [
    # Flet
    'flet',
    'flet_core',
    'flet_runtime',
    'flet_desktop',
    # Network
    'requests',
    'urllib3',
    'urllib3.util',
    'certifi',
    'charset_normalizer',
    'idna',
    # Utils
    'darkdetect',
    'packaging',
    'json',
    'threading',
    'logging',
    'concurrent.futures',
]

# ── Analysis ────────────────────────────────────────────────────
a = Analysis(
    ['main.py'],
    pathex=[],
    binaries=binaries_list,
    datas=datas_list,
    hiddenimports=hidden_imports,
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[
        'matplotlib', 'numpy', 'PIL', 'scipy', 'pandas',
        'IPython', 'notebook', 'jupyter', 'selenium',
        'pytest', 'unittest', 'pydoc', 'pytz',
        'customtkinter', 'tkinter', '_tkinter',
    ],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

# ── EXE ─────────────────────────────────────────────────────────
is_windows = sys.platform == 'win32'

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    [],
    name='edonish-auto',
    debug=False,
    bootloader_ignore_signals=False,
    strip=not is_windows,
    upx=not is_windows,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=False,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
    icon=None,
)

# ── macOS: .app bundle ──────────────────────────────────────────
if sys.platform == 'darwin':
    app = BUNDLE(
        exe,
        name='eDonish Auto.app',
        icon=None,
        bundle_identifier='tj.edonish.auto',
        info_plist={
            'CFBundleName': 'eDonish Auto',
            'CFBundleDisplayName': 'eDonish Auto',
            'CFBundleVersion': '3.17.0',
            'CFBundleShortVersionString': '3.17.0',
            'NSHighResolutionCapable': True,
            'LSMinimumSystemVersion': '10.13',
        },
    )
