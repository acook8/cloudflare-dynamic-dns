FROM golang:1.17 AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./

RUN go build -o /cloudflare-ddns


FROM gcr.io/distroless/base-debian10

WORKDIR /
COPY --from=build /cloudflare-ddns /cloudflare-ddns
USER nonroot:nonroot

ENTRYPOINT ["/cloudflare-ddns"]