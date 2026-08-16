# PearDesk - Makefile
# Requires: Go 1.21+, fyne CLI (go install fyne.io/fyne/v2/cmd/fyne@latest)

APP_NAME    := peardesk
APP_ID      := com.peardesk.app
APP_VERSION := 1.0.0
MAIN_PKG    := ./cmd/peardesk

# Icon
ICON        := assets/icon.png

# Output dirs
DIST        := dist
LINUX_DIR   := $(DIST)/linux
WIN_DIR     := $(DIST)/windows
MAC_DIR     := $(DIST)/macos

.PHONY: all linux windows macos appimage package clean tidy

all: linux

tidy:
	go mod tidy

# ─── Linux ───────────────────────────────────────────────────────────────────
linux: tidy
	@mkdir -p $(LINUX_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
	go build -ldflags="-s -w -X main.Version=$(APP_VERSION)" \
	    -o $(LINUX_DIR)/$(APP_NAME) $(MAIN_PKG)
	@cp $(ICON) $(LINUX_DIR)/
	@echo "✓ Linux binary: $(LINUX_DIR)/$(APP_NAME)"

# ─── AppImage ─────────────────────────────────────────────────────────────────
# Requires: fyne CLI + appimagetool
appimage: linux
	@mkdir -p $(LINUX_DIR)/AppDir/usr/bin $(LINUX_DIR)/AppDir/usr/share/icons
	@cp $(LINUX_DIR)/$(APP_NAME) $(LINUX_DIR)/AppDir/usr/bin/
	@cp $(ICON) $(LINUX_DIR)/AppDir/$(APP_NAME).png
	@cp $(ICON) $(LINUX_DIR)/AppDir/usr/share/icons/$(APP_NAME).png
	@echo "[Desktop Entry]" > $(LINUX_DIR)/AppDir/$(APP_NAME).desktop
	@echo "Name=PearDesk" >> $(LINUX_DIR)/AppDir/$(APP_NAME).desktop
	@echo "Exec=$(APP_NAME)" >> $(LINUX_DIR)/AppDir/$(APP_NAME).desktop
	@echo "Icon=$(APP_NAME)" >> $(LINUX_DIR)/AppDir/$(APP_NAME).desktop
	@echo "Type=Application" >> $(LINUX_DIR)/AppDir/$(APP_NAME).desktop
	@echo "Categories=Network;" >> $(LINUX_DIR)/AppDir/$(APP_NAME).desktop
	@echo "#!/bin/sh" > $(LINUX_DIR)/AppDir/AppRun
	@echo 'exec "$${APPDIR}/usr/bin/$(APP_NAME)" "$$@"' >> $(LINUX_DIR)/AppDir/AppRun
	@chmod +x $(LINUX_DIR)/AppDir/AppRun
	ARCH=x86_64 appimagetool $(LINUX_DIR)/AppDir $(DIST)/PearDesk-$(APP_VERSION)-x86_64.AppImage
	@echo "✓ AppImage: $(DIST)/PearDesk-$(APP_VERSION)-x86_64.AppImage"

# ─── Windows ─────────────────────────────────────────────────────────────────
# Requires: mingw-w64 cross-compiler
# On Ubuntu: sudo apt install gcc-mingw-w64-x86-64
windows: tidy
	@mkdir -p $(WIN_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
	CC=x86_64-w64-mingw32-gcc \
	go build -ldflags="-s -w -H=windowsgui -X main.Version=$(APP_VERSION)" \
	    -o $(WIN_DIR)/$(APP_NAME).exe $(MAIN_PKG)
	@cp $(ICON) $(WIN_DIR)/
	@echo "✓ Windows exe: $(WIN_DIR)/$(APP_NAME).exe"

# ─── macOS ───────────────────────────────────────────────────────────────────
# Must be run on macOS or with macOS cross-compiler
macos: tidy
	@mkdir -p $(MAC_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
	go build -ldflags="-s -w -X main.Version=$(APP_VERSION)" \
	    -o $(MAC_DIR)/$(APP_NAME) $(MAIN_PKG)
	@$(MAKE) _macos_bundle
	@echo "✓ macOS app: $(MAC_DIR)/PearDesk.app"

_macos_bundle:
	@mkdir -p $(MAC_DIR)/PearDesk.app/Contents/MacOS
	@mkdir -p $(MAC_DIR)/PearDesk.app/Contents/Resources
	@mv $(MAC_DIR)/$(APP_NAME) $(MAC_DIR)/PearDesk.app/Contents/MacOS/
	@cp $(ICON) $(MAC_DIR)/PearDesk.app/Contents/Resources/icon.png
	@echo '<?xml version="1.0"?>' > $(MAC_DIR)/PearDesk.app/Contents/Info.plist
	@echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"' >> $(MAC_DIR)/PearDesk.app/Contents/Info.plist
	@echo '  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> $(MAC_DIR)/PearDesk.app/Contents/Info.plist
	@echo '<plist version="1.0"><dict>' >> $(MAC_DIR)/PearDesk.app/Contents/Info.plist
	@echo '  <key>CFBundleName</key><string>PearDesk</string>' >> $(MAC_DIR)/PearDesk.app/Contents/Info.plist
	@echo '  <key>CFBundleExecutable</key><string>$(APP_NAME)</string>' >> $(MAC_DIR)/PearDesk.app/Contents/Info.plist
	@echo '  <key>CFBundleIdentifier</key><string>$(APP_ID)</string>' >> $(MAC_DIR)/PearDesk.app/Contents/Info.plist
	@echo '  <key>CFBundleVersion</key><string>$(APP_VERSION)</string>' >> $(MAC_DIR)/PearDesk.app/Contents/Info.plist
	@echo '  <key>CFBundleIconFile</key><string>icon</string>' >> $(MAC_DIR)/PearDesk.app/Contents/Info.plist
	@echo '  <key>NSHighResolutionCapable</key><true/>' >> $(MAC_DIR)/PearDesk.app/Contents/Info.plist
	@echo '</dict></plist>' >> $(MAC_DIR)/PearDesk.app/Contents/Info.plist

# ─── Package (zip of sources) ─────────────────────────────────────────────────
package:
	@mkdir -p $(DIST)
	zip -r $(DIST)/PearDesk-$(APP_VERSION)-sources.zip . \
	    --exclude "dist/*" --exclude ".git/*" --exclude "*.tsbuildinfo"
	@echo "✓ Sources: $(DIST)/PearDesk-$(APP_VERSION)-sources.zip"

clean:
	rm -rf $(DIST)
