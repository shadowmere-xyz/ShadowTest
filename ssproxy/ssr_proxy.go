package ssproxy

import (
	"fmt"
	"net"
	"strconv"

	"github.com/metacubex/mihomo/transport/shadowsocks/core"
	"github.com/metacubex/mihomo/transport/shadowsocks/shadowstream"
	"github.com/metacubex/mihomo/transport/ssr/obfs"
	"github.com/metacubex/mihomo/transport/ssr/protocol"
	log "github.com/sirupsen/logrus"
)

// GetSSRProxyDetails tests an SSR proxy by using it on a call to wtfismyip.com
func GetSSRProxyDetails(address string, ipv4Only bool, timeout int) (WTFIsMyIPData, error) {
	escapedAddress := sanitizeAddress(address)

	config, err := parseSSRURL(escapedAddress)
	if err != nil {
		return WTFIsMyIPData{}, err
	}

	log.WithFields(log.Fields{
		"host":     config.Host,
		"port":     config.Port,
		"method":   config.Method,
		"protocol": config.Protocol,
		"obfs":     config.Obfs,
	}).Info("Testing SSR proxy")

	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return WTFIsMyIPData{}, fmt.Errorf("invalid port: %w", err)
	}

	serverAddr := net.JoinHostPort(config.Host, config.Port)

	// SSR cipher setup
	cipherName := config.Method
	if cipherName == "none" {
		cipherName = "dummy"
	}

	coreCiph, err := core.PickCipher(cipherName, nil, config.Password)
	if err != nil {
		return WTFIsMyIPData{}, fmt.Errorf("SSR cipher %s error: %w", config.Method, err)
	}

	var ivSize int
	var key []byte
	if cipherName == "dummy" {
		ivSize = 0
		key = core.Kdf(config.Password, 16)
	} else {
		streamCiph, ok := coreCiph.(*core.StreamCipher)
		if !ok {
			return WTFIsMyIPData{}, fmt.Errorf("%s is not a supported stream cipher for SSR", config.Method)
		}
		ivSize = streamCiph.IVSize()
		key = streamCiph.Key
	}

	log.WithFields(log.Fields{
		"cipher": cipherName,
		"ivSize": ivSize,
	}).Debug("SSR cipher initialized")

	// Set up obfuscation layer
	ssrObfs, obfsOverhead, err := obfs.PickObfs(config.Obfs, &obfs.Base{
		Host:   config.Host,
		Port:   port,
		Key:    key,
		IVSize: ivSize,
		Param:  config.ObfsParam,
	})
	if err != nil {
		return WTFIsMyIPData{}, fmt.Errorf("SSR obfs %s error: %w", config.Obfs, err)
	}

	log.WithFields(log.Fields{
		"obfs":     config.Obfs,
		"overhead": obfsOverhead,
	}).Debug("SSR obfuscation layer ready")

	// Set up protocol layer
	ssrProtocol, err := protocol.PickProtocol(config.Protocol, &protocol.Base{
		Key:      key,
		Overhead: obfsOverhead,
		Param:    config.ProtoParam,
	})
	if err != nil {
		return WTFIsMyIPData{}, fmt.Errorf("SSR protocol %s error: %w", config.Protocol, err)
	}

	log.WithFields(log.Fields{
		"protocol": config.Protocol,
	}).Debug("SSR protocol layer ready")

	// Build shadow function that chains: obfs → cipher → protocol
	shadow := func(c net.Conn) net.Conn {
		c = ssrObfs.StreamConn(c)
		c = coreCiph.StreamConn(c)
		var iv []byte
		if conn, ok := c.(*shadowstream.Conn); ok {
			iv, _ = conn.ObtainWriteIV()
		}
		c = ssrProtocol.StreamConn(c, iv)
		return c
	}

	return fetchProxyDetails(serverAddr, shadow, ipv4Only, timeout)
}
