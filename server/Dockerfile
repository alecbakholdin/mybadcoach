# Build the Go server, then run it from a minimal Alpine image.
#
# modernc.org/sqlite is pure Go, so CGO is disabled and the binary is
# effectively static: the runtime image needs no compiler or libc toolchain.
FROM golang:1.27.3-alpine AS build

WORKDIR /src

# Cache module downloads independently of the source tree.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server

FROM alpine:3.21

# Run as an unprivileged user. The named volume in compose is initialized with
# this ownership on first use, so the server can create its SQLite file.
RUN adduser -D -H -u 10001 app \
	&& mkdir -p /data \
	&& chown app:app /data

COPY --from=build /out/server /usr/local/bin/server

USER app
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/server"]
