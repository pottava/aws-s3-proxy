.PHONY: all deps test build

all: build

deps:
	@docker run --rm -it -v "${GOPATH}/src/github.com:/go/src/github.com" \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			golang:1.13.7-alpine3.11 sh -c 'apk --no-cache add git && go mod vendor'

up:
	@docker-compose up -d

logs:
	@docker-compose logs -f

down:
	@docker-compose down -v

test:
	@docker run --rm -it -v "${GOPATH}/src/github.com:/go/src/github.com" \
			-w /go/src/github.com/pottava/aws-s3-proxy \
            golangci/golangci-lint:v1.23.1-alpine \
			golangci-lint run --config .golangci.yml
	@docker run --rm -it -v "${GOPATH}/src/github.com:/go/src/github.com" \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			golangci/golangci-lint:v1.23.1-alpine \
			sh -c "go list ./... | grep -v /vendor/ | xargs go test -p 1 -count=1"

build:
	@docker run --rm -it -v "${GOPATH}/src/github.com:/go/src/github.com" \
			-w /go/src/github.com/pottava/aws-s3-proxy \
			supinf/go-gox:1.11 --osarch "linux/amd64 darwin/amd64 windows/amd64" \
			-ldflags "-s -w" -output "dist/{{.OS}}_{{.Arch}}"
