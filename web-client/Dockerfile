FROM maxmcd/webtty-node-go as builder

COPY . .
RUN go get -v . \
    && cd ./web-client \
    && npm install \
    && npm run build

FROM nginx:alpine
COPY --from=builder \
    /go/src/github.com/maxmcd/webtty/web-client/dist \
    /usr/share/nginx/html
