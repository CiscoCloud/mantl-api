FROM gliderlabs/alpine:3.2
MAINTAINER Ryan Eschinger <ryanesc@gmail.com>

RUN apk add --update ca-certificates bash
COPY launch.sh /launch.sh

COPY . /go/src/github.com/CiscoCloud/mantl-api

RUN apk add go git mercurial \
	&& cd /go/src/github.com/CiscoCloud/mantl-api \
	&& export GOPATH=/go \
	&& go get -t -u github.com/stretchr/testify \
	&& go get -t \
  && go test ./... \
	&& go build -o /bin/mantl-api \
	&& rm -rf /go \
	&& apk del --purge go mercurial

ENTRYPOINT ["/launch.sh"]
