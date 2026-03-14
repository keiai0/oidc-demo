package oidc

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// sendBackChannelLogout は RP の backchannel_logout_uri に logout_token を POST する。
// 仕様参照: OIDC Back-Channel Logout 1.0 Section 2.5
func sendBackChannelLogout(client *model.Client, logoutToken string, httpClient *http.Client) {
	if client.BackchannelLogoutURI == nil || *client.BackchannelLogoutURI == "" {
		return
	}

	uri := *client.BackchannelLogoutURI

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// application/x-www-form-urlencoded で logout_token を送信
	body := url.Values{"logout_token": {logoutToken}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, strings.NewReader(body))
	if err != nil {
		log.Printf("[WARN] back-channel logout: failed to create request for %s (client_id=%s): %v", uri, client.ClientID, err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[WARN] back-channel logout: failed to send to %s (client_id=%s): %v", uri, client.ClientID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[WARN] back-channel logout: unexpected status %d from %s (client_id=%s)", resp.StatusCode, uri, client.ClientID)
		return
	}

	fmt.Printf("[INFO] back-channel logout: successfully notified %s (client_id=%s)\n", uri, client.ClientID)
}
