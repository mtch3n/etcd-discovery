FROM golang:1.22-alpine as builder
ARG TARGETARCH

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY . .

# Build
RUN GOOS=linux CGO_ENABLED=0 go build -a -o build/esd ./main.go

# Final image.
FROM golang:1.22-alpine

ENV GO_VERSION 1.22

COPY --from=builder /workspace/build/esd /usr/local/bin/esd

ENTRYPOINT ["/usr/local/bin/esd"]