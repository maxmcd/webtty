FROM golang:1.11.2-stretch as builder

RUN curl -sL https://deb.nodesource.com/setup_11.x | bash -\
	&& apt-get install -y --no-install-recommends nodejs libssl-dev

RUN go get -u "github.com/maxmcd/webtty"

COPY . /tmp
WORKDIR /tmp
RUN npm install && npm run build

FROM nginx:alpine
COPY --from=builder /tmp/dist /usr/share/nginx/html
