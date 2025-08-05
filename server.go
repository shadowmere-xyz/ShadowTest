package main

import (
	"ShadowTest/offline_cache"
	"ShadowTest/ssproxy"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
)

const ContentType = "Content-Type"
const ContentTypeJson = "application/json"
const WTFIsMyIPTestURL = "https://wtfismyip.com/test"

var offlineCache offline_cache.SafeIsOfflineCache

type proxyJson struct {
	Address string `json:"address"`
	Timeout int    `json:"timeout,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type version struct {
	GitCommit string `json:"git_commit"`
	Version   string `json:"version"`
}

//go:embed index.html
var indexFile embed.FS

//go:embed favicon.ico
var faviconFile embed.FS

func getRouter(ipv4Only bool) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/test", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Deprecated endpoint. Use v2 instead.", http.StatusNotFound)
	})

	mux.HandleFunc("/v2/test", func(w http.ResponseWriter, r *http.Request) {
		defer closeBody(r)
		if r.Method != "POST" {
			http.Error(w, "Method is not supported.", http.StatusMethodNotAllowed)
			return
		}

		if ssproxy.IsWTFIsMyIpOffline(&offlineCache, WTFIsMyIPTestURL) {
			log.Errorf("https://wtfismyip.com is having problems")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		address, timeout, err := getAddressAndTimeout(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		details, err := ssproxy.GetShadowsocksProxyDetails(address, ipv4Only, timeout)
		testsTotal.Inc()
		if err != nil {
			failuresTotal.Inc()
			fillCheckError(w, err, address)
			return
		}

		w.Header().Set(ContentType, ContentTypeJson)
		err = json.NewEncoder(w).Encode(details)

		if err != nil {
			log.Errorf("error occurred when sending the data back to the client %v", err)
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(ContentType, ContentTypeJson)
		_ = json.NewEncoder(w).Encode(version{
			GitCommit: GitCommit,
			Version:   Version,
		})
	})

	var staticFS = http.FS(indexFile)
	mux.Handle("/", http.FileServer(staticFS))

	mux.Handle("/metrics", promhttp.Handler())

	var faviconFS = http.FS(faviconFile)
	mux.Handle("/favicon.ico", http.FileServer(faviconFS))

	return mux, nil
}

func fillCheckError(w http.ResponseWriter, err error, address string) {
	message := fmt.Sprintf("Unable to get information for address %s", address)
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		message = fmt.Sprintf("Timeout getting information for address %s", address)
	}

	response := errorResponse{Error: message}
	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		log.Errorf("error occurred when sending the data back to the client %v", err)
	}
}

func closeBody(r *http.Request) {
	func(Body io.ReadCloser) {
		if Body != nil {
			err := Body.Close()
			if err != nil {
				log.Errorf("impossible to close request body %v", err)
			}
		}
	}(r.Body)
}

func getAddressAndTimeout(r *http.Request) (string, int, error) {
	address := ""
	timeout := 0
	var err error

	if r.Header.Get(ContentType) == ContentTypeJson {
		address, timeout, err = getAddressAndTimeoutFromJSON(r)
	} else {
		address, timeout, err = getAddressAndTimeoutFromForm(r)
	}

	if err != nil {
		return "", 0, err
	}
	if address == "" {
		return "", 0, fmt.Errorf("missing address in the request")
	}

	if timeout <= 0 {
		timeout, err = getDefaultTimeout()
		if err != nil {
			log.Errorf("unable to get default timeout: %v", err)
			return "", 0, fmt.Errorf("unable to get default timeout: %v", err)
		}
	}

	return address, timeout, nil
}

func getAddressAndTimeoutFromForm(r *http.Request) (string, int, error) {
	if err := r.ParseForm(); err != nil {
		return "", 0, fmt.Errorf("unable to parse request data")
	}
	address := html.EscapeString(r.FormValue("address"))
	if r.FormValue("timeout") != "" {
		timeout, err := strconv.Atoi(r.FormValue("timeout"))
		if err != nil {
			return "", 0, fmt.Errorf("unable to parse timeout")
		}
		return address, timeout, nil
	}
	return address, 0, nil
}

func getAddressAndTimeoutFromJSON(r *http.Request) (string, int, error) {
	p := proxyJson{}
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		return "", 0, err
	}
	address := html.EscapeString(p.Address)
	return address, p.Timeout, nil
}

func getDefaultTimeout() (int, error) {
	timeout := 30
	timeoutFromEnv := os.Getenv("TIMEOUT")
	if timeoutFromEnv != "" {
		timeoutInt, err := strconv.Atoi(timeoutFromEnv)
		if err != nil {
			return 0, err
		}
		timeout = timeoutInt
	}
	return timeout, nil
}
