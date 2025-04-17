# Dockerfile has specific requirement to put this ARG at the beginning:
# https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
ARG BUILDER_IMAGE=golang:1.24
ARG BASE_IMAGE=gcr.io/distroless/static:nonroot

## Multistage build
FROM ${BUILDER_IMAGE} AS builder
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

# Install build tools
RUN apt-get update && apt-get install -y gcc libc6-dev && rm -rf /var/lib/apt/lists/*

## NeuralMagic internal repos pull config
ARG NM_TOKEN
ARG GIT_NM_USER
### use git token
RUN echo -e "machine github.com\n\tlogin ${GIT_NM_USER}\n\tpassword ${NM_TOKEN}" >> ~/.netrc
ENV GOPRIVATE=github.com/neuralmagic
ENV GIT_TERMINAL_PROMPT=1

# Dependencies
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download -x
RUN rm -rf ~/.netrc
# Sources
COPY cmd ./cmd
COPY pkg ./pkg
COPY internal ./internal
COPY api ./api
WORKDIR /src/cmd/epp

COPY lib ./lib
RUN ranlib lib/*.a

RUN go build -v -o /epp -ldflags="-extldflags '-L$(pwd)/lib'"

## Multistage deploy
FROM ${BUILDER_IMAGE}

WORKDIR /
COPY --from=builder /epp /epp

ENTRYPOINT ["/epp"]
