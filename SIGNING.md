# Signing the helper downloads

**Windows:** the main route is now the Microsoft Store, which signs the package for free. See [MSSTORE.md](MSSTORE.md). The Azure Artifact Signing setup below is the alternative, for keeping a signed .exe on the website.

**Mac:** see [Mac: Developer ID signing and notarization](#mac-developer-id-signing-and-notarization) at the end.

## Windows: Azure Artifact Signing (optional)

Unsigned downloads make Windows show **"Windows protected your PC"** with
"Unknown publisher". Signing the helper with **Microsoft Artifact Signing**
(formerly "Trusted Signing", about US$10/month) puts **Techma** on that screen
and on the file's properties.

The build and signing run on GitHub (`.github/workflows/sign-windows.yml`).
You only do the one-time Azure setup below, then press a button.

### One-time setup (about 30 minutes, plus Microsoft's identity check)

#### 1. Azure account
1. Go to https://portal.azure.com and sign in (or create a free Microsoft account).
2. Create a **Pay-As-You-Go subscription** with a credit card. Artifact Signing
   doesn't work on free-trial credits.
3. In the subscription, open **Settings > Resource providers**, find
   **Microsoft.CodeSigning** and click **Register**.

#### 2. Signing account
1. In the portal search bar type **Artifact Signing** (it may still say
   "Trusted Signing") and click **Create**.
2. Pick the subscription, create a resource group (e.g. `techma-signing`),
   name the account (e.g. `techma`), choose a region such as **East US**, and
   pick the **Basic** plan (US$9.99/month, 5,000 signatures).
3. When it's created, open it and copy the **Account URI** from Overview, e.g.
   `https://eus.codesigning.azure.net/`. That's the endpoint.

#### 3. Identity validation (Microsoft checks who you are)
1. In the signing account, open **Access control (IAM) > Add role assignment**
   and give *yourself* the role **Artifact Signing Identity Verifier**.
2. Open **Identity validations > New identity > Public**.
3. Choose **Organization** and enter Techma's legal name exactly as registered
   in Quebec (Registraire des entreprises / NEQ), the business address,
   website and a business email on that domain. Upload documents if asked.
   - Microsoft has required organizations to have a verifiable history of a
     few years. If Techma is newer and gets rejected, choose **Individual**
     instead: the publisher then shows your own legal name rather than Techma.
4. Wait for the status to show **Completed**. This usually takes 1 to 7
   business days, and Microsoft may email you for more documents.

#### 4. Certificate profile
Open **Certificate profiles > Create > Public Trust**, name it (e.g.
`techma-public`) and select the completed identity validation.

#### 5. A login for GitHub
1. In the portal open **Microsoft Entra ID > App registrations > New
   registration**. Name it `github-signing` and click Register.
2. Copy the **Application (client) ID** and **Directory (tenant) ID**.
3. Open **Certificates & secrets > New client secret**, choose 24 months, and
   copy the secret **Value** right away (it's shown once).
4. Back in the signing account, **Access control (IAM) > Add role assignment**:
   give the `github-signing` app the role
   **Artifact Signing Certificate Profile Signer**.

#### 6. GitHub secrets
In this repo open **Settings > Secrets and variables > Actions > New
repository secret** and add these six:

| Secret | Value |
|---|---|
| `AZURE_TENANT_ID` | Directory (tenant) ID from step 5 |
| `AZURE_CLIENT_ID` | Application (client) ID from step 5 |
| `AZURE_CLIENT_SECRET` | The secret Value from step 5 |
| `AZURE_SIGNING_ENDPOINT` | Account URI from step 2, e.g. `https://eus.codesigning.azure.net/` |
| `AZURE_SIGNING_ACCOUNT` | Signing account name from step 2, e.g. `techma` |
| `AZURE_CERT_PROFILE` | Certificate profile name from step 4, e.g. `techma-public` |

### Each release

1. Open the **Actions** tab > **Build and sign Windows helper** > **Run workflow**.
2. Tick **publish** to put the signed `ProfileManagerHelperSetup.exe` straight
   into `docs/downloads/` (the website download). Leave it unticked to just
   download the signed files from the run's **Artifacts** section and test first.
3. The run fails on purpose if any file isn't validly signed.

To check a file yourself on Windows: right-click it > **Properties > Digital
Signatures**. It should list Techma (or your name).

### What to expect

- The helper (`profile-helper.exe`) is signed before it's packed into the
  setup, and then the setup is signed, so both show the publisher.
- SmartScreen can still show a warning for the first few weeks after a new
  certificate starts signing, until enough people have downloaded the file.
  It now names Techma, and the warning goes away on its own as downloads add up.
  Artifact Signing certificates renew automatically every few days, which is
  normal; reputation follows the verified publisher identity.
- Linux builds still come from `./build.sh`.

## Mac: Developer ID signing and notarization

`.github/workflows/mac-sign.yml` builds one installer (`ProfileManagerHelper-mac.pkg`)
for Intel and Apple Silicon Macs, signs the helper and the installer with your
Developer ID certificates, sends it to Apple for notarization and staples the
result. People double-click it, click Continue and Install, and it opens with
no warning. It's sold the same way (from the Chrome extension), not through
the Mac App Store, whose sandbox would block deleting Chrome profiles.

### One-time setup (no Mac needed)

You need the **Account Holder** role in the Apple Developer Program to create
Developer ID certificates.

1. **Make the certificate requests.** In Git Bash, from this repo:
   `APPLE_EMAIL=you@yourdomain ./tools/mac-cert.sh csr`. This writes
   `mac-signing/app.csr` and `mac-signing/installer.csr` plus their private keys.
2. **Create the certificates** at https://developer.apple.com/account/resources/certificates/add :
   - **Developer ID Application** (choose the *G2 Sub-CA*), upload `app.csr`,
     download it and save it as `mac-signing/app.cer`.
   - **Developer ID Installer** (G2 Sub-CA), upload `installer.csr`, save it as
     `mac-signing/installer.cer`.
3. **Package them for GitHub:** `./tools/mac-cert.sh p12 <a-strong-password>`.
   It writes `app.p12.base64.txt` and `installer.p12.base64.txt`.
4. **Create a notarization key** at https://appstoreconnect.apple.com/access/integrations/api
   (Team Keys > +, role *Developer*). Download the `.p8` file (only offered once),
   and note the **Key ID** and the **Issuer ID** shown above the list.
5. **Add the GitHub secrets** (repo Settings > Secrets and variables > Actions):

   | Secret | Value |
   |---|---|
   | `MAC_APP_CERT_P12` | contents of `mac-signing/app.p12.base64.txt` |
   | `MAC_INSTALLER_CERT_P12` | contents of `mac-signing/installer.p12.base64.txt` |
   | `MAC_CERT_PASSWORD` | the password from step 3 |
   | `APPLE_API_KEY_P8` | the whole text of the `.p8` file, including the BEGIN/END lines |
   | `APPLE_API_KEY_ID` | Key ID from step 4 |
   | `APPLE_API_ISSUER_ID` | Issuer ID from step 4 |

6. Delete the `mac-signing/` folder (or keep it somewhere private). The keys
   in it can sign software as Techma.

### Each release

**Actions > Build, sign and notarize Mac helper > Run workflow**. Tick
**publish** to replace `docs/downloads/ProfileManagerHelper-mac.pkg` on the
website. The extension's Mac download points at that file, so publish once
before uploading the next extension version to the Chrome Web Store.
