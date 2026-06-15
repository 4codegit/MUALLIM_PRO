#!/bin/bash

set -e

VERSION=${1:-$(git describe --tags 2>/dev/null || echo "dev")}
echo "🔨 Building Edonish App for Linux (version: $VERSION)"

# Проверка Go
if ! command -v go &> /dev/null; then
    echo "❌ Go не установлен"
    exit 1
fi

# Установка зависимостей
echo "📦 Установка зависимостей..."
sudo apt-get update
sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev rpm

# Сборка бинарника
echo "📦 Сборка Linux бинарника..."
go mod tidy
GOOS=linux GOARCH=amd64 go build -o edonish-app-linux .

# Создание DEB пакета
echo "📦 Создание DEB пакета..."
mkdir -p deb/DEBIAN
mkdir -p deb/usr/bin
mkdir -p deb/usr/share/applications
mkdir -p deb/usr/share/icons/hicolor/256x256/apps

cat > deb/DEBIAN/control << EOF
Package: edonish-app
Version: $VERSION
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Edonish Team
Description: Edonish автоматизация десктопное приложение
EOF

cp edonish-app-linux deb/usr/bin/edonish-app
chmod +x deb/usr/bin/edonish-app

cat > deb/usr/share/applications/edonish-app.desktop << EOF
[Desktop Entry]
Name=Edonish App
Exec=/usr/bin/edonish-app
Icon=edonish-app
Type=Application
Categories=Education;
EOF

dpkg-deb --build deb edonish-app_${VERSION}_amd64.deb

# Создание RPM пакета
echo "📦 Создание RPM пакета..."
mkdir -p rpm/SOURCES
mkdir -p rpm/SPECS
mkdir -p rpm/BUILD

tar -czvf rpm/SOURCES/edonish-app-${VERSION}.tar.gz \
    --transform 's|^|edonish-app-${VERSION}/|' \
    main.go controller.go client/ ui/ go.mod go.sum

cat > rpm/SPECS/edonish-app.spec << EOF
Name: edonish-app
Version: $VERSION
Release: 1%{?dist}
Summary: Edonish автоматизация приложение
License: MIT
BuildArch: x86_64

%description
Десктопное приложение для автоматизации edonish.tj

%prep
%setup -q

%build
go mod tidy
GOOS=linux GOARCH=amd64 go build -o edonish-app .

%install
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/usr/share/applications
cp edonish-app %{buildroot}/usr/bin/
chmod +x %{buildroot}/usr/bin/edonish-app

cat > %{buildroot}/usr/share/applications/edonish-app.desktop << DESKTOP
[Desktop Entry]
Name=Edonish App
Exec=/usr/bin/edonish-app
Type=Application
Categories=Education;
DESKTOP

%files
/usr/bin/edonish-app
/usr/share/applications/edonish-app.desktop

%clean
rm -rf %{buildroot}
EOF

rpmbuild -bb rpm/SPECS/edonish-app.spec --define "_topdir $(pwd)/rpm"
cp rpm/RPMS/x86_64/edonish-app-${VERSION}-1.x86_64.rpm .

# Создание директории для релизов
mkdir -p release/linux

mv edonish-app-linux release/linux/
mv edonish-app_${VERSION}_amd64.deb release/linux/
mv edonish-app-${VERSION}-1.x86_64.rpm release/linux/

echo "✅ Готово! Файлы в release/linux/"
ls -lh release/linux/
