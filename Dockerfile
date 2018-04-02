FROM golang:1.10-alpine
WORKDIR /go/src/geoserver
RUN apk add --no-cache git mercurial curl bash
COPY . .
RUN go get -d -v ./...
RUN go get github.com/stretchr/testify
RUN go get golang.org/x/tools/cmd/cover
RUN go get github.com/mattn/goveralls
RUN go install -v ./...
RUN chmod +x scripts/entry.sh
CMD "/bin/sh"