FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server
RUN CGO_ENABLED=0 go build -o /client ./cmd/client

FROM gcr.io/distroless/static-debian12
COPY --from=build /server /server
COPY --from=build /client /client
