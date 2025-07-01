# Draft Gateway Service

Real-time draft event streaming service that consumes events from NATS JetStream and broadcasts them to WebSocket clients.

## Running

```bash
# From this directory
go run main.go

# Or build and run
go build -o draft-gateway .
./draft-gateway
```

## Environment Variables

The service uses the same `.env` file as the main application:

- `GATEWAY_PORT` - HTTP server port (default: 8081)
- `NATS_URL` - NATS server URL (default: nats://localhost:4222)
- Database config: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`

## Endpoints

- `ws://localhost:8081/ws/draft?draft_id={uuid}&user_id={id}` - WebSocket connection
- `GET /api/drafts/{id}/state` - Get current draft state
- `GET /api/drafts/active` - List active drafts  
- `GET /ws/stats` - WebSocket connection statistics
- `GET /health` - Health check
- `GET /info` - Service info

## Prerequisites

1. PostgreSQL database running
2. NATS server with JetStream enabled
3. Outbox listener service running (to publish events to NATS)

## Architecture

```
Dynasty API → Outbox Table → Outbox Listener → NATS JetStream → Draft Gateway → WebSocket Clients
```