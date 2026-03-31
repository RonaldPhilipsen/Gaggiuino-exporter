# Build the application from source
FROM golang:1.26@sha256:595c7847cff97c9a9e76f015083c481d26078f961c9c8dca3923132f51fe12f1 AS build-stage

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
FROM gcr.io/distroless/base-debian11@sha256:ac69aa622ea5dcbca0803ca877d47d069f51bd4282d5c96977e0390d7d256455 AS build-release-stage
LABEL org.opencontainers.image.source=https://github.com/RonaldPhilipsen/gaggiuino-exporter \
      org.opencontainers.image.description="Gaggiuino Prometheus Exporter" \
      org.opencontainers.image.licenses=MIT

WORKDIR /

COPY --from=build-stage /dist/gaggiuino-exporter /gaggiuino-exporter

EXPOSE 9995

USER nonroot:nonroot

ENTRYPOINT ["/gaggiuino-exporter"]