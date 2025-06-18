FROM golang:1.24-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION
RUN CGO_ENABLED=1 go build -ldflags="-w -X 'github.com/jj-style/eventpix.Version=${VERSION}'" -trimpath -o eventpix main.go

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=builder /src/eventpix .

EXPOSE 8080
ENTRYPOINT [ "/app/eventpix" ]