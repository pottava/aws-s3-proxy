.PHONY: all deps test build

all: build

deps:
	@docker run --rm -it -v "${GOPATH}"/src/github.com:/go/src/github.com \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			golang:1.13.4-alpine3.10 sh -c 'apk --no-cache add git && go mod vendor'

up:
	@docker-compose up -d

logs:
	@docker-compose logs -f

down:
	@docker-compose down -v

test:
	@docker run --rm -it -v "${GOPATH}"/src/github.com:/go/src/github.com \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			supinf/golangci-lint:1.12 \
			run --config .golangci.yml
	@docker run --rm -it -v "${GOPATH}"/src/github.com:/go/src/github.com \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			--entrypoint go supinf/go-gox:1.11 \
			test -vet off $(go list ./...)

build:
	@docker run --rm -it -v "${GOPATH}"/src/github.com:/go/src/github.com \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			supinf/go-gox:1.11 --osarch "linux/amd64 darwin/amd64 windows/amd64" \
			-ldflags "-s -w" -output "dist/{{.OS}}_{{.Arch}}"
