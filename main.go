package main

import (
	"ShadowTest/ssproxy"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	_ "embed"
	"net"
	"net/url"
	"strings"

	"github.com/phayes/freeport"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/socks"
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
}

type proxyJson struct {
	Address string `json:"address"`
}

//go:embed index.html
var indexFile embed.FS

func main() {
	sentryDsn := os.Getenv("SENTRY_DSN")
	if sentryDsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryDsn,
			TracesSampleRate: 0.1,
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
	} else {
		log.Warn("SENTRY_DSN was not provided. Running without sentry.")
	}

	http.HandleFunc("/v1/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method is not supported.", http.StatusNotFound)
			return
		}
		defer r.Body.Close()
		address := ""
		p := proxyJson{}
		if r.Header.Get("Content-Type") == "application/json" {
			err := json.NewDecoder(r.Body).Decode(&p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			address = p.Address
		} else {
			if err := r.ParseForm(); err != nil {
				_, err := fmt.Fprintf(w, "ParseForm() err: %v", err)
				if err != nil {
					log.Errorf("impossible to write response %v", err)
					return
				}
				http.Error(w, "Unable to parse request data", http.StatusBadRequest)
				return
			}
			address = r.FormValue("address")
		}

		if address == "" {
			http.Error(w, "Missing address in the request", http.StatusBadRequest)
			return
		}
		details, err := getShadowsocksProxyDetails(address)
		testsTotal.Inc()
		if err != nil {
			failuresTotal.Inc()
			http.Error(w, fmt.Sprintf("Unable to get information for address %s", address), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(details)
		if err != nil {
			log.Errorf("error occurred when sending the data back to the client %v", err)
		}
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	var staticFS = http.FS(indexFile)
	http.Handle("/", http.FileServer(staticFS))

	http.Handle("/metrics", promhttp.Handler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Infof("Starting server at port %s\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatal(err)
	}
}

func getShadowsocksProxyDetails(address string) (WTFIsMyIPData, error) {
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

	go ssproxy.ListenForOneConnection(proxyAddr, addr, ciph.StreamConn, ready, func(c net.Conn) (socks.Addr, error) { return socks.Handshake(c) })
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return WTFIsMyIPData{}, err
	}

	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport, Timeout: time.Second * 30}
	httpTransport.Dial = dialer.Dial
	<-ready
	wtfismyipURL := "https://wtfismyip.com/json"
	if getEnvBool("IPV4_ONLY", false) {
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

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	parseBool, err := strconv.ParseBool(value)
	if err != nil {
		log.Errorf("impossible to parse a bool from %s in your environment. Current value is: %s. Using default value (%t) instead.", key, value, fallback)
		return fallback
	}
	return parseBool
}

func extractCredentialsFromBase64(address string) (string, error) {
	key := address[5:strings.Index(address, "@")]

	creds, err := base64DecodeStripped(key)
	if err != nil {
		return "", err
	}

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
