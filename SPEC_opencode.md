# SPEC_opencode.md — OpenCode-Adapted Load Testing Tool Specification

## Analysis of SPEC_chatgpt.md

After reviewing the existing SPEC_chatgpt.md, I've identified this as a comprehensive specification for a lightweight end-to-end WebSocket + gRPC observability load testing tool. The specification covers:

1. **Overview**: A lightweight internal load testing platform for simulating realtime systems
2. **Problem Statement**: Challenges with testing internal gRPC services through WebSocket gateways
3. **Goals**: Functional and non-functional objectives
4. **Architecture**: High-level system design with UI, backend, load runner, and observed system
5. **Components**: Detailed breakdown of UI, backend API, WebSocket load runner, and system under test
6. **Data Model**: Test run and metrics table structures
7. **API Specification**: REST endpoints for test management
8. **Observability**: Prometheus metrics and Grafana dashboards
9. **Security Model**: Restrictions and execution safety measures
10. **Test Types**: Smoke, stress, spike, and soak tests
11. **Deployment**: Single node deployment approach
12. **Future Enhancements**: Planned improvements

This specification is well-structured and implementable as-is. The OpenCode adaptation focuses on how this tool would be utilized within the OpenCode/AI-assisted development workflow.

---

# 1. Overview

This document defines the OpenCode-adapted specification for the **lightweight internal load testing platform** designed to simulate realtime systems and measure backend performance through WebSocket-driven traffic that indirectly exercises gRPC backend services.

The OpenCode adaptation emphasizes:
- AI-assisted development and testing workflows
- Integration with OpenCode's capabilities for rapid prototyping
- Enhanced observability for AI-assisted performance analysis
- Streamlined deployment for development environments

---

# 2. Problem Statement (OpenCode Context)

In AI-assisted development environments like OpenCode:
- Developers need rapid feedback on performance implications of code changes
- Traditional load testing setups are too heavyweight for iterative development
- There's a need to validate that AI-generated code doesn't introduce performance regressions
- Developers benefit from immediate visualization of performance impact

---

# 3. Goals (OpenCode Adapted)

## Functional Goals
* Simulate WebSocket clients to validate AI-generated WebSocket handlers
* Test protobuf-encoded message handling in AI-generated gRPC services
* Measure performance impact of AI-suggested optimizations
* Provide live feedback during development cycles
* Generate comparative reports for code review processes
* Support rapid iteration with fast test startup/shutdown

## Non-Functional Goals
* Minimal setup time for spontaneous testing sessions
* Low resource consumption to not interfere with development tools
* Seamless integration with OpenCode's workflow
* Quick visualization of performance trends
* Safe execution in shared development environments

---

# 4. High-Level Architecture (OpenCode View)

```
OpenCode Workspace
         ↓ (AI-assisted editing)
Developer → [Code Editor] ←→ [OpenCode Agent]
         ↓
UI (HTML + JavaScript) ←→ [Test Control Panel]
         ↓
Backend API (Go Service) ←→ [Process Manager]
         ↓
Local WebSocket Load Runner (ephemeral process)
         ↓
Realtime Gateway (target system)
         ↓
Internal gRPC Microservices (system under test)
         ↓
Prometheus / Grafana / Logs [Observability Layer]
         ↓
[OpenCode Performance Insights] ←→ [AI Analysis]
```

Key OpenCode Adaptations:
- Ephemeral test runners spun up on-demand
- Integration with OpenCode's file watching for auto-test triggering
- Performance insights fed back to OpenCode for AI analysis
- Minimal persistence - focus on immediate feedback

---

# 5. System Components (OpenCode Specific)

## 5.1 UI Enhancements for OpenCode
* Embeddable test control panel that can dock within OpenCode IDE
* Keyboard shortcuts for common test operations (Ctrl+Shift+T to start/stop)
* Real-time performance annotations in code editor
* Auto-generated test configurations based on open files
* "Test this change" button next to modified WebSocket/gRPC code

## 5.2 Backend API Optimizations
* File system watcher to auto-detect protobuf/schema changes
* Hot-reload capability for test configurations
* Integration with OpenCode's context awareness
* Lightweight SQLite backend for dev (instead of full PostgreSQL)
* WebSocket endpoint for pushing metrics to OpenCode agent

## 5.3 WebSocket Load Runner Improvements
* Ultra-fast startup (<1s) for quick validation tests
* Deterministic mode for reproducible AI-assisted testing
* Resource usage capping to protect development environment
* Detailed failure analysis for debugging AI-generated code
* OpenCode-specific metrics format for AI consumption

## 5.4 Enhanced Observability for AI
* Custom metrics exposing code-level performance hints
* Correlation between code changes and performance deltas
* Anomaly detection flags for OpenCode to highlight
* Baseline establishment for "known good" performance
* Regression detection algorithms

---

# 6. Data Model Extensions

## Enhanced Test Run Table
| Field | Type | OpenCode Purpose |
|-------|------|------------------|
| git_commit | string | Link test to specific code version |
| ai_suggested | boolean | Flag tests initiated by AI |
| performance_baseline | float | Reference for regression detection |
| code_paths_affected | json array | Files modified in this test session |
| opencontext_summary | text | Summary of OpenCode session context |

