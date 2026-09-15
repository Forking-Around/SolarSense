FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /solarsense ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /solarsense /solarsense
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/solarsense"]
