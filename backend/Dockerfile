FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/nexus-mini ./cmd/nexus-mini

FROM alpine:3.20
RUN adduser -D nexus
USER nexus
COPY --from=build /out/nexus-mini /
EXPOSE 8084
CMD ["/nexus-mini", "serve"]
