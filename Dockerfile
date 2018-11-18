FROM golang:1.11.2-stretch

RUN apt-get update && \
    curl -sL https://deb.nodesource.com/setup_11.x | bash -\
    && apt-get install -y --no-install-recommends \
    nodejs=11.2.0* \
    libssl-dev=1.1.0* \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/github.com/maxmcd/webtty


