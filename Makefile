NAME=marin3r
EASYRSA_VERSION = v3.0.6
ENVOY_VERSION = v1.13.1
RELEASE=0.0.0

build: build/$(NAME)_amd64_$(RELEASE)

build/$(NAME)_amd64_$(RELEASE):
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o build/$(NAME)_amd64_$(RELEASE) cmd/main.go

docker-build: build/$(NAME)_amd64_$(RELEASE)
	docker build . -t quay.io/roivaz/$(NAME):v$(RELEASE) --build-arg release=$(RELEASE)

certs:
	example/gen-certs.sh $(EASYRSA_VERSION)

compose:
	docker-compose -f example/compose.yaml up

clean:
	rm -rf certs
	rm -rf build

envoy:
	docker run -ti --rm \
		--network=host \
		--add-host marin3r:127.0.0.1 \
		-v $$(pwd)/certs:/etc/envoy/tls \
		-v $$(pwd)/example:/config \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config/envoy-config.yaml

start:
	KUBECONFIG=$(HOME)/.kube/config go run cmd/main.go