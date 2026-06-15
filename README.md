# Meshweb

<p align="center">
  <img src="assets/logo.png" width="120"/>
</p>

<p align="center">
  <b>Decentralized P2P Cloud Storage & Compute Network</b>
</p>

<p align="center">
  A serverless, censorship-resistant network where
  every device is a node. No central authority.
  No single point of failure.
</p>

---

## What is Meshweb?

Meshweb is a fully decentralized peer-to-peer
protocol for distributed file storage.
Built on LibP2P and Kademlia DHT,
it requires zero central servers.

The network becomes stronger and more
resilient with every new node that joins.
It belongs to no one. It works for everyone.

---

## Features

- 🔒 **End-to-End Encryption** — AES-256-GCM
  client-side encryption before any data
  leaves your device
- 🧩 **Erasure Coding** — Reed-Solomon (10+20)
  means any 10 of 30 shards can
  reconstruct your file
- 🌐 **True P2P** — LibP2P + Kademlia DHT.
  No trackers. No central servers.
- 🔗 **Easy Sharing** — Share files via
  meshweb:// links or .meshweb files,
  just like magnet links
- 🆔 **Self-Sovereign Identity** — BIP39
  seed phrase + Ed25519 keypair.
  You own your identity.
- 💻 **Cross-Platform GUI** — Built with
  Wails + React. Clean, minimal interface.
- 🌍 **Multi-language** — English, Uzbek,
  Russian

---

## How It Works

**Upload:**  
File → AES-256 encrypt → Reed-Solomon split (30 shards) → DHT announce

**Download:**  
Find provider via DHT → Fetch any 10 shards → Reed-Solomon reconstruct → AES decrypt → Original file

---

## Installation

### Download
Download the latest release for Windows:
[Releases](../../releases)

### Build from source
```bash
# Requirements: Go 1.21+, Node.js 18+, Wails v2
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/meshweb-protocol/meshweb
cd meshweb/meshweb-gui
wails build
```

---

## Getting Started

1. Download and install Meshweb
2. Launch the app
3. Create your account (seed phrase generated)
4. Click **"Join Meshweb"**
5. You are now a node ✅

**Upload a file:**
- Go to Storage tab
- Drag & drop or click Upload
- Share the generated link

**Download a file:**
- Go to Storage tab
- Paste a meshweb:// link
- File downloads automatically

---

## Architecture

| Layer | Technology | Purpose |
|---|---|---|
| Transport | LibP2P | P2P connectivity |
| Discovery | Kademlia DHT | Serverless routing |
| Relay | Circuit Relay v2 | NAT traversal |
| Encryption | AES-256-GCM | Privacy |
| Redundancy | Reed-Solomon | Fault tolerance |
| Identity | Ed25519 + BIP39 | Self-sovereign |

---

## Whitepaper

Read the full technical whitepaper:
[Meshweb Whitepaper](./Meshweb_Whitepaper.md)

---

## Roadmap

- [x] P2P Protocol (LibP2P + DHT)
- [x] File Storage (AES + Reed-Solomon)
- [x] Desktop GUI (Windows)
- [x] Self-sovereign Identity
- [x] Multi-language support
- [ ] Compute Market (MWCoin)
- [ ] Mac & Linux support
- [ ] Mobile app
- [ ] Smart contract (MWCoin mainnet)

---

## Contributing

Meshweb is open source and welcomes
contributions. See [CONTRIBUTING.md](./CONTRIBUTING.md)

---

## License

MIT License — see [LICENSE](./LICENSE)

---

## Disclaimer

Meshweb is experimental software.
Use at your own risk.
The creators are anonymous by design,
similar to Bitcoin's origin.
