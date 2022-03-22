#build stage
FROM golang:alpine AS builder
RUN apk add --no-cache git
WORKDIR /go/src/auto-release
COPY . .
RUN go get -d -v ./...
RUN go build -o /go/bin/auto-release -v ./...

#final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN addgroup -S ghgroup && adduser -S ghuser -G ghgroup -u 888
RUN mkdir /usr/bin/gh && \
    wget https://github.com/cli/cli/releases/download/v2.6.0/gh_2.6.0_linux_amd64.tar.gz -O ghcli.tar.gz && \
    tar --strip-components=1 -xf ghcli.tar.gz -C /usr/bin/gh
COPY --from=builder /go/bin/auto-release /auto-release
RUN chown ghuser /auto-release && chown -R ghuser /usr/bin/gh
USER ghuser
ENV PATH="${PATH}:/usr/bin/gh/bin"
ENTRYPOINT /auto-release
