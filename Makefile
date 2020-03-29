EASYRSA_VERSION = v3.0.6

certs:
	wget https://github.com/OpenVPN/easy-rsa/releases/download/$(EASYRSA_VERSION)/EasyRSA-unix-$(EASYRSA_VERSION).tgz -O - | tar -xz && \
		mv EasyRSA-$(EASYRSA_VERSION) certs && \
		cd certs/ && \
		./easyrsa init-pki && \
		EASYRSA_BATCH=1 ./easyrsa build-ca nopass && \
		./easyrsa build-server-full server nopass && \
		./easyrsa build-client-full client1.domain.tld nopass

clean:
	rm -rf ./certs

start:
	KUBECONFIG=$(HOME)/.kube/config go run main.go