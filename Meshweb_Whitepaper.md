# Meshweb: A Peer-to-Peer Protocol for Decentralized Compute and Storage

**Abstract.** The current cloud computing and data storage paradigm relies heavily on centralized trust providers, creating single points of failure, censorship risks, privacy vulnerabilities, and inefficient resource allocation. We propose Meshweb, a fully decentralized, serverless protocol for distributed compute and file storage. Meshweb leverages a peer-to-peer network based on LibP2P and a Kademlia Distributed Hash Table (DHT), completely removing the need for a central authority. Nodes in the network offer idle GPU, CPU, and storage resources in a secure, sandboxed environment, and are algorithmically compensated via MWCoin, a native utility token. Through automated job matching, supply-and-demand based pricing, and erasure-coded AES-256 encrypted storage, Meshweb provides a resilient, autonomous, and self-sustaining ecosystem that grows inexorably more robust as it expands.

---

## 1. Introduction & Core Philosophy
The internet was originally conceived as a decentralized network, yet modern infrastructure has gravitated toward centralized cloud providers. This architecture is vulnerable to outages, censorship, and data breaches. 

Meshweb is engineered under a strict core philosophy:
*   **Absolute Decentralization:** There are zero centralized servers or master nodes. 
*   **Resilience:** The protocol is designed to be unkillable. It propagates and adapts seamlessly across its nodes.
*   **Autonomy:** Meshweb functions entirely without human intervention, maintaining its operations even in the absence of its creators.
*   **Progressive Security:** The network's strength, security, and available capacity scale linearly with time and node adoption.

## 2. The Problem
Today's computational and storage markets face significant bottlenecks:
1.  **Monopolization:** A handful of enterprise providers dictate pricing and access.
2.  **Resource Inefficiency:** Millions of consumer and enterprise devices possess idle GPU, CPU, and storage capacities that remain unutilized.
3.  **Privacy & Trust:** Users are forced to trust third parties with sensitive data and proprietary algorithms.
4.  **Compute Scarcity:** AI startups and independent developers face prohibitive costs and wait times for high-performance compute (HPC) clusters.

## 3. The Meshweb Solution
Meshweb solves these issues by abstracting computing hardware and storage media into a frictionless, global marketplace. It connects those requiring intensive compute (e.g., AI model training, rendering) and secure storage directly with global providers. The protocol handles routing, execution verification, encryption, and financial settlement completely autonomously.

## 4. Technical Architecture
The Meshweb architecture is divided into four interdependent layers.

### 4.1 Layer 1: Protocol (The Transport Layer)
The foundation of Meshweb is a robust, censorship-resistant peer-to-peer network.
*   **P2P Connectivity:** Built on **LibP2P**, ensuring modular and adaptable network transports capable of traversing NATs and firewalls.
*   **Routing:** A **Kademlia DHT** is utilized for node discovery and content routing, entirely eliminating the need for centralized tracking servers or DNS bottlenecks.
*   **Secure Communication:** All internode communications are strictly encrypted using **TLS** and the **Noise Protocol Framework**, guaranteeing privacy and preventing deep packet inspection (DPI) attacks.

### 4.2 Layer 2: Node (The Execution Layer)
Nodes are the physical backbone of the protocol. Any device can join as a node.
*   **Resource Provisioning:** Nodes independently offer GPU, CPU, and Storage resources to the network.
*   **Proof of Work (Verified Execution):** To prevent malicious actors from claiming false computation, Meshweb employs a specialized Proof of Work mechanism. Computations are deterministically verifiable, ensuring the node actually expended the claimed computational effort.
*   **Sandboxed Isolation:** Security for the host node is paramount. All external workloads are executed within strict **Docker** or **WebAssembly (WASM)** sandboxes, preventing malicious code from escaping the execution environment and accessing the host OS.

### 4.3 Layer 3: Market (The Settlement Layer)
This layer acts as the autonomous decentralized exchange for hardware resources.
*   **Automatic Pricing Mechanism:** Prices are dynamically adjusted by an algorithmic market maker. High demand for GPUs in a specific region or globally will automatically increase the MWCoin yield for providing those resources.
*   **Job Matching:** The protocol automatically pairs buyers (computation/storage requesters) with the most optimal nodes based on latency, reputation, hardware specifications, and current pricing.
*   **Smart Contracts:** Agreements are codified in lightweight smart contracts. Once a node proves execution or storage retention, the contract executes automatically, settling the transaction without intermediaries.

### 4.4 Layer 4: Storage (The Persistence Layer)
Meshweb redefines data storage to prioritize absolute privacy and redundancy.
*   **Erasure Coding:** Files are not stored whole on any single machine. They are fragmented using advanced erasure coding algorithms. A file can be entirely reconstructed even if only **30% of its distributed fragments** remain online.
*   **Encryption by Default:** Prior to fragmentation, all data is encrypted client-side using **AES-256**. The network stores fragments of encrypted ciphertext; no node can ever read or comprehend the data they are hosting.

