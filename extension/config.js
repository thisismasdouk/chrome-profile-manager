// Where people download the helper program (served by GitHub Pages from docs/downloads).
const HELPER_DOWNLOADS = {
  win: "https://thisismasdouk.github.io/chrome-profile-manager/downloads/ProfileManagerHelperSetup.exe",
  mac: "https://thisismasdouk.github.io/chrome-profile-manager/downloads/ProfileManagerHelper-mac.zip",
  linux: "https://thisismasdouk.github.io/chrome-profile-manager/downloads/ProfileManagerHelper-linux.tar.gz",
};

// Payments (ExtensionPay.com, built on Stripe). Put the extension ID you
// register on extensionpay.com here. While it is "", payments are off and
// the extension is free (handy for testing).
const EXTPAY_ID = "chrome-profile-manager";
const TRIAL_DAYS = 7;
const PRICE_TEXT = "$1.99/month";

// Chrome Web Store page (fill in after the first upload). Used for the
// "leave a review" prompt shown once after a successful cleanup.
const STORE_URL = "";
