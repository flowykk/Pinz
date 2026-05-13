FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /out/geo-stub ./...

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/geo-stub /geo-stub
EXPOSE 8080
ENTRYPOINT ["/geo-stub"]