## 5. MWCoin Tokenomics
Meshweb operates on **MWCoin**, a pure utility token designed solely to power the ecosystem.

### 5.1 Token Distribution (The 100M Hard Cap)
Similar to Bitcoin's 21 million limit, MWCoin has a mathematically enforced maximum supply of **100,000,000 MWCoin**. The distribution is structured to prioritize network growth:
*   **Mining & Node Rewards (75% - 75,000,000):** Emitted over time strictly to nodes providing compute and storage.
*   **Protocol Fund (10% - 10,000,000):** Algorithmically locked to fund future infrastructure development and core maintenance.
*   **Initial Liquidity (10% - 10,000,000):** Provided to Decentralized Exchanges (DEXs) to ensure stable market entry.
*   **Genesis Contributors (5% - 5,000,000):** Vested and locked via smart contract for 24 months to ensure long-term alignment.

### 5.2 Node Incentive Mathematics (Earnings)
To provide clarity for node operators and investors, Meshweb utilizes a deterministic yield formula adjusted by a dynamic market multiplier. 
**Base Formula:** `Yield = (Hardware Epoch Contribution) × Network Demand Multiplier`

*Estimated Base Yields (subject to algorithmic adjustment):*
*   **1 Hour High-End GPU (e.g., RTX 4090 / A100):** ~0.5 MWCoin
*   **1 Hour Standard CPU (e.g., 8-Core Ryzen):** ~0.05 MWCoin
*   **1 TB Encrypted Storage (Per Month):** ~2.0 MWCoin

### 5.3 Core Mechanics
*   **Minting via Utility:** MWCoin is minted strictly through productive utility. Nodes that successfully provide compute cycles or verifiable storage over time automatically mint new MWCoin as a block reward.
*   **Autonomous Settlement:** Buyers purchase network resources by spending MWCoin. The protocol escrow and distributes these funds automatically to the participating nodes upon job completion.
*   **Protocol Fund:** To ensure the long-term maintenance and evolution of the underlying infrastructure, **2% to 5%** of all newly minted MWCoins are algorithmically routed to a decentralized Protocol Fund. This parameter is hardcoded and immutable.
*   **Fiat On/Off Ramps:** While MWCoin acts as the lifeblood within Meshweb, participants can freely exchange it via Decentralized Exchanges (DEXs) for fiat or other cryptocurrencies, providing real-world value to node operators.

## 6. Competitive Analysis
While there are existing decentralized infrastructure projects, Meshweb introduces a paradigm shift by solving the limitations of current market leaders:

*   **vs. Akash Network:** Akash focuses primarily on general-purpose cloud computing and operates on an active bidding system. Meshweb is explicitly optimized for hybrid workloads (AI GPU compute + Storage) and uses a fully automated pricing mechanism, significantly reducing friction for buyers and nodes.
*   **vs. io.net:** io.net relies heavily on centralized coordinators and sequencers for cluster management. Meshweb is 100% serverless, relying solely on LibP2P and Kademlia DHT. If a central coordinator goes down, Meshweb continues seamlessly.
*   **vs. Filecoin:** Filecoin is strictly a storage protocol with computationally heavy Proof-of-Spacetime requirements. Meshweb seamlessly unifies both compute and storage into a single lightweight node client that almost any consumer device can run.

## 7. Security & Attack Vectors
Meshweb anticipates and mitigates primary P2P attack vectors:
*   **Sybil Attacks:** Mitigated through computational Proof of Work and a reputation staking system tied to MWCoin.
*   **Data Breaches:** AES-256 client-side encryption ensures data is mathematically inaccessible to host nodes.
*   **Host Compromise:** WASM and Docker sandboxing prevent malicious payloads from affecting node operators.

## 8. Roadmap
Meshweb development is structured in sequential phases targeting progressive decentralization and capability.

*   **Phase 1: Genesis:** Release of the core whitepaper, initial open-source codebase, and command-line node software. Testnet launch for CPU compute and basic storage.
*   **Phase 2: Acceleration:** Integration of GPU virtualization and optimization for AI training workloads. Launch of the MWCoin mainnet.
*   **Phase 3: GUI & Onboarding:** Release of user-friendly desktop clients allowing non-technical users to allocate resources with one click.
*   **Phase 4: Autonomous Ecosystem:** Full decentralization of the governance protocol. The system achieves self-sustaining equilibrium with automated fiat gateways.

## 9. Conclusion
Meshweb represents a fundamental shift away from centralized data centers back to the original ethos of the internet: a distributed, resilient, and permissionless network. By combining Kademlia DHTs, secure sandboxing, algorithmic job matching, and a robust tokenomic model, Meshweb creates a self-perpetuating global supercomputer and storage drive. It is a system that belongs to no one, operates for everyone, and grows more indestructible with every node that joins.
