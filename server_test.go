package main

import (
	"ShadowTest/ssproxy"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthcheck(t *testing.T) {
	router, err := getRouter(true)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGetProxyDetailsFromServerJSON(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@localhost:6276/?outline=1"

	router, err := getRouter(true)
	assert.NoError(t, err)

	body := bytes.NewBuffer([]byte(fmt.Sprintf("{ \"address\":\"%s\" }", address)))
	req, _ := http.NewRequest("POST", "/v2/test", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	details := ssproxy.WTFIsMyIPData{}
	err = json.NewDecoder(rr.Body).Decode(&details)
	assert.NoError(t, err)

	assert.NotEmpty(t, details.YourFuckingIPAddress)
	assert.NotEmpty(t, details.YourFuckingLocation)
}

func TestGetProxyDetailsFromServerForm(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@localhost:6276/?outline=1"

	router, err := getRouter(true)
	assert.NoError(t, err)

	body := bytes.NewBuffer([]byte(fmt.Sprintf("address=%s", address)))
	req, _ := http.NewRequest("POST", "/v2/test", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	details := ssproxy.WTFIsMyIPData{}
	err = json.NewDecoder(rr.Body).Decode(&details)
	assert.NoError(t, err)

	assert.NotEmpty(t, details.YourFuckingIPAddress)
	assert.NotEmpty(t, details.YourFuckingLocation)
}

func TestGetProxyDetailsFromServerJSONTimeout(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@shadowtest.akiel.dev:6276"

	router, err := getRouter(true)
	assert.NoError(t, err)

	body := bytes.NewBuffer([]byte(fmt.Sprintf("{ \"address\":\"%s\", \"timeout\": 1 }", address)))
	req, _ := http.NewRequest("POST", "/v2/test", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGetProxyDetailsFromServerJSONWithoutTimeout(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@localhost:6276/?outline=1"

	router, err := getRouter(true)
	assert.NoError(t, err)

	body := bytes.NewBuffer([]byte(fmt.Sprintf("{ \"address\":\"%s\" }", address)))
	req, _ := http.NewRequest("POST", "/v2/test", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDeprecatedV1(t *testing.T) {
	router, err := getRouter(true)
	assert.NoError(t, err)

	body := bytes.NewBuffer([]byte(fmt.Sprintf("{ \"address\":\"%s\" }", "")))
	req, _ := http.NewRequest("POST", "/v1/test", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	content, err := io.ReadAll(rr.Body)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Deprecated endpoint. Use v2 instead.\n"), content)
}

func TestTestMethodNotAllowed(t *testing.T) {
	router, err := getRouter(true)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/v2/test", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	assert.Equal(t, "Method is not supported.\n", rr.Body.String())
}

func TestTestDefaultTimeoutError(t *testing.T) {
	err := os.Setenv("TIMEOUT", "invalid")
	assert.NoError(t, err)
	defer func() {
		err := os.Unsetenv("TIMEOUT")
		if err != nil {
			fmt.Printf("failed to unset TIMEOUT env var: %v", err)
		}
	}()

	router, err := getRouter(true)
	assert.NoError(t, err)

	body := bytes.NewBuffer([]byte(`{ "address": "test" }`))
	req, _ := http.NewRequest("POST", "/v2/test", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "unable to get default timeout: strconv.Atoi: parsing \"invalid\": invalid syntax\n", rr.Body.String())
}

func TestTestFormData(t *testing.T) {
	router, err := getRouter(true)
	assert.NoError(t, err)

	form := "address=test_address&timeout=15"
	req, _ := http.NewRequest("POST", "/v2/test", bytes.NewBufferString(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestTestMissingAddress(t *testing.T) {
	router, err := getRouter(true)
	assert.NoError(t, err)

	body := bytes.NewBuffer([]byte(`{}`))
	req, _ := http.NewRequest("POST", "/v2/test", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "missing address in the request\n", rr.Body.String())
}

func TestTestInvalidTimeoutForm(t *testing.T) {
	router, err := getRouter(true)
	assert.NoError(t, err)

	form := "address=test_address&timeout=notanumber"
	req, _ := http.NewRequest("POST", "/v2/test", bytes.NewBufferString(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "unable to parse timeout\n", rr.Body.String())
}
