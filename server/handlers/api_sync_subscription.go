package handlers

import "net/http"

// Subscription proxies the caller's Premium status from the cloud.
func (h *Sync) Subscription(w http.ResponseWriter, r *http.Request) {
	out, err := h.mgr.Subscription(r.Context())
	proxyJSON(w, out, err)
}

// Subscribe proxies starting a Premium payment (returns the Duitku pay URL).
func (h *Sync) Subscribe(w http.ResponseWriter, r *http.Request) {
	out, err := h.mgr.SubscribePremium(r.Context())
	proxyJSON(w, out, err)
}
