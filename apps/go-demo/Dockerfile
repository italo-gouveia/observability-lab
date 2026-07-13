FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -o /app .

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
COPY --from=build /app /app
EXPOSE 8080
ENTRYPOINT ["/app"]
