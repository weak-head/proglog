
.PHONY : install_dependencies
install_dependencies: get

.PHONY: get
get:
	go get -t -v ./...

.PHONY: compile
compile:
	protoc api/v1/*.proto \
		--gogo_out=Mgogoproto/gogo.proto=github.com/gogo/protobuf/proto:. \
		--proto_path=$$(go list -f '{{ .Dir }}' -m github.com/gogo/protobuf) \
		--proto_path=.

.PHONY: test
test:
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
	go test ./...