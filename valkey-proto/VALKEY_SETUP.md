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

One of the most critical parts of your config is the **"Announce"** flags. These flags solve the **"NAT/Docker Bridge Problem"** where Valkey nodes have two identities: one inside the Docker network (e.g., `172.18.0.2`) and one on your host Mac (e.g., `localhost:6380`).

#### The Problem: Internal vs. External IPs
In a cluster, nodes perform a **"Handshake"** (via the Gossip protocol) to share their location and the hash slots they own. By default, Valkey uses its **local IP**.

If you connect your Go app from your Mac to `valkey0`, and the key you want is on `valkey1`:
1. `valkey0` sees the request.
2. `valkey0` knows the key belongs to `valkey1`.
3. **Without Announce Flags**: `valkey0` sends a `MOVED 172.18.0.3:6379` error.
4. Your Go app tries to connect to `172.18.0.3`—but this fails because that IP only exists inside Docker!

#### The Solution: The Announce Flags
These flags override the "Auto-detected" IP/Port with an identity that works for your Mac.

| Flag | Role |
| :--- | :--- |
| `--cluster-announce-ip` | **The Public Address**: This tells the cluster (and clients) to use `Shubhams-MacBook-Pro.local` instead of the internal container IP (`172.18.x.x`). |
| `--cluster-announce-port` | **The Client Entry Point**: This tells clients which **host port** (mapped in Docker Compose) leads to this specific node (e.g., `6380` instead of the internal `6379`). |
| `--cluster-announce-bus-port` | **The Gossip Port**: A secondary port (default: Data Port + 10000) used for node-to-node heartbeats, failover votes, and slot management. Both nodes and clients need access to this port to monitor cluster health. |

**Key Takeaway**: Every node in your `docker-compose.yml` has a unique `cluster-announce-port` (6379, 6380, 6381, 6382) so that no matter which node your client talks to, the redirection it receives will point to a valid, reachable port on your host machine.

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

---

## 6. Scaling & Advanced Replicas

### The Minimum Master Requirement
Valkey Cluster requires at least **3 Master nodes** for a production-ready quorum. This allows the cluster to survive failures and vote on which slave should become a master.

### Using `--cluster-replicas 2`
Setting `--cluster-replicas 2` means every master must have two slaves. 
- With the minimum 3 masters, you would need **9 nodes total** (3 masters + 6 slaves).
- If you try this with only 4 or 6 nodes, the cluster creation will fail.

### Handling Uneven Nodes (e.g., 5 Nodes)
Automated cluster creation works best with even ratios. If you have 5 nodes and want to assign the 5th node to a specific master:

1. **Phase 1**: Create a 4-node cluster (2 master, 2 slave) normally.
2. **Phase 2**: Find the Master ID of the node you want to give an extra slave to:
   ```bash
   valkey-cli cluster nodes
   ```
3. **Phase 3**: Manually add the 5th node as a slave to that ID:
   ```bash
   valkey-cli --cluster add-node <new_node_ip>:6379 <existing_node_ip>:6379 \
     --cluster-slave \
     --cluster-master-id <master_node_id>
   ```

Over-provisioning a master with extra slaves is useful for **high read-scaling** or extra **failover safety** for critical data shards.
