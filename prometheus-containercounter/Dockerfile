FROM quay.io/deis/lightweight-docker-go:v0.2.0
ENV CGO_ENABLED=0
WORKDIR /go/src/github.com/jeremyrickard/prometheus-containercounter
COPY vendor/ vendor/
COPY cmd/counter cmd/counter
COPY pkg/ pkg/
RUN go build -o bin/counter ./cmd/counter

FROM scratch
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /go/src/github.com/jeremyrickard/prometheus-containercounter/bin/counter /app/counter
CMD ["/app/counter"]
