FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY main.go ./
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/unishare .

FROM alpine:3.22

LABEL org.opencontainers.image.title="Unishare"
LABEL org.opencontainers.image.description="Self-hosted personal dropbox for links, text, and files"
LABEL org.opencontainers.image.source="https://github.com/curserio/unishare"
LABEL org.opencontainers.image.licenses="MIT"

RUN adduser -D -H -u 10001 unishare
WORKDIR /app
COPY --from=build /out/unishare /app/unishare
COPY static /app/static
RUN mkdir -p /data && chown -R unishare:unishare /data

USER unishare
EXPOSE 8080
ENV UNISHARE_ADDR=:8080
ENV UNISHARE_DATA_DIR=/data
ENV UNISHARE_STATIC_DIR=/app/static

CMD ["/app/unishare"]
