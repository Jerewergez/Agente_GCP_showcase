# Agente_GCP — Go Backend (Cloud Run)
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY backend/server/ .
RUN go build -o agente-gcp .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/agente-gcp .
EXPOSE 8080
CMD ["./agente-gcp"]
