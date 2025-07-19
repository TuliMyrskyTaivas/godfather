FROM golang:1.24 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY internal/ ./internal/
COPY cmd/ ./cmd/
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