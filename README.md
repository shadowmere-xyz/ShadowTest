# ShadowTest

A service to test shadowsocks keys.

## How to use

Using curl, call te test endpoint with a SIP002 compatible address: 
`curl -i localhost:8080/v1/test -d address=ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpiYWRwYXNzd29yZA@localhost:6276/?outline=1`
