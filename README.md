# ShadowTest

A service to test shadowsocks keys.

[![Go](https://github.com/jadolg/ShadowTest/actions/workflows/go.yml/badge.svg)](https://github.com/jadolg/ShadowTest/actions/workflows/go.yml)

## How to use

Using curl, call te test endpoint with a SIP002 compatible address:
`curl -i localhost:8080/v2/test -d "address=ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpiYWRwYXNzd29yZA@localhost:6276/?outline=1"`

#### Results

- 200: Everything went well and there's data for you in the https://wtfismyip.com/json format
- 4xx: You are either requesting the wrong URL or passing bad data to the server
- 502: There was an error getting data for this address which means either the address is invalid or the server is
  offline
- 504: There was a timeout getting data for this address

## Demo service

A demo service is deployed at https://shadowtest.akiel.dev/

## Deploy

### Run using snap

[![Get it from the Snap Store](https://snapcraft.io/static/images/badges/en/snap-store-black.svg)](https://snapcraft.io/shadowtest)

Run `snap install shadowtest` and the service will be exposed for you on port **51292**

### Run using docker

Run `docker run -p 8080:8080 -d guamulo/shadowtest` to get it running with docker on port **8080**
