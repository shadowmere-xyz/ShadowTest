package ssproxy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/phayes/freeport"
	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/socks"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
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

func IsWTFIsMyIpOffline() bool {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get("https://wtfismyip.com/test")
	if err != nil || resp.StatusCode != http.StatusOK {
		return true
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Errorf("impossible to close response body %v", err)
		}
	}(resp.Body)

	return false
}

// GetShadowsocksProxyDetails tests a shadowsocks proxy by using it on a call to wtfismyip.com
func GetShadowsocksProxyDetails(address string, ipv4Only bool, timeout int) (WTFIsMyIPData, error) {
	escapedAddress := strings.Replace(address, "\n", "", -1)
	escapedAddress = strings.Replace(escapedAddress, "\r", "", -1)

	addr, cipher, password, err := parseURL(escapedAddress)
	if err != nil {
		return WTFIsMyIPData{}, err
	}

	ciph, err := core.PickCipher(cipher, []byte{}, password)
	if err != nil {
		return WTFIsMyIPData{}, err
	}
	port, err := freeport.GetFreePort()
	if err != nil {
		return WTFIsMyIPData{}, err
	}
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", port)

	ready := make(chan bool, 1)

	go ListenForOneConnection(proxyAddr, addr, ciph.StreamConn, ready, func(c net.Conn) (socks.Addr, error) { return socks.Handshake(c) })
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return WTFIsMyIPData{}, err
	}

	httpTransport := &http.Transport{}
	timeoutDuration := time.Duration(timeout) * time.Second
	httpClient := &http.Client{Transport: httpTransport, Timeout: timeoutDuration}
	httpTransport.Dial = dialer.Dial
	<-ready
	wtfismyipURL := "https://wtfismyip.com/json"
	if ipv4Only {
		wtfismyipURL = "https://ipv4.wtfismyip.com/json"
	}
	request, err := http.NewRequest("GET", wtfismyipURL, nil)
	if err != nil {
		return WTFIsMyIPData{}, err
	}
	request.Header.Set("User-Agent", "ShadowTest")
	response, err := httpClient.Do(request)
	if err != nil {
		return WTFIsMyIPData{}, err
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return WTFIsMyIPData{}, err
	}

	data := WTFIsMyIPData{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return WTFIsMyIPData{}, err
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
	if !strings.HasPrefix(s, "ss://") {
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
