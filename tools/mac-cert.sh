#!/bin/bash
# Makes Apple Developer ID certificates usable by GitHub, without a Mac.
# Run it in Git Bash (Windows), WSL, Linux or macOS Terminal.
#
#   ./tools/mac-cert.sh csr            -> makes 2 private keys + 2 .csr files to upload to Apple
#   ./tools/mac-cert.sh p12 PASSWORD   -> after downloading the 2 .cer files from Apple into
#                                         this folder, makes the 2 .p12 files + base64 text for GitHub
#
# Everything is written to ./mac-signing/ (ignored by git). Keep that folder private
# and delete it once the GitHub secrets are saved.
set -e
D=mac-signing; mkdir -p $D; cd $D
case "$1" in
  csr)
    for n in app installer; do
      openssl req -new -newkey rsa:2048 -nodes -keyout $n.key -out $n.csr \
        -subj "/emailAddress=${APPLE_EMAIL:-you@example.com}/CN=Techma $n/C=CA"
    done
    echo "Upload app.csr for 'Developer ID Application' and installer.csr for 'Developer ID Installer'."
    echo "Save the downloaded certificates here as app.cer and installer.cer."
    ;;
  p12)
    [ -n "$2" ] || { echo "Usage: $0 p12 PASSWORD"; exit 1; }
    for n in app installer; do
      openssl x509 -inform DER -in $n.cer -out $n.pem
      # 3DES/SHA1 so macOS 'security import' accepts it
      openssl pkcs12 -export -inkey $n.key -in $n.pem -out $n.p12 -passout "pass:$2" \
        -certpbe PBE-SHA1-3DES -keypbe PBE-SHA1-3DES -macalg sha1 -name "$(openssl x509 -in $n.pem -noout -subject -nameopt multiline | sed -n 's/ *commonName *= //p')"
      base64 < $n.p12 | tr -d '\n' > $n.p12.base64.txt
      openssl x509 -in $n.pem -noout -subject
    done
    echo "Paste app.p12.base64.txt into MAC_APP_CERT_P12 and installer.p12.base64.txt into MAC_INSTALLER_CERT_P12."
    ;;
  *) sed -n 2,10p "$0" ;;
esac
