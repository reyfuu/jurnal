FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/laporan-bisnis ./cmd/server

FROM scratch
WORKDIR /app
COPY --from=build /out/laporan-bisnis /app/laporan-bisnis
COPY web /app/web
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/laporan-bisnis"]

