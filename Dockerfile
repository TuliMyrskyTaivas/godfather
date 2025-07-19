FROM golang:1.24 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY internal/ ./internal/
COPY cmd/ ./cmd/
RUN ls -lR .
RUN CGO_ENABLED=0 GOOS=linux go build -o /godfather-cmd ./cmd/godfather-cmd
RUN ls -l

# Run the tests in the container
FROM build-stage AS run-test-stage
RUN go test -v ./...

# Deploy the application into a separate lean image
FROM gcr.io/distroless/base-debian11 AS build-release-stage
WORKDIR /
COPY --from=build-stage /godfather-cmd /godfather-cmd
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT [ "/godfather-cmd" ]