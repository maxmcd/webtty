FROM golang:1.11.2-stretch

RUN apt-get update \
	&& apt-get install -y libssl-dev \
	&& rm -rf /var/lib/apt/lists/*

RUN go get -u "github.com/maxmcd/webtty"

CMD ["webtty"]
