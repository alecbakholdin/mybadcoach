# A workspace image that can run every command in docs/testing.md unchanged.
#
# Go is copied from the pinned official Go image into the pinned Node image;
# both are Alpine-based, so the musl-linked toolchain works as-is. This keeps
# the devcontainer to the same two base images the CI/compose stack already
# uses instead of introducing a third vendor image.
FROM golang:1.27.3-alpine AS golang

FROM node:22.22-alpine

COPY --from=golang /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

# git: source control; docker CLI + compose plugin: run the compose-defined
# e2e seam from inside the container via the mounted host socket.
RUN apk add --no-cache bash git docker-cli docker-cli-compose

WORKDIR /workspaces/mybadcoach
