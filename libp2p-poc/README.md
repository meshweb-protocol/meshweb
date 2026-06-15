# Meshweb: LibP2P Kademlia DHT Proof of Concept

Ushbu loyiha Meshweb arxitekturasining asosini — ikkita markazlashmagan (serverless) node'ning bir-birini **Kademlia DHT** orqali topib, to'g'ridan-to'g'ri P2P xabar almashishini namoyish etadi.

## Talablar
*   [Go (Golang)](https://go.dev/dl/) o'rnatilgan bo'lishi kerak (v1.20 yoki undan yuqori tavsiya etiladi).

## Qanday ishlatish kerak?

1. **Terminalni (CMD/PowerShell) oching** va ushbu papkaga kiring:
   ```bash
   cd e:\MeshWeb\libp2p-poc
   ```

2. **Go Module'ni ishga tushiring (agar qilinmagan bo'lsa):**
   ```bash
   go mod init meshweb/libp2p-poc
   ```

3. **Kerakli kutubxonalarni (LibP2P) yuklab oling:**
   ```bash
   go get github.com/libp2p/go-libp2p
   go get github.com/libp2p/go-libp2p-kad-dht
   go get github.com/multiformats/go-multiaddr
   go mod tidy
   ```

4. **Dasturni ishga tushiring:**
   ```bash
   go run main.go
   ```

## Dastur nima ish qiladi? (Qadam-ba-qadam)
Ushbu bitta `main.go` fayli quyidagi vazifalarni bajaradi:
1. **1-qadam:** Xotirada 2 ta mustaqil LibP2P node yaratadi (10001 va 10002 portlarida).
2. **2-qadam:** Har bir node o'zining unikal **Peer ID** sini ekranga chiqaradi.
3. **3-qadam:** Node 2 Node 1 ga boshlang'ich nuqta sifatida ulanadi. Shu ondan boshlab markaziy server yo'qoladi va Kademlia DHT tarmog'i shakllanadi.
4. **4-qadam:** Node 1 Node 2 ning qayerdaligini qidiradi. U faqat Node 2 ning `Peer ID` sini biladi. DHT orqali uning aniq IP manzilini va portini topadi.
5. **5-qadam:** Node 1 topilgan manzilga to'g'ridan-to'g'ri ulanib, `/meshweb/1.0.0` protokoli orqali **"Meshweb Genesis"** xabarini yuboradi.
6. **6-qadam:** Node 2 xabarni qabul qiladi va ekranga chop etadi.

*Barcha jarayon markaziy serverlarsiz, ochiq kodli va ishonchli shifrlangan P2P protokoli ustida yuz beradi.*
