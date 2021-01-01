FROM golang:1.15

RUN apt-get update && \
    curl -sL https://deb.nodesource.com/setup_12.x | bash -\
    && apt-get install -y --no-install-recommends \
    nodejs=12.* \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/github.com/maxmcd/webtty
