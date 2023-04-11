package main

import (
	"ShadowTest/ssproxy"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type proxyJson struct {
	Address string `json:"address"`
}

//go:embed index.html
var indexFile embed.FS

func getRouter(ipv4_only bool) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/test", func(w http.ResponseWriter, r *http.Request) {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Errorf("impossible to close request body %v", err)
			}
		}(r.Body)
		if r.Method != "POST" {
			http.Error(w, "Method is not supported.", http.StatusNotFound)
			return
		}
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
		details, err := ssproxy.GetShadowsocksProxyDetails(address, ipv4_only)
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

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	var staticFS = http.FS(indexFile)
	mux.Handle("/", http.FileServer(staticFS))

	mux.Handle("/metrics", promhttp.Handler())

	return mux, nil
}
