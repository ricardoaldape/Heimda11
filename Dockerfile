FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/heimda11 ./cmd/heimda11

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && addgroup -S heimda11 && adduser -S -G heimda11 heimda11 && mkdir -p /data && chown heimda11:heimda11 /data
COPY --from=build /out/heimda11 /usr/local/bin/heimda11
USER heimda11
VOLUME ["/data"]
EXPOSE 8080
ENV HEIMDA11_ADDR=:8080 HEIMDA11_DATA_PATH=/data/heimda11.json HEIMDA11_EDITION=community
ENTRYPOINT ["heimda11"]
