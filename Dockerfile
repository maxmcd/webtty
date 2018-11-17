FROM golang:1.11.2-stretch

RUN apt-get update && \
    curl -sL https://deb.nodesource.com/setup_11.x | bash -\
    && apt-get install -y nodejs libssl-dev

WORKDIR /go/src/github.com/maxmcd/webtty


