# -------- Stage 1: build Go binary --------
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -ldflags="-X 'main.Version=$(cat VERSION)'" -o vox-nlu main.go

# -------- Stage 2: runtime --------
FROM rasa/rasa:main-full

USER root
WORKDIR /app

COPY --from=builder /app/vox-nlu ./

ENTRYPOINT ["./vox-nlu"]
