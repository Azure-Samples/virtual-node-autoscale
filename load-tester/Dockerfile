FROM quay.io/deis/lightweight-docker-go:v0.2.0
ENV CGO_ENABLED=0
RUN go get -u github.com/rakyll/hey 


FROM scratch
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /go/bin/hey /bin/hey
ENTRYPOINT ["/bin/hey"]
