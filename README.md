# Order Service

gRPC service for managing orders with CRUD operations.

## Configuration

Copy `.env.example` to `.env` and configure the following variables:

- `GRPC_PORT` - Port for gRPC server (default: 50051)
- `LOG_LEVEL` - Log level (debug, info, warn, error)

## Development

### Generate gRPC code:
```bash
make generate