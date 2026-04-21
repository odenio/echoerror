FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -mod=vendor -o /server ./cmd/server
RUN CGO_ENABLED=0 go build -mod=vendor -o /client ./cmd/client

FROM gcr.io/distroless/static-debian12
COPY --from=build /server /server
COPY --from=build /client /client
