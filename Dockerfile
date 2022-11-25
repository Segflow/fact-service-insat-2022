FROM golang:1.18-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 go build -o ./fact


FROM scratch
COPY --from=build /app/fact /app/fact
CMD ["/app/fact"]