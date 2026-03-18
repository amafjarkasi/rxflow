<div align="center">
  <img src="https://via.placeholder.com/150/000000/FFFFFF/?text=RxFlow" alt="RxFlow Logo" width="150" height="150" style="border-radius: 20%">

  # 🚀 RxFlow: The Next-Gen Clinical Orchestration Engine
  ### High-Throughput, AI-Powered, Event-Sourced E-Prescribing

  [![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://golang.org)
  [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql)](https://www.postgresql.org/)
  [![Redpanda](https://img.shields.io/badge/Redpanda-Streaming-FF4A5C?logo=apachekafka)](https://redpanda.com/)
  [![pgvector](https://img.shields.io/badge/AI-pgvector-8A2BE2)](https://github.com/pgvector/pgvector)
  [![Compliance](https://img.shields.io/badge/DEA-EPCS_Ready-28A745)]()
</div>

---

Welcome to **RxFlow**, a cutting-edge prescription orchestration engine built for massive scale, unwavering consistency, and AI-driven clinical safety.

This isn't just a basic CRUD API. This is an **enterprise-grade distributed system** combining the most robust software patterns with state-of-the-art NLP vector embeddings to literally save lives by catching anomalous prescribing behavior.

## ✨ Mind-Blowing Features

### 🧠 Clinical Decision Support (CDS) with AI Embeddings
*   **Vector Search (`pgvector`)**: Integrates with Postgres `pgvector` and HNSW indexing.
*   **Semantic Anomaly Detection**: Parses raw text directions ("Sig") and converts them into 1536-dimensional NLP embeddings.
*   **Life-Saving Precision**: Runs an Approximate Nearest Neighbor (ANN) cosine-similarity query against millions of historical, safe prescriptions. If a doctor prescribes "Take 10 pills daily" when the baseline norm is "Take 1 pill daily", the system instantly flags the semantic divergence before the prescription ever reaches a pharmacy.

### 🔐 Military-Grade Security & Encryption
*   **Field-Level AES-GCM**: Advanced `pkg/crypto` module natively intercepts domain events and encrypts sensitive PHI (Protected Health Information) using Authenticated Encryption (AES-256-GCM) *at rest*.
*   **Keyed HMAC-SHA256**: Deterministically hashes and salts patient identifiers.
*   **Zero-Knowledge Storage**: The underlying TimescaleDB event store never sees raw patient data.

### 🏛️ Unbreakable Architecture
*   **Event Sourcing (`internal/domain/prescription`)**: Every state change (Created -> Validated -> Routed -> Transmitted) is an immutable domain event. This creates a mathematically perfect audit trail for DEA EPCS compliance.
*   **Transactional Outbox Pattern**: Achieving dual-write consistency. Domain events and their corresponding Kafka messages are written to PostgreSQL atomically within a single `pgx` transaction.
*   **Idempotency Inbox**: "Exactly-once" delivery guarantees. The system handles massive distributed retries seamlessly.

### ⚡ Blistering Performance
*   **Target:** 5,000+ Transactions Per Second (TPS).
*   **Redpanda Streaming**: Drop-in Kafka replacement written in C++, utilizing thread-per-core architectures.
*   **Bounded Worker Pools**: Memory-safe concurrency ensuring the application never panics under extreme load.
*   **Circuit Breakers**: Advanced state-machine circuit breakers (`sony/gobreaker`) prevent downstream pharmacy outages from cascading back into the engine.

---

## 🗺️ Architectural Blueprint

```mermaid
graph TD
    A[EHR System<br/>FHIR R5 JSON] -->|POST /api/v1/rx| B(Ingestion API)

    subgraph RxFlow Core Orchestration
    B --> C{Prescription Aggregate<br/>Event Sourced}
    C -->|AES-GCM Encrypted| D[(PostgreSQL<br/>TimescaleDB)]
    C -->|Atomically Write| E[Outbox Table]

    E -.->|Polling / CDC| F[Outbox Relay]
    F -->|Produce| G[[Redpanda Kafka Cluster]]

    G -->|Consume| H[Routing Service / Worker Pool]
    H --> I{AI Clinical Decision Support}
    end

    subgraph AI Semantic Engine
    I -->|NLP Generate Embedding| J(HuggingFace / OpenAI)
    J -->|Vector[1536]| K[(pgvector HNSW Index)]
    K -->|Cosine Similarity <=>| I
    end

    I -->|Safe| L[Circuit Breaker]
    L -->|NCPDP SCRIPT 2023011| M[External Pharmacy Networks]
```

*(Note: If viewing on GitHub, imagine this as a beautiful Mermaid diagram!)*

---

## 🛠️ Technology Stack Breakdown

| Layer | Technology | Why We Chose It |
| :--- | :--- | :--- |
| **Language** | **Go 1.23+** | Raw performance, memory safety, and native concurrency (Goroutines). |
| **Messaging** | **Redpanda** | 10x faster than Kafka, zero-JVM footprint, incredibly stable tail latencies. |
| **Database** | **PostgreSQL 16** | The gold standard. ACID compliance and rock-solid reliability. |
| **Time-Series** | **TimescaleDB** | Hyper-efficient partitioning of time-stamped Event Sourcing streams. |
| **AI/ML** | **pgvector** | True multi-modal database. Doing vector similarity searches alongside relational data is a game changer. |
| **Observability** | **OpenTelemetry** | Vendor-agnostic distributed tracing (Jaeger) and metrics (Prometheus). |

---

## 🚀 Quick Start (Development)

Ready to see it in action?

### 1. Fire up the Cluster
```bash
# This brings up Postgres, Redpanda, Jaeger, and Prometheus locally
make docker-up
```

### 2. Prepare the Database (Vector AI & Timescale)
```bash
# Runs goose migrations and initializes pgvector extension
make migrate-up
```

### 3. Launch the Fleet
```bash
# Compiles binaries for ultimate speed
make build

# Start the ingestion node
./bin/ingestion-api &

# Start the transactional relay
./bin/outbox-relay &

# Start the consumer routing pool
./bin/routing-service &
```

---

## 📜 Development Roadmap

✅ **Phase 1: Foundation**: Go Modules, Docker Compose, Postgres schemas.<br>
✅ **Phase 2: Core Domain**: FHIR R5 data models, NCPDP SCRIPT mapper.<br>
✅ **Phase 3: Event Driven**: Event Sourced aggregates, Outbox/Inbox patterns.<br>
✅ **Phase 4: Security**: AES-256-GCM Field-Level Encryption, HMAC Salting. <br>
✅ **Phase 5: The Future (AI)**: Implement `pgvector` Clinical Decision Support.<br>
⏳ **Phase 6: Compliance**: Post-quantum cryptography (FIPS 203/204).<br>

---

<div align="center">
  <b>Built with ❤️ and extreme engineering rigor.</b>
</div>
