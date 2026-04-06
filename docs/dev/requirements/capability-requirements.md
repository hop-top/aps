# Capability Management Requirements

## Functional Requirements

### 1. Installation & Management
*   **REQ-CAP-001**: The system MUST allow installing a capability from a local directory source.
*   **REQ-CAP-002**: The system MUST allow adopting an existing file/directory into APS management (move and link back).
*   **REQ-CAP-003**: The system MUST allow watching an external file/directory (reference without moving).
*   **REQ-CAP-004**: The system MUST allow deleting a capability, removing artifacts and references based on type.

### 2. Linking & Integration
*   **REQ-CAP-005**: The system MUST allow linking a capability into the current workspace.
*   **REQ-CAP-006**: The system MUST implement "Smart Linking" to automatically resolve target paths for known tools (e.g., Copilot, Windsurf).
*   **REQ-CAP-007**: The system MUST support multiple capability source directories configured via `config.yaml`.

### 3. Environment
*   **REQ-CAP-008**: The system MUST provide a command (`aps env`) to generate shell export statements for all capabilities.
*   **REQ-CAP-009**: Environment variables MUST follow the naming convention `APS_<NAME>_PATH`.
*   **REQ-CAP-010**: The environment generation MUST dynamically reflect the current state of installed capabilities.

### 4. CLI Commands
The following commands MUST be implemented:
*   `aps capability install <source> --name <name>`
*   `aps capability link <name> [--target <path>]`
*   `aps capability adopt <path> --name <name>`
*   `aps capability watch <path> --name <name>`
*   `aps capability delete <name>`
*   `aps capability list`
*   `aps env`

## Non-Functional Requirements

*   **NFR-CAP-001**: Operations MUST be atomic where possible (e.g., install either fully succeeds or fails cleanly).
*   **NFR-CAP-002**: Environment variable generation MUST be fast (< 50ms) to avoid shell startup delay.
*   **NFR-CAP-003**: Smart Link patterns MUST be extensible (defined in code or config).
