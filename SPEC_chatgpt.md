# SPEC.md — Lightweight End-to-End WebSocket + gRPC Observability Load Testing Tool

---

# 1. Overview

This document defines the specification for a **lightweight internal load testing platform** designed to simulate realtime systems and measure backend performance through **WebSocket-driven traffic that indirectly exercises gRPC backend services**.

The system is intentionally designed to be:

* Simple (no Kubernetes)
* Single backend service (Go-based recommended)
* File/process-based execution model
* Suitable for QA, SRE, and backend engineering teams

---

# 2. Problem Statement

Modern realtime architectures typically look like:

```
Client
   ↓ WebSocket
Realtime Gateway
   ↓ gRPC
Backend Microservices
   ↓
Databases / Caches
```

Challenges:

* Internal gRPC services are not directly accessible
* Existing gRPC tools cannot simulate real client behavior
* WebSocket load testing tools lack backend observability
* Teams need end-to-end visibility without infra complexity

---

# 3. Goals

## Functional Goals

* Simulate large-scale WebSocket clients
* Send protobuf-encoded binary messages
* Trigger backend gRPC calls indirectly
* Measure end-to-end system performance
* Provide live test monitoring
* Generate HTML reports per execution
* Maintain historical test runs
* Support test types:

  * Smoke
  * Stress
  * Spike
  * Soak

---

## Non-Functional Goals

* Single binary backend (Go recommended)
* No Kubernetes dependency
* Minimal infrastructure requirements
* High observability via Prometheus/Grafana
* Safe execution with resource limits
* Easy setup and local deployment

---

# 4. High-Level Architecture

```
UI (HTML + JavaScript)
        ↓
Backend API (Go Service)
        ↓
Local WebSocket Load Runner (Process-based)
        ↓
Realtime Gateway
        ↓
Internal gRPC Microservices
        ↓
Prometheus / Grafana / Logs
```

---

# 5. System Components

# 5.1 UI (HTML + JavaScript)

## Purpose

Simple browser-based interface for:

* Creating tests
* Running tests
* Monitoring live execution
* Viewing reports

No frontend framework is required.

---

## Pages

### 1. Dashboard

* Active tests
* Recent test history
* System health
* Running connections

---

### 2. Create Test Page

Fields:

| Field                 | Type      |
| --------------------- | --------- |
| Test Name             | string    |
| WebSocket Endpoint    | string    |
| Event Type            | string    |
| Proto Schema Upload   | file      |
| Payload (JSON editor) | text      |
| Virtual Users         | number    |
| Messages per second   | number    |
| Duration              | string    |
| Ramp-up time          | string    |
| Auth token            | string    |
| Headers               | key-value |

---

### 3. Live Monitoring Page

Displays:

* Active websocket connections
* Messages/sec
* Error rate
* Latency (gateway + backend)
* CPU/Memory (backend)

Auto-refresh every 1 second.

---

### 4. Reports Page

* Download HTML report
* View historical runs
* Compare runs

---

# 5.2 Backend API (Single Orchestrator Service)

## Purpose

Acts as:

* API server
* test lifecycle manager
* process manager
* metrics collector
* report manager

---

## Recommended Tech

* Go (preferred)
* Single binary deployment

---

## Responsibilities

### 1. Test Management

* Create test
* Start test
* Stop test
* Track status

---

### 2. Validation

Enforces:

* max connections limits
* max message rate
* allowed endpoints
* valid protobuf schema

---

### 3. Process Management

Starts local WebSocket load runner process:

```
ws-runner --connections 5000 --rate 10000
```

Tracks PID and lifecycle.

---

### 4. State Management

Test states:

| State     | Meaning               |
| --------- | --------------------- |
| QUEUED    | waiting to start      |
| RUNNING   | active execution      |
| COMPLETED | finished successfully |
| FAILED    | error occurred        |
| CANCELLED | manually stopped      |

---

### 5. Report Management

Stores:

* HTML reports
* JSON metrics
* logs
* metadata

Storage:

* local filesystem (default)
* optional S3

---

### 6. Metrics Endpoint

Exposes:

```
/metrics
```

