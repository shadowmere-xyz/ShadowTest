package ssproxy

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/shadowsocks/go-shadowsocks2/core"
)

// GetShadowsocksProxyDetails tests a shadowsocks proxy by using it on a call to wtfismyip.com
func GetShadowsocksProxyDetails(address string, ipv4Only bool, timeout int) (WTFIsMyIPData, error) {
	escapedAddress := sanitizeAddress(address)

	addr, cipher, password, err := parseURL(escapedAddress)
	if err != nil {
		return WTFIsMyIPData{}, err
	}

	ciph, err := core.PickCipher(cipher, []byte{}, password)
	if err != nil {
		return WTFIsMyIPData{}, err
	}

	return fetchProxyDetails(addr, ciph.StreamConn, ipv4Only, timeout)
}

func extractCredentialsFromBase64(address string) (string, error) {
	atIdx := strings.Index(address, "@")
	if atIdx < 5 {
		return "", fmt.Errorf("invalid SS address: missing or misplaced '@'")
	}
	key := address[5:atIdx]

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
