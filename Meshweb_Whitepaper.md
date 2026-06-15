# Meshweb: A Peer-to-Peer Protocol for Decentralized Storage and Compute

**Version 0.1.0 — June 2026**

---

**Abstract.** Modern cloud infrastructure concentrates data and compute in the hands of a few corporations, creating single points of failure, censorship vectors, and privacy risks. We present Meshweb, a fully decentralized peer-to-peer protocol for distributed file storage. Built on LibP2P and Kademlia DHT, Meshweb requires zero central servers. Files are encrypted client-side with AES-256-GCM, split into 30 shards using Reed-Solomon erasure coding (10 data + 20 parity), and distributed across the network. Any 10 of 30 shards can reconstruct the original file, providing extreme fault tolerance. Node identity is self-sovereign, derived from a BIP39 mnemonic seed phrase and Ed25519 keypair. The protocol operates autonomously — no registration, no accounts, no central authority. Future phases will introduce a compute marketplace powered by MWCoin, a native utility token.

---

## Table of Contents

1. [Introduction & Core Philosophy](#1-introduction--core-philosophy)
2. [The Problem](#2-the-problem)
3. [The Meshweb Solution](#3-the-meshweb-solution)
4. [Technical Architecture](#4-technical-architecture)
5. [Storage Protocol (Implemented)](#5-storage-protocol-implemented)
6. [Identity System](#6-identity-system)
7. [Sharing & Content Addressing](#7-sharing--content-addressing)
8. [Compute Market (Planned)](#8-compute-market-planned)
9. [MWCoin Tokenomics (Planned)](#9-mwcoin-tokenomics-planned)
10. [Security & Threat Model](#10-security--threat-model)
11. [Competitive Analysis](#11-competitive-analysis)
12. [Roadmap](#12-roadmap)
13. [Conclusion](#13-conclusion)

---

## 1. Introduction & Core Philosophy

The internet was originally conceived as a decentralized network, yet modern infrastructure has gravitated toward centralized cloud providers. This architecture is vulnerable to outages, censorship, and data breaches.

Meshweb is engineered under a strict core philosophy:

- **Absolute Decentralization.** There are zero centralized servers. The only infrastructure dependency is a public relay node for NAT traversal, which can be replaced or multiplied by any participant.
- **Resilience.** The protocol is designed to be unkillable. Data survives even when 66% of hosting nodes go offline simultaneously.
- **Autonomy.** Meshweb functions entirely without human intervention, maintaining its operations even in the absence of its creators.
- **Progressive Strength.** The network's capacity, redundancy, and routing efficiency scale with every new node that joins.
- **Privacy by Default.** All data is encrypted before it ever leaves the user's device. No node — including the node storing data — can read it.

---

## 2. The Problem

Today's data storage and computational markets face significant bottlenecks:

1. **Monopolization.** A handful of providers (AWS, Google Cloud, Azure) dictate pricing, terms, and access.
2. **Censorship.** Centralized providers can unilaterally remove content, freeze accounts, or comply with takedown requests.
3. **Privacy.** Users are forced to trust third parties with unencrypted data and metadata.
4. **Single Point of Failure.** A regional outage at one provider can take millions of services offline.
5. **Resource Inefficiency.** Billions of consumer devices possess idle storage and compute capacity that remains unutilized.

---

## 3. The Meshweb Solution

Meshweb transforms every device into a node in a global, permissionless storage network. The protocol handles encryption, fragmentation, distribution, discovery, and reconstruction — completely autonomously.

```
┌──────────────────────────────────────────────────────────┐
│                    MESHWEB PROTOCOL                      │
│                                                          │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐ │
│  │  User    │   │ AES-256 │   │  Reed-  │   │   P2P   │ │
│  │  File    │──▶│   GCM   │──▶│ Solomon │──▶│  Swarm  │ │
│  │         │   │ Encrypt │   │ 10 + 20 │   │  (DHT)  │ │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘ │
│                                                          │
│  Upload: File → Encrypt → Split → Distribute            │
│  Download: Discover → Fetch 10/30 → Reconstruct → Decrypt│
└──────────────────────────────────────────────────────────┘
```

---

## 4. Technical Architecture

The Meshweb architecture is divided into distinct layers, each serving a specific function.

### 4.1 Layer 1: Transport (LibP2P)

The foundation of Meshweb is a robust, censorship-resistant peer-to-peer network built on **LibP2P**.

| Component | Technology | Purpose |
|---|---|---|
| TCP Transport | LibP2P TCP | Primary connectivity |
| WebSocket | LibP2P WS | Browser-compatible transport |
| QUIC | LibP2P QUIC-v1 | Low-latency UDP transport |
| NAT Traversal | AutoRelay + HolePunching | Connectivity behind NATs |
| Relay | Circuit Relay v2 | Fallback for symmetric NATs |
| Encryption | TLS + Noise Protocol | All internode traffic encrypted |

**NAT Traversal Strategy:**
1. Node attempts direct connection via TCP/QUIC.
2. If blocked, AutoRelay activates with static relay servers.
3. HolePunching is attempted for direct peer-to-peer path.
4. If all else fails, Circuit Relay v2 provides guaranteed connectivity.

The current relay server (`/ip4/185.177.116.13/tcp/443/p2p/12D3KooW...`) is a bootstrap facilitator, not a single point of failure — any node can run a relay.

### 4.2 Layer 2: Discovery (Kademlia DHT)

Node discovery and content routing use a **Kademlia Distributed Hash Table**.

- **Peer Discovery:** Nodes advertise themselves under the `meshweb-network` namespace via DHT routing discovery.
- **Resource Announcements:** Nodes broadcast CPU and RAM availability via GossipSub on the `meshweb-nodes` topic every 5 seconds.
- **Content Routing:** File shards are announced and discovered through the DHT, enabling any node to locate and retrieve data without centralized indexing.
- **Bootstrap Sweeper:** A background process runs every 30 seconds, testing bootstrap peers for liveness and pruning unreachable nodes from the local bootstrap list.

### 4.3 Layer 3: Messaging (GossipSub)

Meshweb uses **GossipSub** (a LibP2P pubsub protocol) for real-time network communication.

| Topic | Purpose |
|---|---|
| `meshweb-nodes` | Resource announcements (CPU, RAM) |
| `meshweb-jobs` | Compute job broadcasting (planned) |
| `meshweb-results` | Job result reporting (planned) |

### 4.4 Layer 4: Storage (Implemented)

Detailed in [Section 5](#5-storage-protocol-implemented).

### 4.5 Layer 5: Compute Market (Planned)

Detailed in [Section 8](#8-compute-market-planned).

---

## 5. Storage Protocol (Implemented)

The storage layer is fully implemented and operational. It provides encrypted, redundant, decentralized file storage.

### 5.1 Upload Pipeline

```
Original File (N bytes)
        │
        ▼
┌───────────────────┐
│ AES-256-GCM       │  Random 256-bit key generated
│ Client-Side       │  12-byte nonce prepended to ciphertext
│ Encryption        │  Output: nonce || ciphertext || auth tag
└───────┬───────────┘
        │
        ▼ ciphertext (N + 28 bytes)
┌───────────────────┐
│ SHA-256 Hash       │  Content-addressed via CID (IPFS-compatible)
│ CID Generation     │  Multihash: SHA2-256, Codec: Raw
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ Reed-Solomon       │  10 data shards + 20 parity shards = 30 total
│ Erasure Coding     │  Any 10 of 30 shards → full reconstruction
│ (10, 20)           │  Tolerance: 66% shard loss
└───────┬───────────┘
        │
        ▼ 30 shards
┌───────────────────┐
│ Local Storage      │  Each shard saved as: storage/{CID}/shard_{0..29}
│ + DHT Announce     │  CID announced to DHT for discovery
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ .meshweb Metadata  │  JSON file with: version, filename, CID,
│ File Generation    │  AES key, shard count, original size, creator ID
└───────────────────┘
```

### 5.2 Download Pipeline

```
meshweb:// link or .meshweb file
        │
        ▼
┌───────────────────┐
│ Parse Metadata     │  Extract: CID, AES key, filename, original size
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ DHT + PubSub       │  Find peers hosting the file's shards
│ Provider Discovery  │  Protocol: /meshweb/storage/1.0.0
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ Shard Retrieval     │  Request shards via LibP2P streams
│ (need 10 of 30)    │  JSON-line protocol: ChunkRequest → ChunkResponse
│                     │  Data transmitted as Base64-encoded bytes
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ Reed-Solomon       │  Reconstruct full ciphertext from any 10 shards
│ Reconstruction     │  Trim to OriginalSize (remove RS padding)
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ AES-256-GCM       │  Decrypt using key from metadata
│ Decryption         │  Verify authentication tag (integrity check)
└───────┬───────────┘
        │
        ▼
   Original File
```

### 5.3 Stream Protocol Specification

**Protocol ID:** `/meshweb/storage/1.0.0`

**Request (JSON-line):**
```json
{
  "file_id": "bafkreie...",
  "shard": 0
}
```

**Response (JSON-line):**
```json
{
  "file_id": "bafkreie...",
  "shard": 0,
  "data": "<base64-encoded shard bytes>",
  "error": ""
}
```

### 5.4 Reed-Solomon Parameters

| Parameter | Value | Rationale |
|---|---|---|
| Data Shards | 10 | Minimum fragments needed for reconstruction |
| Parity Shards | 20 | Redundancy fragments |
| Total Shards | 30 | Total distributed fragments |
| Fault Tolerance | 66.7% | Up to 20 of 30 shards can be lost |
| Storage Overhead | 3x | Each file consumes 3x its original size across the network |

### 5.5 Encryption Specification

| Parameter | Value |
|---|---|
| Algorithm | AES-256-GCM (Galois/Counter Mode) |
| Key Size | 256 bits (32 bytes), cryptographically random |
| Nonce Size | 96 bits (12 bytes), cryptographically random |
| Authentication | Built-in GCM auth tag (128 bits) |
| Key Derivation | None — raw random key per file |
| Key Storage | Embedded in `.meshweb` metadata and `meshweb://` links |

### 5.6 Content Addressing

Files are content-addressed using **CID v1** (IPFS-compatible):

- **Hash Function:** SHA2-256
- **Codec:** Raw (0x55)
- **Multihash Format:** Standard multihash encoding
- **Example:** `bafkreie7ohyl7zg6g5wxhvzah5kkgbq...`

This makes Meshweb storage compatible with the broader IPFS content-addressing ecosystem.

---

## 6. Identity System

Meshweb implements a **self-sovereign identity** system with zero registration or central authority.

### 6.1 Key Generation

```
BIP39 Entropy (128 bits)
        │
        ▼
┌───────────────────┐
│ 12-Word Mnemonic   │  Standard BIP39 word list
│ Seed Phrase        │  Human-readable backup
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ BIP39 Seed         │  512-bit deterministic seed
│ Derivation         │  PBKDF2 with empty passphrase
└───────┬───────────┘
        │
        ▼ first 32 bytes
┌───────────────────┐
│ Ed25519 Keypair    │  Deterministic from seed
│ Generation         │  Private key + Public key
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ LibP2P Peer ID     │  Derived from public key
│                    │  Format: 12D3KooW...
└───────────────────┘
```

### 6.2 Properties

| Property | Implementation |
|---|---|
| Mnemonic | BIP39, 12 words, 128-bit entropy |
| Key Algorithm | Ed25519 |
| Peer ID | LibP2P peer.IDFromPrivateKey |
| Storage | Encrypted JSON at `%APPDATA%/meshweb-gui/identity.key` |
| Backup | Export seed phrase or identity.json |
| Recovery | Full identity restoration from 12 words |
| Portability | Same seed phrase → same Peer ID on any device |

### 6.3 Security

- Private keys are stored locally with restrictive file permissions (0600).
- The seed phrase is the master secret — losing it means losing the identity permanently.
- No central registry exists. Identity ownership is proven cryptographically.

---

## 7. Sharing & Content Addressing

Meshweb provides two mechanisms for sharing files:

### 7.1 Meshweb Links

```
meshweb://file/{CID}?k={AES_KEY_HEX}&n={FILENAME_BASE64}&s={ORIGINAL_SIZE}
```

| Parameter | Description |
|---|---|
| `CID` | Content identifier (SHA-256 based) |
| `k` | AES-256 decryption key (hex-encoded) |
| `n` | Original filename (Base64 URL-encoded) |
| `s` | Original ciphertext size (for RS padding removal) |

These links are self-contained: anyone with the link can download and decrypt the file without any account or registration.

### 7.2 .meshweb Files

A `.meshweb` file is a JSON metadata file containing:

```json
{
  "version": "1.0",
  "file_name": "document.pdf",
  "file_size": 1048576,
  "original_size": 1048604,
  "file_id": "bafkreie...",
  "shards": 30,
  "min_shards": 10,
  "encryption": "AES-256-GCM",
  "key_hash": "a1b2c3...",
  "aes_key": "deadbeef...",
  "created_at": "2026-06-15T10:30:00Z",
  "creator_id": "12D3KooW..."
}
```

On Windows, `.meshweb` files can be registered as a file association, allowing double-click to open directly in the Meshweb GUI.

---

## 8. Compute Market (Planned)

The compute marketplace is designed but not yet implemented. It will enable nodes to rent GPU, CPU, and RAM resources to other participants.

### 8.1 Planned Architecture

- **Rental Protocol:** `/meshweb/rent/1.0.0` (stream-based negotiation)
- **Resource Discovery:** GossipSub announcements on `meshweb-nodes`
- **Job Lifecycle:** Request → Accept/Reject → Execute → Settle
- **Sandboxing:** Docker and/or WebAssembly isolation for workload execution
- **Pricing:** Algorithmic supply-and-demand based pricing

### 8.2 Current Status

The protocol scaffolding exists in code:
- `RentalJob` and `RentRequest` data structures are defined
- Stream handler for `/meshweb/rent/1.0.0` is implemented
- Peer-to-peer negotiation flow (request → response) works
- Billing loop infrastructure is in place (currently disabled)

Full implementation awaits MWCoin integration for payment settlement.

---

## 9. MWCoin Tokenomics (Planned)

Meshweb will operate on **MWCoin**, a utility token designed to power the compute and storage marketplace.

### 9.1 Token Distribution (100M Hard Cap)

| Allocation | Percentage | Amount | Purpose |
|---|---|---|---|
| Mining & Node Rewards | 75% | 75,000,000 | Emitted to nodes providing compute/storage |
| Protocol Fund | 10% | 10,000,000 | Infrastructure development and maintenance |
| Initial Liquidity | 10% | 10,000,000 | DEX market making |
| Genesis Contributors | 5% | 5,000,000 | Vested 24 months, smart contract locked |

### 9.2 Estimated Yields

| Resource | Duration | Estimated Yield |
|---|---|---|
| High-End GPU (RTX 4090 / A100) | 1 Hour | ~0.5 MWCoin |
| Standard CPU (8-Core) | 1 Hour | ~0.05 MWCoin |
| 1 TB Encrypted Storage | 1 Month | ~2.0 MWCoin |

*Yields are subject to algorithmic adjustment based on network supply and demand.*

### 9.3 Settlement Mechanics

- **Minting:** MWCoin is minted strictly through productive utility (storage provision, compute execution).
- **Escrow:** Smart contracts hold buyer funds during job execution.
- **Protocol Tax:** 2-5% of minted coins route to the Protocol Fund (hardcoded, immutable).
- **Exchange:** MWCoin will be tradeable on DEXs for fiat or other cryptocurrencies.

---

## 10. Security & Threat Model

### 10.1 Data Security

| Threat | Mitigation |
|---|---|
| Data breach at storage node | AES-256-GCM encryption — nodes store ciphertext fragments only |
| Key interception | Keys embedded in links/files shared out-of-band by the user |
| Data corruption | GCM authentication tag detects any tampering |
| Mass node failure | Reed-Solomon tolerates 66.7% shard loss |

### 10.2 Network Security

| Threat | Mitigation |
|---|---|
| Sybil Attack | Resource-based reputation (future: MWCoin staking) |
| Eclipse Attack | Multiple bootstrap peers, DHT-based diverse routing |
| Man-in-the-Middle | All LibP2P connections use TLS/Noise encryption |
| DPI / Censorship | QUIC transport, WebSocket support, relay fallback |
| Relay Compromise | Relay sees encrypted traffic only; cannot decrypt data or shards |

### 10.3 Identity Security

| Threat | Mitigation |
|---|---|
| Identity theft | Ed25519 private key stored locally with 0600 permissions |
| Key loss | BIP39 seed phrase enables full recovery on any device |
| Impersonation | Peer ID is cryptographically bound to Ed25519 public key |

### 10.4 Known Limitations (v0.1.0)

- **No shard replication protocol.** Currently, shards are stored only on the uploader's node. Multi-node distribution requires the uploader to remain online.
- **No incentive for storage.** Nodes store their own files but lack economic incentive to store others' data (awaiting MWCoin).
- **Single relay dependency.** While architecturally any node can relay, the current deployment uses one relay server.
- **No data persistence guarantees.** If the uploading node goes offline permanently and no other node has the shards, the file is lost.

---

## 11. Competitive Analysis

| Feature | Meshweb | Filecoin | IPFS | Akash | io.net |
|---|---|---|---|---|---|
| Decentralized Storage | ✅ | ✅ | ✅ | ❌ | ❌ |
| Decentralized Compute | 🔜 Planned | ❌ | ❌ | ✅ | ✅ |
| Client-Side Encryption | ✅ Default | ❌ Optional | ❌ | ❌ | ❌ |
| Erasure Coding | ✅ RS(10,20) | ✅ | ❌ | ❌ | ❌ |
| Zero Registration | ✅ | ❌ | ✅ | ❌ | ❌ |
| Self-Sovereign Identity | ✅ BIP39 | ❌ | ❌ | ❌ | ❌ |
| Desktop GUI | ✅ | ❌ | ✅ | ❌ | ❌ |
| Lightweight Client | ✅ ~30MB | ❌ Heavy | ❌ Heavy | ❌ | ❌ |
| Central Coordinator | ❌ None | Partial | ❌ None | Partial | ✅ Required |

**Key Differentiators:**
- **vs. Filecoin:** Filecoin requires computationally expensive Proof-of-Spacetime. Meshweb is lightweight enough for any consumer device.
- **vs. IPFS:** IPFS provides content addressing but no encryption, no erasure coding, and no compute market.
- **vs. Akash/io.net:** These focus on compute but rely on centralized coordinators. Meshweb is 100% serverless.

---

## 12. Roadmap

### Phase 1: Genesis ✅ (Current — v0.1.0)
- [x] Core P2P protocol (LibP2P + Kademlia DHT)
- [x] Encrypted file storage (AES-256-GCM + Reed-Solomon)
- [x] Self-sovereign identity (BIP39 + Ed25519)
- [x] Desktop GUI for Windows (Wails + React)
- [x] Multi-language interface (English, Uzbek, Russian)
- [x] `meshweb://` link sharing and `.meshweb` file association
- [x] Open-source release on GitHub

### Phase 2: Resilience (v0.2.0)
- [ ] Multi-node shard distribution (store shards across network peers)
- [ ] Shard replication protocol (automatic re-replication on node departure)
- [ ] Mac and Linux desktop builds
- [ ] File pinning and persistence guarantees
- [ ] Multiple relay servers

### Phase 3: Economy (v0.3.0)
- [ ] MWCoin mainnet launch
- [ ] Compute marketplace activation
- [ ] Storage incentive system (earn MWCoin by hosting shards)
- [ ] Smart contract-based settlement
- [ ] DEX liquidity provision

### Phase 4: Scale (v1.0.0)
- [ ] Mobile applications (Android, iOS)
- [ ] GPU compute orchestration (AI/ML workloads)
- [ ] Decentralized governance
- [ ] Automated fiat gateways
- [ ] SDK and API for developers

---

## 13. Conclusion

Meshweb v0.1.0 delivers a working, production-ready decentralized file storage protocol. Files are encrypted, fragmented, and content-addressed — no central server ever touches user data. Identity is self-sovereign, derived from a simple 12-word seed phrase.

This is the Genesis release. The foundation is laid. What follows — multi-node distribution, economic incentives, and compute markets — will transform Meshweb from a storage protocol into a global, permissionless infrastructure layer.

The network belongs to no one. It works for everyone. And it grows stronger with every node that joins.

---

*Meshweb is open-source software released under the MIT License.*
*Repository: [github.com/meshweb-protocol/meshweb](https://github.com/meshweb-protocol/meshweb)*
