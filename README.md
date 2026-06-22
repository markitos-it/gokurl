# gOKurl — gRPC Artisan Client

**gOKurl** is an elegant, microservice-focused desktop GUI client designed to parse Protobuf files natively and trigger execution payloads over gRPC targets. Engineered following clean desktop patterns, it replaces terminal command hacking with a high-fidelity, artisan-tailored developer cockpit.

---

## Key Features

* 🛠️ **Native Protobuf Auto-Parsing:** Drop any `.proto` file; the app maps out all services, RPC endpoints, requests, and nested message fields instantly.
* 🗂️ **Smart Asset Synchronization:** Automatically tracks schemas in a local `assets/` context directory. External schemas opened from other workspaces are cached dynamically for rapid persistence next time you boot the tool.
* 📐 **Bento-Inspired Visual Hierarchy:** Segregated layout grouping endpoints on the left, input generation forms in the center, and standalone stream panels on the bottom.
* 🚦 **Isolated Log Telemetry:** Dedicated, high-contrast twin consoles separating the outgoing **Client Request Log** from the inbound **Server Response Log**.
* 🛡️ **Real-Time Input Validation:** Structural field assertions and visual safeguards that lock down execution pathways until valid endpoints are present.

---

## Architectural Breakdown

```
 _____________________________________________________________________
|  MÉTODOS DETECTADOS        |  PANEL DE CONFIGURACIÓN                |
|  - service.Method1         |  [ Host: localhost:50051            ]  |
|  - service.Method2         |                                        |
|____________________________|  Parámetros del Request:               |
|  ASSETS DISPONIBLES        |  [ Field A: Type                     ]  |
|  - api.proto               |  [ Button: Enviar Request 🚀         ]  |
|  - health.proto            |________________________________________|
|                            |  Client Request Log (Monospace Slate)  |
|                            |  Server Response Log (Monospace Slate) |
|____________________________|________________________________________|

```

---

## Getting Started

### Prerequisites

Your host workstation must have `grpcurl` available in its system PATH to bridge network payloads:

```bash
# Mac (Homebrew)
brew install grpcurl

# Linux (Ubuntu/Debian)
sudo apt install grpcurl

```

### Installation & Run

1. Clone the repository and navigate into the workspace directory:
```bash

```



git clone https://github.com/your-username/gokurl.git
cd gokurl

```
2. Build and run the binary natively using the standard Go toolchain:
   ```bash
go run main.go

```

---

## Cross-Platform Engineering

This project utilizes a structured `Makefile` and `fyne-cross` within an isolated containerized infrastructure to guarantee artifact consistency across target families.

### Compiling Natively via Docker Toolchains

```bash
# Compile and output development artifacts for standard distributions
make build-all

# Build clean production distribution bundles (Linux, Windows GUI, macOS .app)
make release-all

```

> ⚠️ **Note:** To ensure correct rendering of taskbar elements during distribution compiling, make sure an `Icon.png` file exists in the root workspace directory before calling the production release routines.

---

## License

This project is licensed under the **Apache License, Version 2.0**. You may freely use, modify, and distribute this software under the terms outlined in the license file.

```
Copyright 2026 Markitos

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

```

---

---

## Deep Dive: Open Source Licensing Integrity

Choosing the **Apache License 2.0** brings essential compliance characteristics to a desktop tool compiling cross-platform binaries:

### Structural Patent Protection

Unlike more permissive licenses like MIT, Apache 2.0 includes an explicit grant of patent rights from contributors to users. If anyone contributes code to your project that relies on a patent they own, they automatically grant you and your downstream users a royalty-free license to use it. Furthermore, it contains a defensive **patent retaliation clause**: if a user brings a patent lawsuit against you claiming your app infringes their IP, their license to this software terminates immediately.

### Modification Transparency

Section 4(b) explicitly requires that if you modify any files under this codebase, you must carry prominent notices indicating that the files have been altered. This guarantees downstream provenance tracking, protecting your original architectural footprint while allowing forks to innovate in the clear.

### Trademark Preservation

The Apache 2.0 license grants permissions for source code and compiled binaries but strictly excludes trade names and trademarks. This ensures that while developers can fork and adapt your runtime engine under compliance, they cannot use the identity or name of the tool to masquerade commercial ecosystem solutions without written permission.