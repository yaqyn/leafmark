FROM golang:1.27.2-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /leafmark .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/* && groupadd --gid 10001 leafmark && useradd --uid 10001 --gid leafmark --no-create-home leafmark && mkdir /data && chown leafmark:leafmark /data
COPY --from=build /leafmark /usr/local/bin/leafmark
WORKDIR /data
USER 10001:10001
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["leafmark"]
