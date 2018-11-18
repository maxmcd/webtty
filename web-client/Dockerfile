FROM maxmcd/webtty-node-go as builder

COPY . .
RUN go get -v . \
    && cd ./web-client \
    && npm install \
    && npm run build

FROM nginx:alpine

RUN echo "application/wasm    wasm" >> /etc/mime.types \
    && sed -i -e 's/wmlc;/wmlc;\n    application\/wasm    wasm;\n/g' /etc/nginx/mime.types

COPY --from=builder \
    /go/src/github.com/maxmcd/webtty/web-client/dist \
    /usr/share/nginx/html

