// Where people download the helper program (served by GitHub Pages from docs/downloads).
const HELPER_DOWNLOADS = {
  win: "https://thisismasdouk.github.io/chrome-profile-manager/downloads/ProfileManagerHelperSetup.exe",
  mac: "https://thisismasdouk.github.io/chrome-profile-manager/downloads/ProfileManagerHelper-mac.pkg",
  linux: "https://thisismasdouk.github.io/chrome-profile-manager/downloads/ProfileManagerHelper-linux.tar.gz",
};

// Payments (ExtensionPay.com, built on Stripe). Put the extension ID you
// register on extensionpay.com here. While it is "", payments are off and
// the extension is free (handy for testing).
const EXTPAY_ID = "chrome-profile-manager";
// Free deletions before the one-time unlock is needed.
const FREE_DELETIONS = 3;
const PRICE_TEXT = "$4.99 one-time";

// Chrome Web Store page (fill in after the first upload). Used for the
// "leave a review" prompt shown once after a successful cleanup.
const STORE_URL = "https://chromewebstore.google.com/detail/nanlppofdmogahjbobcoaojkhlceackf";
