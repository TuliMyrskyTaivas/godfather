FROM golang:1.24 AS lint-stage
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.2.2
WORKDIR /app
COPY deployments/golangci-lint.yml /app/.golangci.yml
COPY go.mod go.sum ./
RUN go mod download
# Copy the source code into the container
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN golangci-lint run --timeout 5m --config /app/.golangci.yml

FROM golang:1.24 AS build-stage
WORKDIR /app
#COPY go.mod go.sum ./
#RUN go mod download
COPY --from=lint-stage /app /app
RUN CGO_ENABLED=0 GOOS=linux go build -o /godfather-cmd ./cmd/godfather-cmd

# Run the tests in the container
FROM build-stage AS run-test-stage
RUN go test -v ./...

# Deploy the application into a separate lean image
FROM gcr.io/distroless/base-debian11 AS build-release-stage
WORKDIR /
COPY --from=build-stage /godfather-cmd /godfather-cmd
COPY configs/godfather.json /godfather.json
COPY deployments/godfather.key /godfather.key
COPY deployments/godfather.crt /godfather.crt
COPY db/migrations/ /migrations
EXPOSE 8443
USER nonroot:nonroot
ENTRYPOINT [ "/godfather-cmd", "-c", "godfather.json" ]