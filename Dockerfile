# Build the application from source
FROM golang:1.27@sha256:e519d3b0a5b285e5aa53ab9eef02bc6bbb9cb04608389b743a136a9ce75f0d1e AS build-stage

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
WORKDIR /

COPY --from=build-stage /dist/gaggiuino-exporter /gaggiuino-exporter

EXPOSE 9995

USER nonroot:nonroot

ENTRYPOINT ["/gaggiuino-exporter"]

# OCI image labels. Keep this block last so that changed values only rebuild this step.
ARG VERSION=v0.0.0
ARG GIT_SHA=unknown
LABEL \
    org.opencontainers.image.title="gaggiuino-exporter" \
    org.opencontainers.image.description="Gaggiuino Prometheus Exporter" \
    org.opencontainers.image.url="https://github.com/RonaldPhilipsen/gaggiuino-exporter" \
    org.opencontainers.image.source="https://github.com/RonaldPhilipsen/gaggiuino-exporter" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.version="${VERSION}" \
    org.opencontainers.image.revision="${GIT_SHA}"
