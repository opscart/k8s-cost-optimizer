# Changelog

All notable changes to k8s-cost-optimizer will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Prometheus P95/P99 integration for accurate historical metrics
- Automatic fallback to metrics-server when Prometheus unavailable
- Multi-cloud pricing support (Azure real-time API, AWS/GCP defaults)
- Comprehensive test suite (94% coverage)
- E2E testing with real Kubernetes clusters
- Cloud provider auto-detection
- Configurable safety buffers and lookback periods
- PostgreSQL storage for recommendation history
- Audit logging for executed recommendations
- kubectl command generation

### Testing
- Unit tests for all core packages
- Integration tests with real cloud APIs
- Contract tests with recorded responses
- E2E tests on real minikube cluster

## [0.1.0] - Initial Development

### Added
- Basic Kubernetes cluster scanning
- Pod resource analysis
- Cost calculation with default pricing
- Recommendation engine (RIGHT_SIZE, SCALE_DOWN, NO_ACTION)
- CLI interface with namespace filtering
- metrics-server integration