Prometheus-compatible metrics:

* active_connections
* messages_per_second
* error_rate
* test_duration
* backend_latency_p95

---

# 5.3 WebSocket Load Runner (Local Process)

## Purpose

Simulates real clients connecting to WebSocket gateway.

---

## Responsibilities

Each virtual client:

* opens WebSocket connection
* authenticates
* sends protobuf binary messages
* receives responses
* reconnects on failure

---

## Execution Model

Spawned as local process:

```
ws-runner
```

Arguments:

```
--connections N
--messages-per-second N
--duration T
--endpoint URL
--payload file
```

---

## Metrics Collection

Reports:

* connection success rate
* reconnect count
* message latency
* throughput
* failure rate

---

## Failure Handling

* retry reconnect
* exponential backoff
* circuit breaker for overload

---

# 5.4 Backend System Under Test

## Architecture Under Test

```
WebSocket Gateway
        ↓
Internal gRPC Services
        ↓
Databases / Caches
```

---

## Observability Required

To fully validate system:

* Prometheus metrics
* distributed tracing (optional)
* logs aggregation

---

# 6. Data Model

## Test Run Table

| Field        | Type      |
| ------------ | --------- |
| id           | string    |
| name         | string    |
| status       | enum      |
| created_at   | timestamp |
| started_at   | timestamp |
| completed_at | timestamp |
| config_json  | json      |
| report_path  | string    |

---

## Metrics Table

| Field               | Type   |
| ------------------- | ------ |
| test_id             | string |
| connections         | int    |
| messages_per_sec    | float  |
| error_rate          | float  |
| p95_latency         | float  |
| backend_latency_p95 | float  |

---

# 7. API Specification

## 7.1 Create Test

```
POST /api/tests
```

Request:

```json
{
  "name": "ws-stress-test",
  "endpoint": "ws://gateway:8080",
  "connections": 5000,
  "messagesPerSecond": 20000,
  "duration": "5m",
  "payload": {
    "event": "PLAYER_MOVE"
  }
}
```

Response:

```json
{
  "testId": "abc123",
  "status": "QUEUED"
}
```

---

## 7.2 Start Test

```
POST /api/tests/{id}/start
```

---

## 7.3 Stop Test

```
POST /api/tests/{id}/stop
```

---

## 7.4 Get Status

```
GET /api/tests/{id}
```

Response:

```json
{
  "status": "RUNNING",
  "connections": 4500,
  "messagesPerSecond": 18000,
  "errorRate": 0.02
}
```

---

## 7.5 Get Report

```
GET /api/tests/{id}/report
```

Returns HTML report.

---

# 8. Observability

## Prometheus Metrics

* ws_active_connections
* ws_messages_sent_total
* ws_messages_failed_total
* ws_latency_ms
* backend_grpc_latency_ms

---

## Grafana Dashboards

### Dashboard 1: WebSocket Load

* connections
* message throughput
* reconnect rate

### Dashboard 2: Backend Health

* gRPC latency
* error rate
* CPU/memory usage

---

# 9. Security Model

## Restrictions

* whitelist allowed endpoints
* prevent production targeting
* limit max connections per user
* limit max RPS

---

## Execution Safety

* max CPU usage per runner
* max memory limit
* timeout enforcement

---

## Authentication

* JWT-based auth
* role-based access control

---

# 10. Test Types

## 1. Smoke Test

* small connections
* quick validation

## 2. Stress Test

* high load until failure

## 3. Spike Test

* sudden load surge

## 4. Soak Test

* long duration stability

---

# 11. Deployment

## Single Node Deployment

```
UI (static files)
Backend (Go binary)
ws-runner (local process)
Prometheus
Grafana
```

---

# 12. Future Enhancements

* distributed runners
* multi-node load generation
* Kubernetes integration
* real-time tracing integration
* AI-based anomaly detection

---

# 13. Summary

This system provides a lightweight but powerful end-to-end platform to:

* simulate realtime WebSocket traffic
* indirectly load backend gRPC services
* observe full system performance
* generate reproducible performance reports

It prioritizes simplicity, observability, and practical engineering over heavy infrastructure complexity.
