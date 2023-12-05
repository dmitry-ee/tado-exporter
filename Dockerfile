FROM --platform=${BUILDPLATFORM} tonistiigi/xx as xx
FROM --platform=${BUILDPLATFORM} golang:1.18.1-alpine3.15 as builder

COPY --from=xx / /
ARG TARGETPLATFORM

WORKDIR /build

ENV TINI_VERSION=v0.19.0

COPY go.mod .

RUN wget https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static-$(xx-info arch) && \
    go mod download && \
    chmod +x /build/tini-static-$(xx-info arch)

COPY . .

RUN CGO_ENABLED=0 xx-go build -ldflags="-w -s" -o /main *.go

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder main /bin/main
COPY --from=builder /build/tini-* /tini

ENTRYPOINT ["/tini", "--"]
CMD ["/bin/main"]