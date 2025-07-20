FROM golang:1.24 AS go-lint-stage
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.2.2
WORKDIR /app
COPY deployments/golangci-lint.yml /app/.golangci.yml
COPY go.mod go.sum ./
RUN go mod download
# Copy the source code into the container
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN golangci-lint run --timeout 5m --config /app/.golangci.yml

FROM golang:1.24 AS go-build-stage
WORKDIR /app
COPY --from=go-lint-stage /app /app
RUN CGO_ENABLED=0 GOOS=linux go build -o /godfather-cmd ./cmd/godfather-cmd

# Run the tests in the container
FROM go-build-stage AS go-run-test-stage
RUN go test -v ./...

# Build Angular frontend
FROM node:24-alpine AS angular-build-stage
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm install
COPY web/ .
RUN npm run build -- --configuration=production

# Deploy the application into a separate lean image
FROM gcr.io/distroless/base-debian11 AS build-release-stage
WORKDIR /
COPY --from=go-build-stage /godfather-cmd /godfather-cmd
COPY --from=angular-build-stage /app/dist/ui/browser /browser
COPY configs/godfather.json /godfather.json
COPY deployments/godfather.key /godfather.key
COPY deployments/godfather.crt /godfather.crt
COPY db/migrations/ /migrations
EXPOSE 8443
USER nonroot:nonroot
ENTRYPOINT [ "/godfather-cmd", "-c", "godfather.json" ]