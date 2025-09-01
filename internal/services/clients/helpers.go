package clients

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
)

func validateCallbackURL(callbackURL string) bool {
	if callbackURL == "" {
		return false
	}

	url, err := url.Parse(callbackURL)
	if err != nil {
		return false
	}

	if !slices.Contains([]string{"http", "https"}, url.Scheme) {
		return false
	}

	if url.Host == "" {
		return false
	}

	return true
}

func validateExternalIP(backendIP string) bool {
	if backendIP == "" {
		return false
	}

	ip := net.ParseIP(backendIP)
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() {
		return false
	}

	return true
}

func getPublicAddress(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ip.pn", http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Add("User-Agent", "curl/8.7.1")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	type IP struct {
		Query string `json:"query"`
	}
	var resp IP
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	return resp.Query, nil
}

func validateDomain(ctx context.Context, domain string) bool { //nolint:unused
	dom, err := url.Parse(domain)
	if err != nil {
		return false
	}
	addrs, err := net.DefaultResolver.LookupHost(ctx, dom.Host)
	if err != nil {
		return false
	}
	if len(addrs) == 0 {
		return false
	}
	return true
}
