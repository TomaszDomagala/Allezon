# syntax = docker/dockerfile:1-experimental

FROM golang:1.19 as builder

ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0

WORKDIR /src

COPY go.* .
RUN go mod download

COPY . .

WORKDIR /src/services/api

RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /usr/bin/api



FROM gcr.io/distroless/static-debian11

COPY --from=builder /usr/bin/api /usr/bin/api

ENTRYPOINT ["/usr/bin/api"]