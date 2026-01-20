FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETPLATFORM
RUN echo "Building for $TARGETPLATFORM"
RUN CGO_ENABLED=0 GOOS=linux \
    $(case $TARGETPLATFORM in \
        "linux/amd64") echo "GOARCH=amd64";; \
        "linux/arm64") echo "GOARCH=arm64";; \
        "linux/arm/v7") echo "GOARCH=arm";; \
    esac) \
    go build -o vox-nlu main.go



FROM --platform=$TARGETPLATFORM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir rasa==3.6.0

COPY --from=builder /app/vox-nlu-adapter ./
COPY config/ config/


CMD ["./vox-nlu"]
