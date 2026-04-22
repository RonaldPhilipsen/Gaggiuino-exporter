# Build the application from source
FROM golang:1.26@sha256:f7159064a17ccc65d0e10e342ae8783026182704bf4af8f6df8d5ba9af2be2fd AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG VERSION=v0.0.0 \
    GIT_SHA=unknown

ENV CGO_ENABLED=0 \
    GOOS=linux \
    PACKAGE=github.com/RonaldPhilipsen/gaggiuino-exporter 

ENV LDFLAGS="-s -w -extldflags '-static' \
    -X $PACKAGE/build.BuildVersion=$VERSION \
    -X $PACKAGE/build.BuildCommitSha=$GIT_SHA"

RUN go build -ldflags="$LDFLAGS" -o /dist/gaggiuino-exporter


# Run the tests in the container
FROM build-stage AS run-test-stage
RUN go test -v ./...

# Deploy the application binary into a lean image
FROM gcr.io/distroless/static-debian11@sha256:1dbe426d60caed5d19597532a2d74c8056cd7b1674042b88f7328690b5ead8ed AS build-release-stage
LABEL org.opencontainers.image.source=https://github.com/RonaldPhilipsen/gaggiuino-exporter 
LABEL org.opencontainers.image.description="Gaggiuino Prometheus Exporter" 
LABEL org.opencontainers.image.licenses=MIT

WORKDIR /

COPY --from=build-stage /dist/gaggiuino-exporter /gaggiuino-exporter

EXPOSE 9995

USER nonroot:nonroot

ENTRYPOINT ["/gaggiuino-exporter"]