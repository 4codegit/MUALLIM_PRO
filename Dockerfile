# ── eDonish Auto — Multi-stage Docker Build ──────────────────────────
# Stage 1: GUI mode (X11 forwarding) 
# Stage 2: CLI/Headless mode (no display needed)

# ============================================
# BASE: Common dependencies
# ============================================
FROM python:3.12-slim AS base

WORKDIR /app

# System deps for CustomTkinter + X11
RUN apt-get update && apt-get install -y --no-install-recommends \
    libx11-6 \
    libxext6 \
    libxrender1 \
    libxrandr2 \
    libxinerama1 \
    libxcursor1 \
    libxi6 \
    libxtst6 \
    libgtk-3-0 \
    libgdk-pixbuf2.0-0 \
    libgl1 \
    libdbus-1-3 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libatspi2.0-0 \
    fonts-noto \
    fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# ============================================
# STAGE: GUI mode (with X11 forwarding)
# ============================================
FROM base AS gui

ENV DISPLAY=:0
ENV QT_X11_NO_MITSHM=1

ENTRYPOINT ["python3", "main.py"]

# ============================================
# STAGE: CLI/Headless mode (for Docker/servers)
# ============================================
FROM base AS cli

ENTRYPOINT ["python3", "main_cli.py"]

# ============================================
# DEFAULT: CLI mode
# ============================================
FROM cli AS latest
