CONFIG_PATH=${HOME}/.proglog/

.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

.PHONY: clean
clean:
	# Delete old certificates
	rm -f ${CONFIG_PATH}/*.csr ${CONFIG_PATH}/*.pem

.PHONY: gencert
gencert: init clean
	# Generate root CA
	cfssl gencert \
		-initca test/ca-csr.json | cfssljson -bare ca

	# Generate server certificate
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=server \
		test/server-csr.json | cfssljson -bare server

	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: install_dependencies
install_dependencies: get

.PHONY: get
get:
	go get -t -v ./...

.PHONY: compile
compile:
	protoc api/v1/*.proto \
		--gogo_out=Mgogoproto/gogo.proto=github.com/gogo/protobuf/proto,plugins=grpc:. \
		--proto_path=$$(go list -f '{{ .Dir }}' -m github.com/gogo/protobuf) \
		--proto_path=.

.PHONY: test
test: gencert
	echo "" > coverage.txt
	for d in `go list ./...`; do \
		go test -p 1 -v -timeout 240s -race -coverprofile=profile.out -covermode=atomic $$d || exit 1; \
		if [ -f profile.out ]; then \
			cat profile.out >> coverage.txt; \
			rm profile.out; \
		fi \
	done

.PHONY: test_no_cov
test_no_cov:
	go test -race ./...