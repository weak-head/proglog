IMAGE_ID ?= github.com/weak-head/proglog
IMAGE_TAG ?= 0.0.1

CONFIG_PATH=${HOME}/.proglog/

ACL_MODEL=acl-model.conf
ACL_POLICY=acl-policy.csv

TEST_COVERAGE_TEMP=profile.out
TEST_COVERAGE=coverage.txt

.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

.PHONY: clean
clean:
	# Delete old certificates
	rm -f ${CONFIG_PATH}/*.csr ${CONFIG_PATH}/*.pem

	# Delete old ACL model and policy
	rm -f ${CONFIG_PATH}/${ACL_MODEL}
	rm -f ${CONFIG_PATH}/${ACL_POLICY}

.PHONY: get
get:
	go get -t -v ./...

.PHONY: install_dependencies
install_dependencies: get
	go get github.com/gogo/protobuf
	go get github.com/gogo/protobuf/protoc-gen-gogo
	go get github.com/cloudflare/cfssl/cmd/cfssl
	go get github.com/cloudflare/cfssl/cmd/cfssljson
	go get google.golang.org/grpc/internal/channelz
	# go get github.com/gogo/protobuf/proto
	# go get github.com/gogo/protobuf/jsonpb
	# go get github.com/gogo/protobuf/gogoproto

.PHONY: copy-acl
copy-acl: init clean
	# Copy ACL model and policy
	cp test/${ACL_MODEL} ${CONFIG_PATH}/${ACL_MODEL}
	cp test/${ACL_POLICY} ${CONFIG_PATH}/${ACL_POLICY}

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

	# Generate root client certificate
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="root" \
		test/client-csr.json | cfssljson -bare root-client

	# Generate nobody client certificate
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="nobody" \
		test/client-csr.json | cfssljson -bare nobody-client

	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: compile
compile:
	protoc api/v1/*.proto \
		--gogo_out=Mgogoproto/gogo.proto=github.com/gogo/protobuf/proto,plugins=grpc:. \
		--proto_path=$$(go list -f '{{ .Dir }}' -m github.com/gogo/protobuf) \
		--proto_path=.

.PHONY: coverage
coverage: gencert copy-acl
	# Run tests generating coverage report
	echo "" > ${TEST_COVERAGE}
	for d in `go list ./...`; do \
		go test \
			-p 1 \
			-v \
			-timeout 240s \
			-coverprofile=${TEST_COVERAGE_TEMP} \
			-covermode=atomic \
			$$d || exit 1; \
		if [ -f ${TEST_COVERAGE_TEMP} ]; then \
			cat ${TEST_COVERAGE_TEMP} >> ${TEST_COVERAGE}; \
			rm ${TEST_COVERAGE_TEMP}; \
		fi \
	done

.PHONY: test
test: gencert copy-acl
	go test ./...

.PHONY: build-docker
build-docker:
	docker build -t $(IMAGE_ID):$(IMAGE_TAG) .

.PHONY: clean-deploy
clean-deploy:
	helm uninstall proglog
	kind delete cluster

.PHONY: deploy
deploy: build-docker
	kind create cluster
	kind load docker-image $(IMAGE_ID):$(IMAGE_TAG)
	helm install proglog deploy/proglog
	# kubectl port-forward pod/proglog-0 8400 8400
	