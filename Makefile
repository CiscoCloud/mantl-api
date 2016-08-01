NAME = $(shell awk -F\" '/^const Name/ { print $$2 }' main.go)
VERSION = $(shell awk -F\" '/^const Version/ { print $$2 }' main.go)
TESTDEPS = $(shell go list -f '{{range .TestImports}}{{.}} {{end}}' $(shell glide novendor))
DOCKERREPO=ciscocloud

all: deps build

deps:
	glide install --strip-vcs --strip-vendor --update-vendored
	find vendor -not -name '*.go' -not -name '*.s' -not -name '*.pl' -not -name '*.c' -not -name LICENSE -type f -delete
	echo $(TESTDEPS) | xargs -n1 go get

updatedeps:
	glide update --strip-vcs --strip-vendor --update-vendored
	find vendor -not -name '*.go' -not -name '*.s' -not -name '*.pl' -not -name '*.c' -not -name LICENSE -type f -delete

build:
	go build .

quicktest:
	go test $(shell glide novendor) -timeout=30s -parallel=4

test: quicktest
	go vet $(shell glide novendor)

docker:
	find . -name ".DS_Store" -depth -exec rm {} \;
	docker build -t $(DOCKERREPO)/$(NAME) .
	docker tag -f $(DOCKERREPO)/$(NAME) $(DOCKERREPO)/$(NAME):$(VERSION)

pushedge: docker
	docker tag -f $(DOCKERREPO)/$(NAME) $(DOCKERREPO)/$(NAME):edge
	docker push $(DOCKERREPO)/$(NAME):edge

push: docker
	docker push $(DOCKERREPO)/$(NAME):$(VERSION)
	docker push $(DOCKERREPO)/$(NAME):latest
