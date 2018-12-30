.PHONY: all deps test build

all: build

deps:
	@docker run --rm -it -v "${GOPATH}"/src/github.com/pottava:/go/src/github.com/pottava \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			supinf/go-dep:0.5 init

test:
	@docker run --rm -it -v "${GOPATH}"/src/github.com/pottava:/go/src/github.com/pottava \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			supinf/golangci-lint:1.12 \
			run --config .golangci.yml
	@docker run --rm -it -v "${GOPATH}"/src/github.com/pottava:/go/src/github.com/pottava \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			--entrypoint go supinf/go-gox:1.11 \
			test -vet off $(go list ./...)

build:
	@docker run --rm -it -v "${GOPATH}"/src/github.com/pottava:/go/src/github.com/pottava \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			supinf/go-gox:1.11 --osarch "linux/amd64 darwin/amd64 windows/amd64" \
			-ldflags "-s -w" -output "dist/{{.OS}}_{{.Arch}}"
