# Windows: publishing the helper on the Microsoft Store

On the Microsoft Store, Microsoft signs the package, so people never see
"Windows protected your PC". The Windows download button in the extension
then opens the Store page instead of downloading an .exe.

## How it works

- `.github/workflows/msstore.yml` builds `ProfileManagerHelper-1.0.N.0.msix`
  (N = the run number, so every upload has a higher version).
- The package contains the same one-click setup as the website download.
  People click **Get**, then **Open**, then **OK**. The setup copies the helper
  to `%LOCALAPPDATA%\ChromeProfileManager` and registers it with Chrome, exactly
  like the .exe did.
- The package turns off MSIX file and registry redirection
  (`unvirtualizedResources`), because Chrome has to see the helper and its
  registry entry. That is a *restricted capability*: Partner Center asks you to
  explain it (text below).
- The $4.99 unlock stays inside the Chrome extension (ExtensionPay/Stripe). The
  Store app itself is **free** and sells nothing, so the Store's purchase rules
  don't apply to it.

## One-time setup in Partner Center

1. Go to https://partner.microsoft.com/dashboard and sign up as an
   **individual** developer (free) or a **company** (Techma). A company
   account needs Microsoft to verify the business, which takes longer.
2. **Apps and games > New product > MSIX or PWA app**, and reserve the name
   **Profile Manager Helper for Chrome** (or another free name).
3. Open the app > **Product management > Product identity** and copy these into
   GitHub (repo **Settings > Secrets and variables > Actions > Variables** tab):

   | Variable | Partner Center field |
   |---|---|
   | `MSSTORE_IDENTITY_NAME` | Package/Identity/Name |
   | `MSSTORE_PUBLISHER` | Package/Identity/Publisher (starts with `CN=`) |
   | `MSSTORE_PUBLISHER_DISPLAY_NAME` | Package/Properties/PublisherDisplayName |
   | `MSSTORE_DISPLAY_NAME` | The reserved name |

4. Note the **Store ID** (12 characters, e.g. `9NXXXXXXXXXX`) for the extension link.

## Each release

1. GitHub **Actions > Microsoft Store package > Run workflow**. Download the
   `.msix` from the run's **Artifacts** (unzip it first).
2. In Partner Center, **Start your submission** and fill in:
   - **Pricing and availability:** Free, all markets.
   - **Properties:** category *Utilities & tools*. Privacy policy URL:
     https://thisismasdouk.github.io/chrome-profile-manager/privacy.html
   - **Age ratings:** complete the questionnaire (no objectionable content).
   - **Packages:** upload the `.msix`.
   - **Store listings (English):** description from below, at least one
     screenshot (`store-assets/screenshot-*.png`) and the logo
     `store-assets/msstore-logo-300.png`.
   - **Submission options > Notes for certification**: paste the text below.
3. Submit. Certification usually takes 1 to 3 business days.

### Store description (paste)

> Free companion for the **Profile Manager** Chrome extension. Chrome doesn't
> let extensions delete profile folders, so this small helper does it for the
> extension: it lists your Chrome profiles and removes the ones you select.
>
> 1. Install Profile Manager from the Chrome Web Store.
> 2. Get this helper, click Open, then OK.
> 3. Back in Chrome, pick the profiles to delete.
>
> The helper only talks to the Profile Manager extension, works only on your
> own Windows account and never sends data anywhere.

### Notes for certification (paste)

> This app is the companion (Chrome "native messaging host") for the Profile
> Manager Chrome extension: https://chromewebstore.google.com/detail/nanlppofdmogahjbobcoaojkhlceackf
> When opened, it asks for confirmation, then copies its helper program to
> %LOCALAPPDATA%\ChromeProfileManager and registers it under
> HKCU\Software\Google\Chrome\NativeMessagingHosts. Google Chrome (outside the
> package) must be able to read both, so the package uses
> unvirtualizedResources to turn off file system and registry write
> virtualization. runFullTrust is needed because it's a Win32 app. It downloads
> nothing; everything is inside the package. To test: install Google Chrome,
> open it once, open this app and click OK. It reports success, and
> Settings > Apps lists "Profile Manager helper (for Chrome)", which removes it.

## After the Store approves it

Point the extension at the Store: in `extension/config.js` set
`HELPER_DOWNLOADS.win` to `https://apps.microsoft.com/detail/<Store ID>`, then
upload the new extension zip to the Chrome Web Store.
