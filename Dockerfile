# Lint stage
FROM golang:1.24 AS go-lint-stage
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.2.2
WORKDIR /app
# Install Delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest
# Copy the GolangCI configuration file
COPY deployments/golangci-lint.yml /app/.golangci.yml
# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
# Copy the source code into the container
COPY cmd/ ./cmd/
COPY internal/ ./internal/
# Run linters
RUN golangci-lint run --timeout 5m --config /app/.golangci.yml

# MOEXMON build stage
FROM golang:1.24 AS moexmon-build-stage
WORKDIR /app
COPY --from=go-lint-stage /app /app
COPY --from=go-lint-stage /go/pkg/mod /go/pkg/mod
RUN CGO_ENABLED=0 GOOS=linux go build -o /moexmon-cmd ./cmd/moexmon-cmd

# SQUEALER build stage
FROM golang:1.24 AS squealer-build-stage
WORKDIR /app
COPY --from=go-lint-stage /app /app
COPY --from=go-lint-stage /go/pkg/mod /go/pkg/mod
RUN CGO_ENABLED=0 GOOS=linux go build -o /squealer-cmd ./cmd/squealer-cmd

# GODFATHER build stage
FROM golang:1.24 AS godfather-build-stage
WORKDIR /app
COPY --from=go-lint-stage /app /app
COPY --from=go-lint-stage /go/pkg/mod /go/pkg/mod
RUN CGO_ENABLED=0 GOOS=linux go build -o /godfather-cmd ./cmd/godfather-cmd

# Run the tests in the container
FROM go-lint-stage AS go-run-test-stage
WORKDIR /app
RUN go test -coverprofile=coverage.out -v ./...

# Build Angular frontend
FROM node:24-alpine AS angular-build-stage
WORKDIR /app
COPY web/godfather/package.json web/godfather/package-lock.json ./
RUN npm install
COPY web/godfather/ .
RUN npm run build -- --configuration=production

# Deploy the Godfather application into a separate lean image
FROM gcr.io/distroless/base-debian12 AS godfather-cmd
WORKDIR /
COPY --from=godfather-build-stage /godfather-cmd /godfather-cmd
COPY --from=angular-build-stage /app/dist/godfather/browser /browser
COPY --from=go-lint-stage /go/bin/dlv /usr/local/bin/dlv
COPY configs/godfather.json /godfather.json
COPY deployments/godfather.key /godfather.key
COPY deployments/godfather.crt /godfather.crt
COPY db/migrations/ /migrations
EXPOSE 9443
USER nonroot:nonroot
ENTRYPOINT [ "/godfather-cmd", "-v", "-c", "godfather.json" ]

# Deploy the MOEX Monitor application into a separate lean image
FROM gcr.io/distroless/base-debian12 AS moexmon-cmd
WORKDIR /
COPY --from=moexmon-build-stage /moexmon-cmd /moexmon-cmd
COPY configs/moexmon.json /moexmon.json
USER nonroot:nonroot
ENTRYPOINT [ "/moexmon-cmd", "-v", "-c", "moexmon.json" ]

# Deploy the Squealer application into a separate lean image
FROM gcr.io/distroless/base-debian12 AS squealer-cmd
WORKDIR /
COPY --from=squealer-build-stage /squealer-cmd /squealer-cmd
COPY configs/squealer.json /squealer.json
USER nonroot:nonroot
ENTRYPOINT [ "/squealer-cmd", "-v", "-c", "squealer.json" ]