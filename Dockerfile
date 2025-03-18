FROM golang:1.24.1 AS build

WORKDIR /tmp/vsexporter

COPY ./go.mod /tmp/vsexporter/go.mod

RUN go mod download

COPY . /tmp/vsexporter

RUN CGO_ENABLED=0 go build -o bin/vsexporter ./cmd/vsexporter

FROM scratch

COPY --from=build /tmp/vsexporter/bin/vsexporter /app/vsexporter

ENTRYPOINT ["/app/vsexporter"]
