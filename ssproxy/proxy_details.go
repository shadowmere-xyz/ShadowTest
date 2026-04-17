package ssproxy

import (
	"ShadowTest/offlinecache"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/socks"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

type IPInfo struct {
	IPAddress   string `json:"IPAddress"`
	Location    string `json:"Location"`
	Hostname    string `json:"Hostname"`
	ISP         string `json:"ISP"`
	TorExit     bool   `json:"TorExit"`
	CountryCode string `json:"CountryCode"`
	City        string `json:"City"`
	Country     string `json:"Country"`
}

func IsIPInfoOffline(offlineCache *offlinecache.SafeIsOfflineCache, testURL string) bool {
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
		}).Error("Error checking ip.r4bbit.net status. Setting the cache to offline.")
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

func GetShadowsocksProxyDetails(address string, ipv4Only bool, timeout int) (IPInfo, error) {
	escapedAddress := strings.ReplaceAll(address, "\n", "")
	escapedAddress = strings.ReplaceAll(escapedAddress, "\r", "")

	addr, cipher, password, err := parseURL(escapedAddress)
	if err != nil {
		return IPInfo{}, err
	}

	ciph, err := core.PickCipher(cipher, []byte{}, password)
	if err != nil {
		return IPInfo{}, err
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return IPInfo{}, err
	}
	defer func(l net.Listener) {
		err := l.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			log.Errorf("failed to close listener: %v", err)
		}
	}(l)
	proxyAddr := l.Addr().String()

	go ListenForOneConnection(l, addr, ciph.StreamConn, func(c net.Conn) (socks.Addr, error) { return socks.Handshake(c) })
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return IPInfo{}, err
	}

	httpTransport := &http.Transport{
		DisableKeepAlives: true,
	}
	defer httpTransport.CloseIdleConnections()

	timeoutDuration := time.Duration(timeout) * time.Second
	httpClient := &http.Client{Transport: httpTransport, Timeout: timeoutDuration}
	httpTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.(proxy.ContextDialer).DialContext(ctx, network, addr)
	}
	ipinfoURL := "https://ip.r4bbit.net/json"
	if ipv4Only {
		ipinfoURL = "https://ipv4.r4bbit.net/json"
	}
	request, err := http.NewRequest("GET", ipinfoURL, nil)
	if err != nil {
		return IPInfo{}, err
	}
	request.Header.Set("User-Agent", "ShadowTest")
	response, err := httpClient.Do(request)
	if err != nil {
		return IPInfo{}, err
	}
	defer func() {
		if response.Body != nil {
			closeErr := response.Body.Close()
			if closeErr != nil {
				log.Errorf("failed to close response body: %v", closeErr)
				sentry.CaptureException(closeErr)
			}
		}
	}()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return IPInfo{}, err
	}

	data := IPInfo{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return IPInfo{}, err
	}
	return data, nil
}

func extractCredentialsFromBase64(address string) (string, error) {
	key := address[5:strings.Index(address, "@")]

	creds, err := base64DecodeStripped(key)
	if err != nil {
		return "", err
	}

	creds = strings.ReplaceAll(creds, "/", "%2F")

	return strings.ReplaceAll(address, key, creds), nil
}

func base64DecodeStripped(s string) (string, error) {
	if i := len(s) % 4; i != 0 {
		s += strings.Repeat("=", 4-i)
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	return string(decoded), err
}

func parseURL(s string) (addr, cipher, password string, err error) {
	if !strings.HasPrefix(s, "ss://") || !strings.Contains(s, "@") {
		return "", "", "", fmt.Errorf("address %s does not seem to be a shadowsocks SIP002 address", s)
	}
	s, err = extractCredentialsFromBase64(s)
	if err != nil {
		return
	}

	u, err := url.Parse(s)
	if err != nil {
		return
	}

	addr = u.Host
	if u.User != nil {
		cipher = u.User.Username()
		password, _ = u.User.Password()
	}
	return
}
