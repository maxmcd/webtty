FROM golang:1.11.2-stretch

RUN apt-get update && \
 	curl -sL https://deb.nodesource.com/setup_11.x | bash -\
	&& apt-get install -y nodejs libssl-dev \
	&& rm -rf /var/lib/apt/lists/*

RUN go get -u "github.com/maxmcd/webtty"

add . /tmp
RUN cd /tmp && npm install && npm run build

FROM nginx:alpine
COPY --from=0 /tmp/dist /usr/share/nginx/html
