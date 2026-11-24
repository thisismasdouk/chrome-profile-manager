// Where people download the helper program (GitHub Releases of the public repo).
// "latest/download" always points to the newest release, so these never need changing.
const HELPER_DOWNLOADS = {
  win: "https://github.com/thisismasdouk/chrome-profile-manager/releases/latest/download/ProfileManagerHelperSetup.exe",
  mac: "https://github.com/thisismasdouk/chrome-profile-manager/releases/latest/download/ProfileManagerHelper-mac.zip",
  linux: "https://github.com/thisismasdouk/chrome-profile-manager/releases/latest/download/ProfileManagerHelper-linux.tar.gz",
};

// Payments (ExtensionPay.com, built on Stripe). Put the extension ID you
// register on extensionpay.com here. While it is "", payments are off and
// the extension is free (handy for testing).
const EXTPAY_ID = "";
const TRIAL_DAYS = 7;
const PRICE_TEXT = "$1.99/month";

// Chrome Web Store page (fill in after the first upload). Used for the
// "leave a review" prompt shown once after a successful cleanup.
const STORE_URL = "";
