# ShadowTest

A service to test shadowsocks keys.

## How to use

### Start your server
`docker run --rm -p 8080:8080 -d guamulo/shadowtest`

### Query the server
Using curl, call te test endpoint with a SIP002 compatible address: 
`curl -i localhost:8080/v1/test -d "address=ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpiYWRwYXNzd29yZA@localhost:6276/?outline=1"`

#### Results
- 200: Everything went well and there's data for you in the https://wtfismyip.com/json format
- 4xx: You are either requesting the wrong URL or passing bad data to the server
- 502: There was an error getting data for this address which means either the address is invalid or the server is offline

## Demo service

A demo service is deployed at https://sshadowtest.herokuapp.com/
