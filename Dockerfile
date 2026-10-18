FROM node:22-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend ./
RUN npm run build

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/laporan-bisnis ./cmd/server

FROM scratch
WORKDIR /app
COPY --from=build /out/laporan-bisnis /app/laporan-bisnis
COPY --from=web /src/web/dist /app/web/dist
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/laporan-bisnis"]
