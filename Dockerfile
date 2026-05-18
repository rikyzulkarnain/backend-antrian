# syntax=docker/dockerfile:1.7

# --- build stage ---------------------------------------------------------
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache deps separately from source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static binary, no CGO. Tags 'osusergo,netgo' avoid glibc resolver use.
RUN CGO_ENABLED=0 GOOS=linux \
    go build -tags 'osusergo netgo' -trimpath -ldflags '-s -w' \
    -o /out/server ./cmd/server

# --- runtime stage -------------------------------------------------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=build /out/server ./server
# Bundled so the server / future migrate runs can find SQL files.
COPY migrations ./migrations

USER app
ENV PORT=8080
EXPOSE 8080

CMD ["./server"]
