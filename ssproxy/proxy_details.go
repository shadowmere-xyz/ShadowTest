package ssproxy

import (
	"ShadowTest/offlinecache"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/shadowsocks/go-shadowsocks2/socks"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

// WTFIsMyIPData is a data representation with the same structure returned by https://wtfismyip.com/json
type WTFIsMyIPData struct {
	YourFuckingIPAddress   string `json:"YourFuckingIPAddress"`
	YourFuckingLocation    string `json:"YourFuckingLocation"`
	YourFuckingHostname    string `json:"YourFuckingHostname"`
	YourFuckingISP         string `json:"YourFuckingISP"`
	YourFuckingTorExit     bool   `json:"YourFuckingTorExit"`
	YourFuckingCountryCode string `json:"YourFuckingCountryCode"`
	YourFuckingCity        string `json:"YourFuckingCity"`
	YourFuckingCountry     string `json:"YourFuckingCountry"`
}

// IsWTFIsMyIpOffline checks if wtfismyip.com is offline by making a request to a test URL
func IsWTFIsMyIpOffline(offlineCache *offlinecache.SafeIsOfflineCache, testURL string) bool {
	if !offlineCache.Expired() && !offlineCache.IsZero() {
		return offlineCache.GetIsOfflineFromCache()
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(testURL)

	if err != nil || resp.StatusCode != http.StatusOK {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		log.WithFields(map[string]interface{}{
			"err":    err,
			"status": status,
		}).Error("Error checking wtfismyip.com status. Setting the cache to offline.")
		if err != nil {
			sentry.CaptureException(err)
		}
		offlineCache.SetIsOfflineToCache(true, 5*time.Minute)
	} else {
		offlineCache.SetIsOfflineToCache(false, 5*time.Minute)
	}

	if resp != nil && resp.Body != nil {
		err := resp.Body.Close()
		if err != nil {
			log.Errorf("impossible to close response body %v", err)
			sentry.CaptureException(err)
		}
	}

	return offlineCache.GetIsOfflineFromCache()
}

// TestProxy tests a proxy by dispatching to the appropriate handler based on URL scheme.
func TestProxy(address string, ipv4Only bool, timeout int) (WTFIsMyIPData, error) {
	if strings.HasPrefix(address, "ssr://") {
		return GetSSRProxyDetails(address, ipv4Only, timeout)
	}
	return GetShadowsocksProxyDetails(address, ipv4Only, timeout)
}

func sanitizeAddress(address string) string {
	s := strings.ReplaceAll(address, "\n", "")
	return strings.ReplaceAll(s, "\r", "")
}

// fetchProxyDetails spins up a local SOCKS5 relay to the given server through
// the provided shadow function, then queries wtfismyip.com to retrieve the
// exit-node details. Both SS and SSR proxy testers delegate to this after
// setting up their protocol-specific cipher/obfs/protocol layers.
func fetchProxyDetails(serverAddr string, shadow func(net.Conn) net.Conn, ipv4Only bool, timeout int) (WTFIsMyIPData, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return WTFIsMyIPData{}, err
	}
	defer func(l net.Listener) {
		err := l.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			log.Errorf("failed to close listener: %v", err)
		}
	}(l)
	proxyAddr := l.Addr().String()

	go ListenForOneConnection(l, serverAddr, shadow, func(c net.Conn) (socks.Addr, error) { return socks.Handshake(c) })
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return WTFIsMyIPData{}, err
	}

	httpTransport := &http.Transport{
		DisableKeepAlives: true,
	}
	defer httpTransport.CloseIdleConnections()

	timeoutDuration := time.Duration(timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	httpClient := &http.Client{Transport: httpTransport, Timeout: timeoutDuration}
	httpTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.(proxy.ContextDialer).DialContext(ctx, network, addr)
	}
	wtfismyipURL := "https://wtfismyip.com/json"
	if ipv4Only {
		wtfismyipURL = "https://ipv4.wtfismyip.com/json"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, wtfismyipURL, nil)
	if err != nil {
		return WTFIsMyIPData{}, err
	}
	request.Header.Set("User-Agent", "ShadowTest")
	response, err := httpClient.Do(request)
	if err != nil {
		return WTFIsMyIPData{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Errorf("impossible to close response body %v", err)
			sentry.CaptureException(err)
		}
	}(response.Body)

	var data WTFIsMyIPData
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return WTFIsMyIPData{}, err
	}
	return data, nil
}
