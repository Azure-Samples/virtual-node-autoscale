FROM quay.io/deis/lightweight-docker-go:v0.2.0
ENV CGO_ENABLED=0
WORKDIR /go/src/online-store
COPY vendor/ vendor/
COPY cmd/app cmd/app
COPY public/ public
RUN go build -o bin/app ./cmd/app

FROM scratch
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /go/src/online-store/bin/app /app/server
COPY --from=0 /go/src/online-store/public /app/content
CMD ["/app/server"]
