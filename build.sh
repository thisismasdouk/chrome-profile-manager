#!/bin/bash
# Builds every download from source. Needs Go 1.21+ and zip.
#   ./build.sh                 -> build with the unpacked (developer) extension ID only
#   STORE_ID=abcdef... ./build.sh  -> also allow the Chrome Web Store extension ID
set -e
cd "$(dirname "$0")"
OUT=dist; rm -rf $OUT; mkdir -p $OUT/bin
LDF="-s -w"; [ -n "$STORE_ID" ] && LDF="$LDF -X main.storeExtensionID=$STORE_ID"
cd helper
for t in windows/amd64 windows/arm64 darwin/arm64 darwin/amd64 linux/amd64 linux/arm64; do
  o=${t%/*}; a=${t#*/}; e=""; [ $o = windows ] && e=.exe
  mkdir -p ../$OUT/bin/$o-$a
  CGO_ENABLED=0 GOOS=$o GOARCH=$a go build -trimpath -ldflags="$LDF" -o ../$OUT/bin/$o-$a/profile-helper$e .
done
# Windows one-click setup (contains the amd64 helper; also runs on ARM PCs)
mkdir -p setup/payload && cp ../$OUT/bin/windows-amd64/profile-helper.exe setup/payload/
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -H windowsgui" -o ../$OUT/ProfileManagerHelperSetup.exe ./setup
cd ..
# Mac download
M=$OUT/mac/ProfileManagerHelper; mkdir -p $M
cp -r $OUT/bin/darwin-arm64 $OUT/bin/darwin-amd64 $M/
cat > "$M/Install Profile Manager helper.command" <<'EOS'
#!/bin/bash
cd "$(dirname "$0")"
if [ "$(uname -m)" = "arm64" ]; then B=darwin-arm64; else B=darwin-amd64; fi
xattr -dr com.apple.quarantine . 2>/dev/null
chmod +x "$B/profile-helper"
"./$B/profile-helper" --install
echo; read -r -p "  Press Enter to close this window..." _
EOS
chmod +x "$M/Install Profile Manager helper.command" $M/*/profile-helper
(cd $OUT/mac && zip -qr -X ../ProfileManagerHelper-mac.zip ProfileManagerHelper)
# Linux download
L=$OUT/linux/ProfileManagerHelper; mkdir -p $L
cp -r $OUT/bin/linux-amd64 $OUT/bin/linux-arm64 $L/
cat > $L/install-linux.sh <<'EOS'
#!/bin/bash
# Usage: ./install-linux.sh [Chrome data folder]   (default ~/.config/google-chrome)
cd "$(dirname "$0")"
case "$(uname -m)" in aarch64|arm64) B=linux-arm64;; *) B=linux-amd64;; esac
chmod +x "$B/profile-helper"; "./$B/profile-helper" --install "$1"
EOS
chmod +x $L/install-linux.sh
tar -czf $OUT/ProfileManagerHelper-linux.tar.gz -C $OUT/linux ProfileManagerHelper
# Chrome Web Store upload: same extension without the developer "key" field
mkdir -p $OUT/webstore && cp -r extension/* $OUT/webstore/
python3 -c "import json,sys;p='$OUT/webstore/manifest.json';m=json.load(open(p));m.pop('key',None);json.dump(m,open(p,'w'),indent=2)"
(cd $OUT/webstore && zip -qr -X ../profile-manager-webstore.zip .)
rm -rf $OUT/mac $OUT/linux $OUT/webstore
echo "Built:"; ls -1 $OUT
