# Kratos Project Template

## Install Kratos

```
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
```

## Create a service

```
# Create a template project
kratos new server

cd server
# Add a proto template
kratos proto add api/server/server.proto
# Generate the proto code
kratos proto client api/server/server.proto
# Generate the source code of service by proto file
kratos proto server api/server/server.proto -t internal/service

go generate ./...
go build -o ./bin/ ./...
./bin/server -conf ./configs
```

## Generate other auxiliary files by Makefile

```
# Download and update dependencies
make init
# Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file
make api
# Generate all files
make all
```

## Automated Initialization (wire)

```
# install wire
go get github.com/google/wire/cmd/wire

# generate wire
cd cmd/server
wire
```

## Docker

### 🚀 Quick Start with Docker Compose (Recommended)

Run the entire stack (Kafka + MongoDB + Application) with auto-seeded data:

**Linux/Mac:**

```bash
chmod +x start.sh
./start.sh
```

**Windows:**

```bash
start.bat
```

**Or manually:**

```bash
# Create .env file
cp env.template .env

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f
```

📚 **Detailed Documentation:**

- [🚀 QUICKSTART.Docker.md](./QUICKSTART.Docker.md) - Quick start guide (Vietnamese)
- [📖 README.Docker.md](./README.Docker.md) - Full Docker documentation

### What's Included?

✅ **Kafka** - Message queue (port 9092)  
✅ **MongoDB** - Database with **auto-seeded data** (port 27017)

- 6 Companies (One Mount, MB Bank, Techcombank, etc.)
- 7 Job Postings with full details

✅ **JoblyBE Application** - Backend API

- HTTP API: `http://localhost:8000`
- WebSocket: `ws://localhost:8000/ws`
- gRPC: `localhost:9090`

### Traditional Docker Build

```bash
# build
docker build -t joblybe .

# run
docker run --rm -p 8000:8000 -p 9000:9000 -v $(pwd)/configs:/data/conf joblybe
```
