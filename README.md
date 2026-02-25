# RxFlow - High-Volume Prescription Orchestration Engine

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![FHIR R5](https://img.shields.io/badge/FHIR-R5-E73C3E?style=flat)](https://hl7.org/fhir/R5/)
[![NCPDP SCRIPT](https://img.shields.io/badge/NCPDP%20SCRIPT-v2023011-1E88E5?style=flat)](https://www.ncpdp.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> **Event-driven prescription orchestration engine transforming FHIR R5 to NCPDP SCRIPT v2023011 at 5,000+ TPS with DEA EPCS compliance**

RxFlow (`go-oec`) is a high-performance prescription processing platform designed for healthcare interoperability. It bridges modern EHR systems using HL7 FHIR R5 with pharmacy networks via NCPDP SCRIPT v2023011, implementing event sourcing patterns for complete DEA audit compliance.

---

## Project Status

| Aspect | Status |
|--------|--------|
| **Maturity** | Pre-Alpha / Development |
| **Core Engine** | Implemented |
| **FHIR R5 Models** | Complete |
| **NCPDP SCRIPT v2023011** | Complete |
| **Event Sourcing** | Complete |
| **Distributed Patterns** | Complete |
| **API Layer** | Implemented |
| **Security/EPCS** | Stub Implementation |
| **Production Ready** | No |

### What's Working

- FHIR R5 MedicationRequest ingestion via REST API
- NCPDP SCRIPT v2023011 NewRx message generation
- Bidirectional FHIR-to-SCRIPT transformation
- Event-sourced prescription aggregate with TimescaleDB
- Transactional outbox pattern for reliable messaging
- Idempotency inbox for exactly-once processing
- Circuit breaker pattern for pharmacy routing
- Worker pool for concurrent processing
- OpenTelemetry tracing and Prometheus metrics
- Docker Compose development environment

### What's Stubbed/Planned

- DEA EPCS two-factor authentication
- Digital signature infrastructure (PKI)
- Surescripts production integration
- State Prescription Monitoring Program (PMP) integration
- Drug-drug interaction checking
- Post-quantum cryptography (FIPS 203/204)

---

## Architecture Overview

```
                                    RxFlow Architecture
    
    +------------------+     +------------------+     +------------------+
    |    EHR System    |     |   Ingestion API  |     |    Redpanda      |
    |  (FHIR R5 JSON)  +---->+  /api/v1/...     +---->+  (Kafka Topics)  |
    +------------------+     +--------+---------+     +--------+---------+
                                      |                        |
                                      v                        v
                             +------------------+     +------------------+
                             |   PostgreSQL     |     |  Routing Service |
                             |  + TimescaleDB   |<----+  + Worker Pool   |
                             |  + pgvector      |     |  + Circuit Break |
                             +------------------+     +--------+---------+
                                      ^                        |
                                      |                        v
                             +------------------+     +------------------+
                             |  Outbox Relay    |     |  Pharmacy        |
                             |  (Transactional) |     |  (NCPDP SCRIPT)  |
                             +------------------+     +------------------+
```

### Key Design Patterns

| Pattern | Purpose |
|---------|---------|
| **Event Sourcing** | Immutable audit trail for DEA compliance |
| **CQRS** | Separate read/write models for performance |
| **Transactional Outbox** | Atomic DB + message queue writes |
| **Idempotency Inbox** | Exactly-once message processing |
| **Circuit Breaker** | Prevent cascade failures to pharmacies |
| **Worker Pool** | Bounded concurrency for throughput control |

---

## Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| **Language** | Go 1.23+ | High-performance, type-safe |
| **Message Broker** | Redpanda | Kafka-compatible, low-latency |
| **Database** | PostgreSQL 16 | ACID compliance, JSON support |
| **Time-Series** | TimescaleDB | Event compression, retention |
| **Embeddings** | pgvector | AI-powered drug matching |
| **HTTP Router** | chi | Lightweight, composable |
| **Kafka Client** | franz-go | High-throughput batching |
| **Tracing** | OpenTelemetry + Jaeger | Distributed observability |
| **Metrics** | Prometheus + Grafana | Performance monitoring |

---

## Quick Start

### Prerequisites

- Go 1.23+
- Docker & Docker Compose
- Make

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/go-oec.git
cd go-oec
```

### 2. Start Infrastructure

```bash
# Start Redpanda, PostgreSQL, Jaeger, Prometheus, Grafana
make docker-up

# Run database migrations
make migrate-up
```

### 3. Build and Run

```bash
# Build all services
make build

# Run the ingestion API
./bin/ingestion-api

# In another terminal, run the outbox relay
./bin/outbox-relay

# In another terminal, run the routing service
./bin/routing-service
```

### 4. Test the API

```bash
# Create a prescription
curl -X POST http://localhost:8080/api/v1/prescriptions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d @test/fixtures/medication_request_lisinopril.json
```

---

## API Reference

### Create Prescription

```http
POST /api/v1/prescriptions
Content-Type: application/json
X-API-Key: <api-key>
```

**Request Body:**
```json
{
  "medication_request": { /* FHIR R5 MedicationRequest */ },
  "patient": { /* FHIR R5 Patient */ },
  "practitioner": { /* FHIR R5 Practitioner */ },
  "pharmacy": { /* FHIR R5 Organization */ }
}
```

**Response:**
```json
{
  "id": "uuid",
  "status": "pending",
  "idempotency_key": "sha256-hash",
  "is_controlled": false,
  "created_at": "2026-02-25T10:00:00Z"
}
```

### Get Prescription

```http
GET /api/v1/prescriptions/{id}
X-API-Key: <api-key>
```

### Get Prescription Events

```http
GET /api/v1/prescriptions/{id}/events
X-API-Key: <api-key>
```

### Route Prescription to Pharmacy

```http
POST /api/v1/prescriptions/{id}/route
X-API-Key: <api-key>
```

---

## Project Structure

```
go-oec/
├── cmd/                          # Service entry points
│   ├── ingestion-api/            # HTTP API service
│   ├── outbox-relay/             # Outbox pattern processor
│   └── routing-service/          # Pharmacy routing consumer
├── internal/
│   ├── api/                      # HTTP handlers and middleware
│   ├── domain/prescription/      # Event-sourced aggregate
│   ├── fhir/r5/                  # FHIR R5 data models
│   ├── infrastructure/
│   │   ├── postgres/             # PostgreSQL repositories
│   │   └── redpanda/             # Kafka producer/consumer
│   ├── ncpdp/
│   │   ├── mapper/               # FHIR <-> SCRIPT transformation
│   │   └── script2023011/        # NCPDP SCRIPT v2023011 types
│   └── observability/            # Tracing and metrics
├── pkg/
│   ├── circuitbreaker/           # Circuit breaker pattern
│   ├── idempotency/              # Inbox pattern
│   └── workerpool/               # Bounded worker pool
├── deploy/docker/                # Docker Compose environment
├── migrations/                   # PostgreSQL migrations
└── test/fixtures/                # Test data (FHIR JSON)
```

---

## Kafka Topics

| Topic | Purpose |
|-------|---------|
| `prescription.events` | Domain events (event sourcing) |
| `prescription.commands` | Command messages |
| `routing.requests` | Pharmacy routing requests |
| `routing.responses` | Pharmacy responses |
| `audit.trail` | DEA audit events |
| `ncpdp.outbound` | SCRIPT messages to pharmacies |
| `ncpdp.inbound` | SCRIPT responses from pharmacies |
| `dead.letter` | Failed message handling |

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | `postgres://...` | PostgreSQL connection |
| `REDPANDA_BROKERS` | `localhost:9092` | Kafka brokers |
| `OTEL_ENDPOINT` | `localhost:4317` | OpenTelemetry collector |
| `LOG_LEVEL` | `info` | Logging verbosity |
| `API_KEYS` | - | Comma-separated API keys |

---

## Development Roadmap

### Phase 1: Foundation (Complete)
- [x] Go module setup and Makefile
- [x] Docker Compose infrastructure
- [x] Database migrations with TimescaleDB
- [x] FHIR R5 data models
- [x] NCPDP SCRIPT v2023011 structures

### Phase 2: Core Domain (Complete)
- [x] Event-sourced prescription aggregate
- [x] FHIR-to-SCRIPT mapper
- [x] Idempotency key generation
- [x] Transactional outbox pattern
- [x] Idempotency inbox pattern

### Phase 3: Orchestration (Complete)
- [x] Redpanda producer/consumer
- [x] Worker pool implementation
- [x] Circuit breaker pattern
- [x] Routing service

### Phase 4: Integration (In Progress)
- [ ] Pharmacy response handling
- [ ] Structured Sig mapping
- [ ] NPI/DEA validation
- [ ] Drug database integration

### Phase 5: Security & Compliance (Planned)
- [ ] DEA EPCS two-factor auth
- [ ] Digital signature (PKI)
- [ ] Patient hash salting
- [ ] Field-level encryption
- [ ] Post-quantum cryptography

### Phase 6: Production Readiness (Planned)
- [ ] Surescripts integration
- [ ] PMP integration
- [ ] Performance optimization
- [ ] Load testing (5,000 TPS)
- [ ] Security audit

---

## Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| **Throughput** | 5,000 TPS | Not tested |
| **Latency (p99)** | < 100ms | Not tested |
| **Availability** | 99.99% | N/A |
| **Event Retention** | 7 years | Configured |

### Tuning Parameters

```go
// Producer (franz-go)
BatchMaxBytes:     16 * 1024 * 1024,  // 16MB batches
Linger:            50 * time.Millisecond,
MaxBufferedRecords: 1_000_000,

// Worker Pool
Workers:    100,
QueueSize:  10_000,
MaxRetries: 3,
```

---

## Observability

### Jaeger UI (Tracing)
```
http://localhost:16686
```

### Prometheus (Metrics)
```
http://localhost:9090
```

### Grafana (Dashboards)
```
http://localhost:3000
```

### Redpanda Console
```
http://localhost:8081
```

---

## Testing

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run with coverage
make test-coverage
```

---

## Contributing

We welcome contributions! Please see our contributing guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Standards

- Follow Go idioms and conventions
- Maintain test coverage above 80%
- Document public APIs
- Use meaningful commit messages

---

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- [HL7 FHIR](https://hl7.org/fhir/) - Healthcare interoperability standard
- [NCPDP SCRIPT](https://www.ncpdp.org/) - E-prescribing standard
- [Redpanda](https://redpanda.com/) - Kafka-compatible streaming
- [TimescaleDB](https://www.timescale.com/) - Time-series database
- [franz-go](https://github.com/twmb/franz-go) - High-performance Kafka client

---

## Contact

For questions or support, please open an issue on GitHub.

---

**Disclaimer:** This software is provided for development and demonstration purposes. It is not certified for production use in healthcare settings. Proper regulatory compliance, security audits, and certifications are required before processing real patient data.
