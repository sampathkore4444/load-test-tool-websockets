# Architecture Diagram: OpenCode-Adapted Load Testing Tool

```mermaid
graph TD
    %% OpenCode Integration Layer
    subgraph OpenCode_Environment[OpenCode Development Environment]
        direction TB
        CodeEditor[Code Editor] <-->|Edits| OpenCodeAgent[OpenCode AI Agent]
        CodeEditor <-->|Triggers| TestControlPanel[Test Control Panel]
        TestControlPanel <-->|Controls| BackendAPI[Backend API Service]
        OpenCodeAgent <-->|Receives| PerformanceInsights[Performance Insights Engine]
        PerformanceInsights <-->|Feeds| OpenCodeAgent
    end

    %% Core Load Testing System
    subgraph LoadTestingSystem[Load Testing System]
        direction TB
        TestControlPanel -->|API Calls| BackendAPI
        BackendAPI -->|Spawns/Manages| LoadRunner[WebSocket Load Runner Process]
        LoadRunner -->|WebSocket Traffic| RealtimeGateway[Realtime Gateway]
        RealtimeGateway -->|gRPC Calls| BackendServices[Internal gRPC Microservices]
        BackendServices -->|Responses| RealtimeGateway
        RealtimeGateway -->|Responses| LoadRunner
        
        %% Observability
        BackendServices -->|Metrics/Logs| Prometheus[Prometheus]
        LoadRunner -->|Metrics| Prometheus
        Prometheus -->|Data| Grafana[Grafana]
        Grafana -->|Dashboards| Developer[Developer View]
        Grafana -->|Alerts| PerformanceInsights
    end

    %% Data and Storage
    subgraph DataLayer[Data & Storage]
        direction TB
        BackendAPI -->|Test Configs/Results| TestDatabase[Test Database<br/>(SQLite/FS)]
        BackendAPI -->|Reports| ReportStorage[Report Storage<br/>(Local/S3)]
        TestDatabase -->|Historical Data| PerformanceInsights
        ReportStorage -->|Historical Reports| Developer
    end

    %% System Under Test (Can be Local/Shared)
    subgraph SUT[System Under Test]
        direction TB
        RealtimeGateway
        BackendServices
        Databases[Databases/Caches]
        BackendServices -->|Reads/Writes| Databases
    end

    %% Styling
    classDef env fill:#f9f9f9,stroke:#333,stroke-width:1px;
    classDef core fill:#e3f2fd,stroke:#1565c0,stroke-width:2px;
    classDef obs fill:#fff3e0,stroke:#ef6c00,stroke-width:1px;
    classDef data fill:#f3e5f5,stroke:#6a1b9a,stroke-width:1px;
    classDef sut fill:#e8f5e8,stroke:#2e7d32,stroke-width:1px;
    
    class OpenCode_Environment env;
    class LoadTestingSystem core;
    class Grafana,Prometheus obs;
    class TestDatabase,ReportStorage data;
    class RealtimeGateway,BackendServices,Databases sut;
    
    %% Labels
    style OpenCode_Environment fill:#f9f9f9,stroke:#9e9e9e,stroke-dasharray: 5 5
    style LoadTestingSystem fill:#e3f2fd,stroke:#1565c0,stroke-width:2px
    style SUT fill:#e8f5e8,stroke:#2e7d32,stroke-width:2px
```

## Diagram Description

This architecture diagram illustrates the OpenCode-adapted load testing tool showing:

1. **OpenCode Environment Layer**: Where developers interact with the code editor and AI agent
2. **Load Testing System Core**: The main components (UI, backend API, load runner) 
3. **Observability Layer**: Prometheus/Grafana for metrics collection and visualization
4. **Data Layer**: Storage for test configurations, results, and reports
5. **System Under Test**: The actual services being tested (gateway → gRPC → databases)

Key OpenCode-specific integrations shown:
- Direct connection between OpenCode AI agent and Performance Insights Engine
- Test control panel integration in the IDE
- Automatic triggering from file changes in the code editor
- Performance insights feeding back to inform AI suggestions

The diagram maintains the core architecture from SPEC_chatgpt.md while highlighting the enhanced integration points for OpenCode/AI-assisted development workflows.