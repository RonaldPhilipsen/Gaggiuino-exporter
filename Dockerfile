# Build the application from source
FROM golang:1.25 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG VERSION=v0.0.0 \
    GIT_SHA=unknown

ENV LDFLAGS="-s -w -extldflags '-static' \
            -X '$(PACKAGE)/build.BuildVersion=$(VERSION)' \
            -X '$(PACKAGE)/build.BuildCommitSha=$(GIT_SHA)' \
            -X '$(PACKAGE)/build.BuildDate=$(shell LC_ALL=en_US.UTF-8 date)'"

RUN go build -ldflags="$LDFLAGS" -o /dist/gaggiuino-exporter


# Run the tests in the container
FROM build-stage AS run-test-stage
RUN go test -v ./...

# Deploy the application binary into a lean image
FROM gcr.io/distroless/base-debian11 AS build-release-stage
LABEL org.opencontainers.image.source=https://github.com/RonaldPhilipsen/gaggiuino-exporter \
      org.opencontainers.image.description="Gaggiuino Prometheus Exporter" \
      org.opencontainers.image.licenses=MIT

WORKDIR /

COPY --from=build-stage /dist/gaggiuino-exporter /gaggiuino-exporter

EXPOSE 9995

USER nonroot:nonroot

ENTRYPOINT ["/gaggiuino-exporter"]