## Enhanced Metrics Table
| Field | Type | OpenCode Purpose |
|-------|------|------------------|
| test_id | string | Reference to test run |
| cpu_delta | float | CPU usage change from baseline |
| memory_delta | float | Memory usage change from baseline |
| latency_regression_score | float | 0-1 score indicating regression likelihood |
| ai_confidence_impact | float | How much this test affects AI confidence in code |

---

# 7. API Specification Enhancements

## 7.6 Get Performance Insights (OpenCode Specific)
```
GET /api/insights/{testId}
```
Returns AI-interpretable performance data including:
- Regression likelihood scores
- Resource usage trends
- Correlation with recent code changes
- Suggested areas for AI investigation

## 7.7 Auto-Trigger from File Changes
```
POST /api/watch
```
Body:
```json
{
  "paths": ["src/**/*.go", "proto/**/*.proto"],
  "debounceMs": 2000,
  "onChange": "run-smoke-test"
}
```

Enables OpenCode to automatically validate performance impact of saves.

---

# 8. OpenCode-Specific Observability

## Prometheus Metrics Extensions
* `opencode_test_session_active`: Whether an OpenCode-initiated test is running
* `opencode_code_change_correlation`: Correlation coefficient between recent edits and metrics
* `opencode_ai_confidence_delta`: Change in AI confidence score based on performance
* `opencode_baseline_deviation`: Percentage deviation from established baseline

## Grafana Dashboard: AI Performance Assistant
* Panel showing performance trend over recent commits
* Alert when performance deviates beyond AI-acceptable thresholds
* View of which code paths most affect performance
* Resource usage prediction for proposed changes

---

# 9. Security Model (OpenCode Adapted)

* Automatic restriction to localhost/127.0.0.1 in development mode
* Integration with OpenCode's permission system
* No persistent storage of sensitive data between sessions
* Automatic cleanup of test artifacts on OpenCode workspace switch
* Dev-only mode with relaxed limits for experimentation

---

# 10. Test Types (OpenCode Workflow Integrated)

## 1. Smoke Test (AI Validation)
* Triggered automatically on file save
* Validates basic connectivity and message flow
* Provides go/no-go signal for AI to continue with current approach

## 2. Stress Test (Manual Validation)
* Initiated by developer for deeper investigation
* Runs before major code reviews
* Establishes performance boundaries for AI suggestions

## 3. Regression Detection (Continuous)
* Compares against baseline from "known good" commit
* Flags performance degradations for AI attention
* Integrates with OpenCode's suggestion confidence scoring

## 4. Exploratory Test (AI-Driven)
* OpenCode agent initiates tests to understand system behavior
* Used when AI needs to gather performance characteristics
* Results inform future code suggestions

---

# 11. Deployment (OpenCode Development Mode)

```
OpenCode Workspace
         ↓
[Dev UI] (embedded in IDE)
         ↓
[Dev Backend] (Go binary, launched by OpenCode)
         ↓
[Dev Load Runner] (ephemeral process per test)
         ↓
[Target Services] (can be local dev services or shared dev environment)
         ↓
[Local Observability] (Prometheus/Grafana in dev containers)
         ↓
[OpenCode AI] ←→ [Performance Insights Engine]
```

Key aspects:
- All components can be launched via OpenCode commands
- Zero-configuration mode for spontaneous testing
- Automatic port allocation to avoid conflicts
- Temporary data stores cleaned on session end
- Option to connect to shared development environment

---

# 12. OpenCode Integration Points

## Command Palette Integration
* `Load Test: Create Quick Test`
* `Load Test: Run Smoke on Current File`
* `Load Test: Compare with Baseline`
* `Load Test: Show Performance Timeline`

## File Watchers
* Auto-discovery of WebSocket handler files
* Protobuf file change detection
* Configuration file monitoring for test parameters

## AI Context Enhancement
* Performance data included in OpenCode's context window
* Test results inform AI's confidence in suggested changes
* Historical performance trends available to AI for better suggestions

## Version Control Integration
* Automatic association of tests with git commits
* Performance data attached to pull requests
* Baseline establishment from main branch

---

# 13. OpenCode-Specific Future Enhancements

* Natural language test creation: "Create a stress test for the player movement handler"
* AI-generated test scenarios based on code analysis
* Predictive performance modeling for suggested code changes
* Automated bisecting to find performance-introducing commits
* Integration with OpenCode's pair programming mode for collaborative performance tuning
* Export of test configurations as code snippets for documentation

---

# 14. Summary

This OpenCode-adapted specification maintains all core functionality of the original SPEC_chatgpt.md while enhancing it for AI-assisted development workflows. The key improvements focus on:

1. Reduced friction for spontaneous performance validation
2. Tight integration with OpenCode's AI capabilities
3. Immediate feedback loops for code-performance correlation
4. Enhanced observability tailored for AI consumption
5. Development-friendly deployment model

The system enables developers to leverage OpenCode's AI assistance not just for writing code, but for immediately understanding and optimizing the performance implications of their changes—creating a tighter code-performance feedback loop essential for high-quality, performant software development.