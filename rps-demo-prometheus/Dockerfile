FROM quay.io/deis/lightweight-docker-go:v0.2.0
ENV CGO_ENABLED=0
WORKDIR /go/src/rpsdemo
COPY vendor/ vendor/
COPY cmd/app cmd/app
COPY public/ public
RUN go build -o bin/app ./cmd/app

FROM scratch
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /go/src/rpsdemo/bin/app /app/server
COPY --from=0 /go/src/rpsdemo/public /app/content
CMD ["/app/server"]
