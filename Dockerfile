FROM golang:1.25 AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/p2p-fileshare-api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/p2p-fileshare-migrate ./cmd/migrate

FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=build /out/p2p-fileshare-api /app/p2p-fileshare-api
COPY --from=build /out/p2p-fileshare-migrate /app/p2p-fileshare-migrate
COPY --from=build /src/migrations /app/migrations

ENV HTTP_ADDR=:8080
EXPOSE 8080
EXPOSE 4001

USER nonroot:nonroot
ENTRYPOINT ["/app/p2p-fileshare-api"]

