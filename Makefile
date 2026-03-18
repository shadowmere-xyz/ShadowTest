TAG?=$(shell git describe --tags --always --dirty)
IMAGE_NAME?=guamulo/shadowtest:$(TAG)
SSR_DIR?=/tmp/shadowsocksr

.PHONY:
start_ss_test_server:
	- pkill ss-server
	ss-server -v -p 6276 -k password &

.PHONY:
start_ssr_test_server: $(SSR_DIR)
	- pkill -f "shadowsocks/server.py" 2>/dev/null; true
	cd $(SSR_DIR) && python3 shadowsocks/server.py -p 16276 -k testpassword -m aes-256-cfb -O origin -o plain &

$(SSR_DIR):
	git clone --depth 1 https://github.com/shadowsocksrr/shadowsocksr.git $(SSR_DIR)
	sed -i 's/collections\.MutableMapping/collections.abc.MutableMapping/g' $(SSR_DIR)/shadowsocks/lru_cache.py

.PHONY:
test: start_ss_test_server start_ssr_test_server
	go test ./... -count=1

.PHONY:
build_image:
	docker build -t $(IMAGE_NAME) .

.PHONY:
start_server_rust:
	shadowsocks-rust.ssserver -s 127.0.0.1:6276 -k password -m chacha20-ietf-poly1305
