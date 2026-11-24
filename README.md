# Profile Manager: bulk delete Chrome profiles

A Chrome extension that lists every Chrome profile on your computer and deletes many of them at once. Chrome normally makes you remove them one by one.

- **Website:** https://thisismasdouk.github.io/chrome-profile-manager/
- **Download the helper app:** [Windows](https://thisismasdouk.github.io/chrome-profile-manager/downloads/ProfileManagerHelperSetup.exe) · [Mac](https://thisismasdouk.github.io/chrome-profile-manager/downloads/ProfileManagerHelper-mac.zip) · [Linux](https://thisismasdouk.github.io/chrome-profile-manager/downloads/ProfileManagerHelper-linux.tar.gz)
- **Privacy policy:** https://thisismasdouk.github.io/chrome-profile-manager/privacy.html

## How it works
Chrome extensions can't delete files, so the extension talks to a small helper app through Chrome's native messaging.

The helper:
- reads Chrome's `Local State` file to list profiles;
- waits for Chrome to close, then removes the selected profiles' folders and their entries in `Local State`;
- reopens Chrome with every window and tab restored.

The profile you're using is always protected.

## Repository layout
- `extension/` is the Manifest V3 extension.
- `helper/` is the helper app (Go, no dependencies). `helper/setup` is the one-click Windows installer.
- `docs/` is the landing page and privacy policy (GitHub Pages).
- `store-assets/` and `STORE_LISTING.md` are the Chrome Web Store listing.
- `PUBLISHING.md` covers releasing, payments, code signing and store review.

## Build
Install Go 1.21+ and run `./build.sh`. Everything lands in `dist/`. To also allow the Web Store copy of the extension, run `STORE_ID=<item id> ./build.sh`.
