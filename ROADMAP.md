# Roadmap

## Near-Term Enhancements

- **GPU / Driver Diagnostics**
  - Add `dgx diagnostics` to run `nvidia-smi`, NVSwitch/NVLink health checks, and validate driver ↔ CUDA ↔ container runtime alignment.

- **Resource Tuning Helpers**
  - Provide `dgx tune limits` to inspect/update `ulimit`, `/dev/shm`, `vm.max_map_count`, and other recommended settings for heavy AI workloads.

- **Job Orchestration Integrations**
  - Slurm wrappers for job submission/log tailing, and Kubernetes port-forward helpers tailored to DGX Spark + NIM/vLLM deployments.

- **Support Bundle Generator**
  - `dgx support bundle` to gather logs, configs, `dgx status`, and Docker diagnostics into a scrubbed archive for NVIDIA support.

- **Playbook Documentation Automation**
  - Auto-generate Markdown/CLI help for each playbook so docs stay in sync as workflows evolve.

- **Firmware / Driver Awareness**
  - `dgx firmware check` to surface recommended BIOS/firmware/driver versions and warn when the system drifts.

- **Sandbox Containers**
  - `dgx sandbox` to launch a preconfigured development container (GPU access, env vars, storage mounts) for safe experimentation.

## Longer-Term Ideas

- Device-code style authentication flows if upstream CLIs (e.g., Codex) add support.
- Telemetry opt-in to anonymously surface performance bottlenecks and common failure modes.

