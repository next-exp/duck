# NEXT-100 Data Acquisition System (Duck-Docker)

Data acquisition software for the [NEXT experiment](https://next.ific.uv.es/next/) at the 
[Laboratorio Subterráneo de Canfranc (LSC)](https://www.lsc-canfranc.es/). This system handles 
readout from detector electronics, event building, and provides a web-based control interface 
for the NEXT-100 detector.

## Overview

The NEXT (Neutrino Experiment with a Xenon TPC) collaboration searches for neutrinoless double 
beta decay (ββ0ν) using high-pressure xenon gas Time Projection Chambers with electroluminescent 
amplification. The NEXT-100 detector contains ~100 kg of enriched xenon at 13.5 bar and features:

- **60 PMTs** in the Energy Plane for energy measurement and S1 detection
- **3,584 SiPMs** in the Tracking Plane for 3D event reconstruction
- **13 DAQ modules** (7 EP + 6 TP) reading sensor data via ATCA blades
- **7 DAQ servers** processing data with up to 875 MB/s combined throughput

This repository contains the complete DAQ software stack, from detector readout to event 
building and user interface.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              User Interface                                  │
│                         ┌─────────────────────┐                             │
│                         │   Vue.js Web GUI    │                             │
│                         │   (localhost:5173)  │                             │
│                         └──────────┬──────────┘                             │
│                                    │ ConnectRPC                              │
│                         ┌──────────▼──────────┐                             │
│                         │     API Server      │                             │
│                         │   (localhost:1323)  │                             │
│                         └──────────┬──────────┘                             │
│                                    │ gRPC                                    │
└────────────────────────────────────┼────────────────────────────────────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────────┐
│                           Event Building Layer                               │
│            ┌───────────────────────┼───────────────────────┐                │
│            │                       │                       │                │
│   ┌────────▼────────┐     ┌────────▼────────┐     ┌────────▼────────┐      │
│   │   GDC Server    │     │   GDC Server    │     │   GDC Server    │      │
│   │ (Event Builder) │     │ (Event Builder) │     │ (Event Builder) │      │
│   └────────┬────────┘     └────────┬────────┘     └────────┬────────┘      │
│            │ TCP                   │ TCP                   │ TCP            │
└────────────┼───────────────────────┼───────────────────────┼────────────────┘
             │                       │                       │
┌────────────┼───────────────────────┼───────────────────────┼────────────────┐
│            │         Data Concentration Layer               │                │
│   ┌────────▼────────┐     ┌────────▼────────┐     ┌────────▼────────┐      │
│   │   LDC Server    │     │   LDC Server    │     │   LDC Server    │      │
│   │(Local Concentr.)│     │(Local Concentr.)│     │(Local Concentr.)│      │
│   └────────┬────────┘     └────────┬────────┘     └────────┬────────┘      │
│            │ UDP                  │ UDP                  │ UDP             │
└────────────┼───────────────────────┼───────────────────────┼────────────────┘
             │                       │                       │
┌────────────┼───────────────────────┼───────────────────────┼────────────────┐
│            │           Front-End Electronics                   │             │
│   ┌────────▼────────┐     ┌────────▼────────┐     ┌────────▼────────┐      │
│   │  ATCA Blade     │     │  ATCA Blade     │     │  ATCA Blade     │      │
│   │  (FPGA)         │     │  (FPGA)         │     │  (FPGA)         │      │
│   └────────┬────────┘     └────────┬────────┘     └────────┬────────┘      │
│            │                       │                       │                │
│   ┌────────▼────────┐     ┌────────▼────────┐     ┌────────▼────────┐      │
│   │  PMTs / SiPMs   │     │  PMTs / SiPMs   │     │  PMTs / SiPMs   │      │
│   │  (Sensors)      │     │  (Sensors)      │     │  (Sensors)      │      │
│   └─────────────────┘     └─────────────────┘     └─────────────────┘      │
│                         Detector Planes                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Components

### Core Services

| Component | Directory | Description |
|-----------|-----------|-------------|
| **LDC** (Local Data Concentrator) | `ldcRPC/` | Reads UDP packets from ATCA blades, assembles event fragments, forwards to GDCs via TCP |
| **GDC** (Global Data Concentrator) | `gdcRPC/` | Receives events from multiple LDCs, builds complete events, writes to HDF5 files with Blosc compression |
| **API Server** | `api/` | REST/gRPC API for system control, run management, database operations |
| **Web GUI** | `gui/` | Vue.js 3 interface for detector control and monitoring |

### Supporting Packages

| Package | Description |
|---------|-------------|
| `pkg/` | Shared utilities: configuration, database access, messaging, metrics |
| `database/` | MySQL schema and sqlc-generated query code |
| `rpc/` | Protocol buffer definitions for gRPC services |
| `deviceSimulator/` | Replays real detector data from DATE format files for testing |

## Data Flow

1. **Detection**: Sensors (PMTs/SiPMs) detect S1 (primary scintillation) and S2 (electroluminescence) signals
2. **Digitization**: ATCA FPGA boards sample at 40 MHz with 25 ns resolution
3. **Local Concentration**: LDCs receive UDP frames, buffer in ring buffers, assemble event fragments
4. **Global Building**: GDCs receive TCP streams from LDCs, construct complete events with headers
5. **Storage**: Events written to HDF5 files with ~80% compression (Blosc)
6. **Processing**: Files transferred to computing cluster for reconstruction with [Invisible Cities](https://github.com/next-exp/IC)

## System Internals

### Local Data Concentrator (LDC)

#### UDP Reception with Sequence Tracking

Each LDC receives UDP datagrams from multiple equipment boards (ATCA blades). Every packet contains:

- **Magic number**: For packet validation
- **Sequence counter**: Monotonically increasing counter per equipment
- **Event data**: Payload from the front-end electronics

The LDC tracks the expected sequence number for each equipment. When a received packet's counter doesn't match the expected value, a **missing datagram is detected** and the entire event is marked with an error flag (`0xfafafafa` marker). This corrupted event is skipped during processing, ensuring data integrity.

Packets are buffered in a **ring buffer** that coordinates producer (UDP receiver) and consumer (event assembler) goroutines with reference counting for safe memory reuse.

#### Event Assembly

The LDC assembles complete sub-events by collecting data from all configured equipments belonging to the same event ID. An event is considered complete when all expected equipment fragments have been received.

#### Round-Robin Distribution to GDCs

Assembled sub-events are distributed to GDCs using a **round-robin algorithm**:

```
currentGDC = (currentGDC + 1) % nGDCs
```

This ensures even load distribution across all GDC servers. Each complete event is sent via TCP to the next GDC in the pool. On errors, the counter still advances to prevent a single failing GDC from blocking the system.

### Global Data Concentrator (GDC)

#### Event Building

Each GDC receives sub-events from multiple LDCs via TCP connections. The GDC assembles complete detector events by:

1. Receiving LDC data chunks with magic number validation
2. Tracking event IDs (incremented by `nGDCs` due to round-robin distribution)
3. Detecting missing events when expected IDs don't arrive
4. Assembling full events with proper headers from all LDC fragments

#### Output Formats

The GDC supports two output modes:

**Binary Format (Raw)**
- Writes raw event data to `.rd` files in DATE-compatible binary format
- File naming: `run_{run}.{host}.{experiment}.{subrun}.rd`
- Minimal CPU overhead, maximum throughput
- Suitable for later offline processing

**HDF5 Format (Decoded Waveforms)**
- Runs the [decoder_go](https://github.com/next-exp/decoder_go) library to convert raw data
- Extracts digitized waveforms from PMTs and SiPMs
- Writes to HDF5 files with Blosc compression (~80% size reduction)
- Parallel processing: separate worker pools for Type 1 and Type 2 triggers
- File closing coordinated via `fileCloser` to ensure all events are written

The decoder configuration enables/disables HDF5 writing at runtime without code changes.

## Event Detection

The DAQ implements a dual-trigger scheme:

- **Type 1 (Calibration)**: Continuous 83mKr calibration, tens of Hz rate
- **Type 2 (Physics)**: Physics events in 1-2.5 MeV range, <<1 Hz rate

Both triggers operate in parallel with double-buffering to minimize dead time (11-16% for Type 2). 
The trigger window spans twice the maximum drift time to capture both S1 and S2 signals.

## Technology Stack

### Backend
- **Go 1.24** - Primary language for all services
- **ConnectRPC** - RPC communication between components
- **HDF5 + Blosc** - Data storage with compression
- **MySQL** - Configuration and run metadata
- **Prometheus** - Metrics and monitoring
- **Centrifuge** - Real-time WebSocket messaging

### Frontend
- **Vue.js 3** + **TypeScript**
- **Tailwind CSS** + **DaisyUI**
- **Vite** build tooling
- **Vitest** + **Playwright** testing

### Infrastructure
- **Docker** + **Docker Compose**
- **GitHub Actions** CI/CD
- **Mage** build automation
- **Task** (go-task) for development workflows

## Quick Start

### Prerequisites

- Docker and Docker Compose
- [Task](https://taskfile.dev/) (go-task)
- ~10 GB disk space for base image

### Development Environment

```bash
# Build base Docker image (one-time, ~10 min)
task e2e:build-base

# Start full stack with GUI dev server
task e2e:up:dev
```

Access the services:
- **GUI**: http://localhost:5173 (hot reload enabled)
- **API**: http://localhost:1323
- **Centrifugo**: http://localhost:8000
- **MySQL**: localhost:3306

### NEXT-100 Simulation

Run with real data replay simulating the full NEXT-100 detector:

```bash
# Start NEXT-100 simulation (7 LDCs, 7 GDCs, 28 equipment simulators)
task e2e:up:next100:dev
```

## Testing

```bash
# Unit tests
task test-unit

# Integration tests
task test-integration:all

# All tests
task test

# Frontend tests
task frontend-test

# Coverage report
task test-coverage-html
```

## Development

### Code Generation

```bash
# Generate RPC stubs, SQL queries, mocks
task gen:all

# Individual generators
task gen:rpc      # LDC/GDC control interface
task gen:api      # Web API + TypeScript
task gen:mocks    # Mock implementations
```

### Component-Specific Development

```bash
# GDC development
task gdcrpc:shell           # Interactive shell with HDF5
task gdcrpc:test:all        # All tests
task gdcrpc:test:coverage   # Coverage report

# LDC development
task ldcrpc:shell           # Interactive shell
task ldcrpc:test:all        # All tests
task ldcrpc:test:coverage   # Coverage report
```

### Recompiling During Development

```bash
# Recompile Go services without resetting database
task e2e:recompile
```

## Project Structure

```
.
├── api/                    # API server (Go)
├── database/               # MySQL schema and queries
├── deviceSimulator/        # Data replay from DATE files
├── docker/                 # Docker configurations
│   ├── e2e/               # End-to-end testing setup
│   └── testing/           # Test infrastructure
├── gdcRPC/                 # Global Data Concentrator
├── gui/                    # Vue.js web interface
├── integration_tests/      # Integration test suites
├── ldcRPC/                 # Local Data Concentrator
├── pkg/                    # Shared Go packages
│   ├── config/            # Configuration management
│   ├── database/          # Database access (sqlc)
│   ├── centrifuge/        # Messaging client
│   └── ...
└── rpc/                    # Protocol buffer definitions
    └── proto/
        ├── api/           # Web API definitions
        └── control/       # LDC/GDC control interface
```

## Performance

- **Throughput**: Up to 125 MB/s per server (875 MB/s total)
- **Compression**: ~80% reduction with Blosc
- **Latency**: Sub-second event building
- **Dead Time**: 11-16% for physics triggers during calibration

## References

- [The NEXT-100 Detector](https://arxiv.org/abs/2505.17848) - Detector paper describing DAQ system (Section 6)
- [NEXT Collaboration](https://next.ific.uv.es/next/)
- [Invisible Cities](https://github.com/next-exp/IC) - Event reconstruction software
- [LSC - Laboratorio Subterráneo de Canfranc](https://www.lsc-canfranc.es/)
