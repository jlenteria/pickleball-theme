// Package handlers provides HTTP request handlers for the pickleball theme plugin,
// including voucher redemption, settings management, authentication, and portal status.
package handlers

import (
	"net/http"
	"strings"

	sdkapi "sdk/api"
)

// voucherCodeField is the form field name for the theme's own voucher
// input -- distinct from the wifi-hotspot plugin's field, since this
// redemption path is owned by the theme.
const voucherCodeField = "voucher_code"

// PortalVoucherCtrl redeems a voucher code entered directly on the
// pickleball-theme portal, independently of the wifi-hotspot plugin's voucher
// page. The core Vouchers API looks up codes globally (not per-plugin), so
// this activates the same real vouchers, then connects the device.
func PortalVoucherCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := api.Http().Response()
		ctx := r.Context()

		if err := r.ParseForm(); err != nil {
			api.Logger().Error("pickleball-theme: voucher form parse error: " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "Invalid voucher code"), sdkapi.FlashMsgError)
			res.RedirectToPortal(w, r)
			return
		}

		// Voucher codes are case-insensitive; normalize to uppercase and validate format
		code := strings.ToUpper(strings.TrimSpace(r.FormValue(voucherCodeField)))
		if code == "" {
			res.FlashMsg(w, r, api.Translate("error", "Please enter a voucher code"), sdkapi.FlashMsgError)
			res.RedirectToPortal(w, r)
			return
		}

		// Basic validation: alphanumeric and dashes only
		if !isValidVoucherFormat(code) {
			res.FlashMsg(w, r, api.Translate("error", "Voucher code can only contain letters, numbers, and dashes"), sdkapi.FlashMsgError)
			res.RedirectToPortal(w, r)
			return
		}

		clnt, err := api.Http().GetClientDevice(r)
		if err != nil {
			res.FlashMsg(w, r, api.Translate("error", "Unable to identify your device"), sdkapi.FlashMsgError)
			res.RedirectToPortal(w, r)
			return
		}

		voucher, err := api.Vouchers().FindByCode(ctx, code)
		if err != nil {
			res.FlashMsg(w, r, api.Translate("error", "Voucher code not found"), sdkapi.FlashMsgError)
			res.RedirectToPortal(w, r)
			return
		}

		// Reject already-used vouchers (also guards the check-then-activate
		// race: a second request that found the same voucher is rejected
		// here).
		if voucher.ActivatedAt() != nil {
			res.FlashMsg(w, r, api.Translate("error", "This voucher has already been used"), sdkapi.FlashMsgError)
			res.RedirectToPortal(w, r)
			return
		}

		if _, err := api.Vouchers().Activate(ctx, sdkapi.ActivateVoucherParams{ID: voucher.ID(), Device: clnt}); err != nil {
			api.Logger().Error("pickleball-theme: voucher activation failed: " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "Unable to apply voucher. Please try again"), sdkapi.FlashMsgError)
			res.RedirectToPortal(w, r)
			return
		}

		// Activation only adds the session; grant internet if not already on.
		if !clnt.Session().IsConnected() {
			if err := clnt.Session().Connect(ctx, api.Translate("info", "Connected via voucher")); err != nil {
				api.Logger().Error("pickleball-theme: connect after voucher failed: " + err.Error())
				res.FlashMsg(w, r, api.Translate("error", "Voucher applied, but connection failed. Please try again"), sdkapi.FlashMsgError)
				res.RedirectToPortal(w, r)
				return
			}
		}

		res.FlashMsg(w, r, api.Translate("success", "Voucher <% .code %> applied — you're connected!", "code", code), sdkapi.FlashMsgSuccess)
		res.RedirectSuccess(w, r)
	}
}

// isValidVoucherFormat validates that a voucher code contains only allowed characters
func isValidVoucherFormat(code string) bool {
	if len(code) == 0 || len(code) > 50 {
		return false
	}
	for _, ch := range code {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return false
		}
	}
	return true
}
