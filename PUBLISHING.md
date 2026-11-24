# Publishing Profile Manager on the Chrome Web Store

People install the extension from the Chrome Web Store. The first time they open it, they see **"Download the helper for Windows"** (or Mac). They run that one file and click OK. The page notices the helper within a few seconds and shows their profiles. Nobody has to find a folder or load anything by hand.

The Web Store can only host the extension. The helper program has to be downloadable from somewhere else. The steps below cover that.

## What's in this kit

| Item | Purpose |
|---|---|
| `extension/` | The extension source. `config.js` holds the download links, payment ID and store link |
| `helper/` | The helper app source (Go) and the Windows setup program |
| `docs/` | Landing page and privacy policy, served free by GitHub Pages |
| `store-assets/` | Store screenshots and promo images |
| `STORE_LISTING.md` | Store name, descriptions and category to copy and paste, plus tips for ranking |
| `ready-to-upload/` | Built files: the Web Store zip and the helper downloads |

## Step by step

1. **GitHub repo (free).** Create the public repo `thisismasdouk/chrome-profile-manager`. The helper downloads live in `docs/downloads/` and are served by GitHub Pages; the extension links to them.
2. **Landing page (free).** In the repo, go to Settings → Pages, choose "Deploy from a branch", branch `main`, folder `/docs`. The site appears at https://thisismasdouk.github.io/chrome-profile-manager/ and includes the privacy policy the store asks for.
3. **Payments ($4.99 one-time, 3 free deletions).**
   1. Sign up at https://extensionpay.com and connect a Stripe account (Stripe works in Canada). ExtensionPay and Stripe each take a small cut of every payment, so check their current fees.
   2. Register the extension in ExtensionPay with an ID such as `chrome-profile-manager`, and create a **$4.99 one-time ("Once - Lifetime")** plan. (Done: the extension is registered as `chrome-profile-manager`.)
   3. Put that ID in `extension/config.js` → `EXTPAY_ID`. While it's empty, the extension is free, which is handy for testing.
   4. Pricing is built in. Everyone can delete up to 3 profiles for free (counted in Chrome's synced storage, so reinstalling doesn't reset it). After that, the Delete button changes to "Unlock unlimited · $4.99 one-time", which opens the ExtensionPay checkout. Once someone has paid, the extension shows "Unlimited ✓".
4. **Chrome Web Store developer account.** Register at https://chrome.google.com/webstore/devconsole (US$5 one-time fee). Upload `ready-to-upload/profile-manager-webstore.zip` as a **draft** to get the extension's permanent **Item ID**.
5. **Rebuild with the store ID.** The helper only answers extensions it knows, so run `STORE_ID=<item id> ./build.sh`, then copy the new helper files from `dist/` into `docs/downloads/` and push.
6. **Add the store link** to `STORE_URL` in `extension/config.js` (this enables the "leave a review" prompt) and in `docs/index.html` (this enables the Add to Chrome buttons). The link looks like `https://chromewebstore.google.com/detail/<item id>`.
7. **Build once more and upload** the new `profile-manager-webstore.zip` to the draft. Fill in the listing from `STORE_LISTING.md`, add the images from `store-assets/`, and in the store's pricing section say the extension "contains in-app purchases". Then submit for review.

## Code signing (strongly recommended before going public)

Without a signature, Windows shows a blue **"Windows protected your PC"** box, and macOS refuses to open the helper until the user right-clicks and chooses Open. Many people stop at that point, and some antivirus tools flag unsigned downloads.

**Windows**
- **Cheapest: Microsoft Artifact Signing** (formerly "Trusted Signing"). It's a monthly Azure subscription, roughly US$10/month. Microsoft verifies your identity. Individual developers in the US and Canada have been eligible, but check the current rules. Sign with `signtool` or the GitHub Action Microsoft provides.
- **Alternative: an OV code-signing certificate** from a provider such as Sectigo, DigiCert or SSL.com, roughly US$200–500/year. The key now has to live on a hardware token or in the provider's cloud signing service.
- **What to sign:** `ProfileManagerHelperSetup.exe`. Also sign `profile-helper.exe` before it's embedded, by running build.sh, signing `helper/setup/payload/profile-helper.exe`, then re-running only the setup build line.
- Even signed downloads can show a SmartScreen warning for the first few weeks, until enough people have installed them. That goes away by itself.

**Mac**
- Join the **Apple Developer Program** (US$99/year) and create a **Developer ID Application** certificate.
- Sign with `codesign --options runtime`, then notarize with `xcrun notarytool submit … --wait`. For the smoothest experience, ship a signed and notarized `.pkg` installer instead of the zip. That last step needs a Mac.

## Web Store review: what they will ask for

- **Single purpose:** "Lets users view their local Chrome profiles and delete several of them at once."
- **Permission justifications:**
  - `nativeMessaging`: "Chrome does not allow extensions to delete profile folders. The extension talks to a companion helper program the user installs, which lists profiles from Chrome's Local State file and deletes the ones the user selects."
  - `storage`: "Stores a random ID used to recognise which profile the user is in, so that profile can never be deleted."
- **Disclose payment:** mention the 3 free deletions and the $4.99 one-time unlock in the description (already in `STORE_LISTING.md`).
- **Disclose the helper up front.** The first lines of the store description must say a free companion program is required and where it comes from. Reviewers reject extensions that hide required downloads.
- **Privacy policy URL:** https://thisismasdouk.github.io/chrome-profile-manager/privacy.html (it's in `docs/privacy.html`).
- **Data-use form:** declare "Personally identifiable information" (emails) and "Website content: no". Certify that data isn't sold or transferred and is used only for the single purpose.
- **Listing assets:** a 128×128 icon (included in `extension/icons`), at least one 1280×800 screenshot, and optionally a 440×280 promo tile.
- **Review time** is usually a few days, and longer for extensions that use `nativeMessaging`.

## Updating later

- **Extension changes:** bump `"version"` in `extension/manifest.json`, rebuild, and upload a new zip.
- **Helper changes:** bump `version` in `helper/main.go`, rebuild with the same `STORE_ID`, and attach the files to a new GitHub release. Existing users keep their old helper until they reinstall it. If that matters, the extension can compare versions and show an "Update the helper" button. That's a small change.

## Building from source

Install Go (https://go.dev/dl), then run `./build.sh` from this folder (on Windows, use Git Bash or WSL). Everything lands in `dist/`.
