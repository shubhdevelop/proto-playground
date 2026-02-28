# Valkey Cluster Setup & Networking Guide

This document explains the architecture and configuration of the Valkey cluster defined in your `docker-compose.yml`.

## 1. Networking Architecture

### Internal vs. External Communication
The setup uses a hybrid approach to Ensure that the cluster works both **inside** the Docker network and **outside** (from your Mac terminal).

- **Internal Network (`valkey-net`)**: A standard Docker bridge network. Containers can see each other via their service names (`valkey0`, `valkey1`, etc.).
- **Host Resolution**:
    - `extra_hosts` is used to map `Shubhams-MacBook-Pro.local` to the Docker host's gateway IP (`host-gateway`).
    - This allows containers to resolve your Mac's hostname even when they are inside their own network namespace.

### Cluster Announcement Logic
One of the most critical parts of your config is the "Announce" flags:
```yaml
--cluster-announce-ip Shubhams-MacBook-Pro.local
--cluster-announce-port "6380"
--cluster-announce-bus-port "16380"
```
**Why this is needed:**
In a cluster, nodes tell each other (and clients) where they are located. By default, they would announce their internal Docker IP (e.g., `172.18.0.2`). However, your Mac terminal cannot reach that internal IP. 
By announcing `Shubhams-MacBook-Pro.local` and the **mapped host ports** (6380, 6381, etc.), the cluster ensures that when a client (like your Go app or CLI) gets a `MOVED` error, the redirection address provided is one that the client can actually reach from the outside.

---

## 2. Valkey Configuration Flags

The `command` section for each service configures Valkey for Cluster Mode:

| Flag | Description |
| :--- | :--- |
| `--cluster-enabled yes` | Enables the cluster features in the Valkey engine. |
| `--cluster-config-file ...` | Path to the auto-generated file where the node stores its cluster state. |
| `--cluster-node-timeout 5000` | Max time (ms) a node can be unreachable before it's considered failing. |
| `--appendonly yes` | Enables AOF (Append Only File) persistence for data safety. |
| `--protected-mode no` | Allows connections from outside the loopback interface (required for Docker). |
| `--bind 0.0.0.0` | Listens on all network interfaces. |

---

## 3. Cluster Formation (`cluster-init`)

The `cluster-init` service is a "one-shot" container that creates the cluster topology once all nodes are up.

**The Command Executed:**
```bash
valkey-cli --cluster create \
  Shubhams-MacBook-Pro.local:6379 \
  Shubhams-MacBook-Pro.local:6380 \
  Shubhams-MacBook-Pro.local:6381 \
  Shubhams-MacBook-Pro.local:6382 \
  --cluster-replicas 1
```
- **Replicas**: `--cluster-replicas 1` means for every 1 Master node, create 1 Slave node.
- **Topology**: With 4 nodes total, this results in **2 Masters** and **2 Slaves**.

---

## 4. How to Interact with the Cluster

### From your Mac Terminal
Always use the `-c` flag for automatic redirection:
```bash
valkey-cli -c -h Shubhams-MacBook-Pro.local -p 6379
```

### From inside Docker (Exec)
```bash
docker exec -it valkey0 valkey-cli -c
```

### Reading from Slaves
By default, slaves will redirect you to the master. To read directly from a slave:
1. Connect to a slave port (e.g., `6381` or `6382`).
2. Run the command `READONLY`.
3. Then run your `GET` command.

---

## 5. Port Mapping Summary

| Service | Host Data Port | Host Bus Port | Role |
| :--- | :--- | :--- | :--- |
| `valkey0` | 6379 | 16379 | Master |
| `valkey1` | 6380 | 16380 | Master |
| `valkey2` | 6381 | 16381 | Slave (of `valkey1`) |
| `valkey3` | 6382 | 16382 | Slave (of `valkey0`) |

*Note: The **Bus Port** (Data Port + 10000) is used by nodes to talk to each other about the cluster state (heartbeats, failover votes, etc.).*
