package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/reedsolomon"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// Layer 2
type ResourceAnnouncement struct {
	PeerID string  `json:"peer_id"`
	CPU    float64 `json:"cpu_percent"`
	RAM    float64 `json:"ram_percent"`
}

// Layer 7 (Decentralized Market)
type ComputeJob struct {
	JobID       string  `json:"job_id"`
	BuyerID     string  `json:"buyer_id"`
	CPUCores    int     `json:"cpu_cores"`
	DurationSec int     `json:"duration_sec"`
	PriceMWCoin float64 `json:"price_mwcoin"`
}

type JobResult struct {
	JobID  string `json:"job_id"`
	NodeID string `json:"node_id"`
	Status string `json:"status"`
}

// Layer 4
type StorageMsg struct {
	Type       string `json:"type"`
	FileID     string `json:"file_id"`
	ShardIndex int    `json:"shard_index"`
	Data       []byte `json:"data,omitempty"`
}

var (
	balances   = make(map[string]float64)
	balancesMu sync.Mutex

	buyerActiveJobs   = make(map[string]bool)
	buyerActiveJobsMu sync.Mutex

	localShards   = make(map[string]map[int][]byte)
	localShardsMu sync.Mutex

	fileKeys   = make(map[string][]byte)
	fileKeysMu sync.Mutex
)

// =========================================================================
// LAYER 5 & 6: ADVANCED NETWORKING (IP, AutoRelay, HolePunching)
// =========================================================================
func getPublicIP() string {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil { return "" }
	defer resp.Body.Close()
	ip, err := io.ReadAll(resp.Body)
	if err != nil { return "" }
	return strings.TrimSpace(string(ip))
}

func getBestAddress(h host.Host) string {
	bestAddr := ""
	for _, a := range h.Addrs() {
		s := a.String()
		if !strings.Contains(s, "127.0.0.1") && !strings.Contains(s, "169.254.") {
			bestAddr = s
		}
	}
	if bestAddr == "" && len(h.Addrs()) > 0 { bestAddr = h.Addrs()[0].String() }
	return bestAddr
}

func generateInviteLink(h host.Host, port int) string {
	publicIP := getPublicIP()
	if publicIP == "" {
		addr := getBestAddress(h)
		parts := strings.Split(addr, "/")
		for i, p := range parts {
			if p == "ip4" && i+1 < len(parts) { publicIP = parts[i+1] }
		}
	}
	if publicIP == "" { publicIP = "127.0.0.1" }
	return fmt.Sprintf("meshweb://%s:%d/%s", publicIP, port, h.ID().String())
}

func parseInviteLink(link string) (string, error) {
	link = strings.TrimPrefix(link, "meshweb://")
	parts := strings.Split(link, "/")
	if len(parts) != 2 { return "", fmt.Errorf("Xato havola formati") }
	ipPort := strings.Split(parts[0], ":")
	if len(ipPort) != 2 { return "", fmt.Errorf("Xato IP/PORT") }
	return fmt.Sprintf("/ip4/%s/tcp/%s/p2p/%s", ipPort[0], ipPort[1], parts[1]), nil
}

func addToBootstrap(maddr string) {
	bootstraps := loadBootstrapNodes()
	for _, b := range bootstraps {
		if b == maddr { return }
	}
	bootstraps = append(bootstraps, maddr)
	data, _ := json.MarshalIndent(bootstraps, "", "  ")
	os.WriteFile("bootstrap.json", data, 0644)
}

func loadBootstrapNodes() []string {
	file, err := os.ReadFile("bootstrap.json")
	if err == nil {
		var addrs []string
		if json.Unmarshal(file, &addrs) == nil && len(addrs) > 0 {
			return addrs
		}
	}

	defaultBootstraps := []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmRvOWkpiCF5yXq29rE7A1gAM36yQh4U8f5vFvX7FDEgGq",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	}

	data, _ := json.MarshalIndent(defaultBootstraps, "", "  ")
	os.WriteFile("bootstrap.json", data, 0644)
	return defaultBootstraps
}

// =========================================================================
// MAIN LOGIC
// =========================================================================
func main() {
	port := flag.Int("port", 0, "TCP port")
	isBuyer := flag.Bool("buyer", false, "Xaridor rejimi")
	flag.Parse()

	inviteLinks := []string{}
	for _, arg := range flag.Args() {
		if strings.HasPrefix(arg, "meshweb://") {
			maddrStr, err := parseInviteLink(arg)
			if err == nil { inviteLinks = append(inviteLinks, maddrStr) }
		}
	}

	ctx := context.Background()
	addr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port)
	var idht *dht.IpfsDHT
	node, err := libp2p.New(
		libp2p.ListenAddrStrings(addr),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			idht, err = dht.New(ctx, h)
			return idht, err
		}),
	)
	if err != nil { log.Fatal("Node yaratishda xato:", err) }
	defer node.Close()

	if *isBuyer { balances[node.ID().String()] = 100.0 } else { balances[node.ID().String()] = 0.0 }

	fmt.Println("=====================================================")
	fmt.Printf("[Node ishga tushdi] Peer ID: %s\n", node.ID())
	fmt.Println("[Node] AutoRelay yoqildi ✅")
	fmt.Println("[Node] HolePunching yoqildi ✅")
	invite := generateInviteLink(node, *port)
	fmt.Printf("\nDo'stingizga shu linkni yuboring:\n%s\n", invite)
	fmt.Println("=====================================================\n")

	// =========================================================================
	// LAYER 4: STORAGE HANDLER
	// =========================================================================
	node.SetStreamHandler("/meshweb/storage/1.0.0", func(s network.Stream) {
		reader := bufio.NewReader(s)
		msgStr, err := reader.ReadString('\n')
		if err != nil { return }
		var msg StorageMsg
		if err := json.Unmarshal([]byte(msgStr), &msg); err != nil { return }

		if msg.Type == "STORE" {
			localShardsMu.Lock()
			if localShards[msg.FileID] == nil { localShards[msg.FileID] = make(map[int][]byte) }
			localShards[msg.FileID][msg.ShardIndex] = msg.Data
			localShardsMu.Unlock()
		} else if msg.Type == "FETCH_REQ" {
			localShardsMu.Lock()
			shards := localShards[msg.FileID]
			localShardsMu.Unlock()

			if shards != nil {
				for idx, data := range shards {
					rep := StorageMsg{Type: "FETCH_RES", FileID: msg.FileID, ShardIndex: idx, Data: data}
					b, _ := json.Marshal(rep)
					s.Write(append(b, '\n'))
				}
			}
			s.Close()
		}
	})

	// =========================================================================
	// DHT Bootstrap & Ulanish
	// =========================================================================
	if err := idht.Bootstrap(ctx); err != nil { log.Fatal("DHT Bootstrap xatosi:", err) }

	if len(inviteLinks) > 0 {
		fmt.Println("[Node] Ulanmoqda...")
		for _, link := range inviteLinks {
			maddr, err := multiaddr.NewMultiaddr(link)
			if err == nil {
				addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
				if err == nil && addrInfo.ID != node.ID() {
					fmt.Println("[Node] HolePunch koordinatsiyasi...")
					
					// Dastlab to'g'ridan-to'g'ri (Direct IP) ulanishga harakat qilamiz
					if err := node.Connect(ctx, *addrInfo); err == nil {
						fmt.Println("[Node] To'g'ridan-to'g'ri ulandi ✅")
						fmt.Printf("[PubSub] Node %s ko'rindi ✅\n", addrInfo.ID.String()[:8])
						addToBootstrap(link)
					} else {
						// Agar to'g'ridan-to'g'ri ulanmasa, xato ekanligini yashirmaymiz.
						// Ulanish xatosi chiqqanda DHT orqali (Relay serverlar yordamida) qidirib ko'ramiz
						fmt.Println("[Node] To'g'ridan-to'g'ri ulanish bloklandi. Relay aylanma yo'li qidirilmoqda...")
						
						// Faqat Peer ID ni berib, qolgan IP qidirishni DHT ga (IPFS tarmog'iga) topshiramiz
						relayAddrInfo := peer.AddrInfo{ID: addrInfo.ID}
						if err := node.Connect(ctx, relayAddrInfo); err == nil {
							fmt.Println("[Node] Relay orqali ulandi ✅")
							fmt.Printf("[PubSub] Node %s ko'rindi ✅\n", addrInfo.ID.String()[:8])
							addToBootstrap(link)
						} else {
							fmt.Println("[Xato] Ikkala kompyuter ham juda kuchli NAT/Firewall orqasida yoki Relay tarmog'iga to'liq sinxron bo'lmadi.")
							fmt.Printf("Xato tafsiloti: %v\n> ", err)
						}
					}
				}
			}
		}
	} else {
		bootstraps := loadBootstrapNodes()
		var relayFound bool
		for _, bAddrStr := range bootstraps {
			maddr, err := multiaddr.NewMultiaddr(bAddrStr)
			if err == nil {
				addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
				if err == nil && addrInfo.ID != node.ID() {
					if err := node.Connect(ctx, *addrInfo); err == nil {
						if strings.Contains(bAddrStr, "bootstrap.libp2p.io") { relayFound = true }
					}
				}
			}
		}
		if relayFound { fmt.Println("[Node] Public relay topildi ✅") } else if len(bootstraps) > 0 { fmt.Println(">> Bootstrap node'larga ulandi") }
	}

	routingDiscovery := drouting.NewRoutingDiscovery(idht)
	util.Advertise(ctx, routingDiscovery, "meshweb-network")
	
	ps, _ := pubsub.NewGossipSub(ctx, node)
	
	// LAYER 2
	topic, _ := ps.Join("meshweb-resources")
	sub, _ := topic.Subscribe()
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil || msg.ReceivedFrom == node.ID() { continue }
			var ann ResourceAnnouncement
			json.Unmarshal(msg.Data, &ann)
		}
	}()
	go func() {
		for {
			c, _ := cpu.Percent(time.Second, false)
			m, _ := mem.VirtualMemory()
			cpuVal := 0.0
			if len(c) > 0 { cpuVal = c[0] }
			ann := ResourceAnnouncement{PeerID: node.ID().String(), CPU: cpuVal, RAM: m.UsedPercent}
			data, _ := json.Marshal(ann)
			topic.Publish(ctx, data)
			time.Sleep(15 * time.Second)
		}
	}()

	// =========================================================================
	// LAYER 7: DECENTRALIZED PUBSUB MARKET
	// =========================================================================
	jobTopic, _ := ps.Join("meshweb-jobs")
	jobSub, _ := jobTopic.Subscribe()

	resultTopic, _ := ps.Join("meshweb-results")
	resultSub, _ := resultTopic.Subscribe()

	// Ishchilar (Workers) mantiq
	go func() {
		for {
			msg, err := jobSub.Next(ctx)
			if err != nil || msg.ReceivedFrom == node.ID() { continue }
			var job ComputeJob
			if err := json.Unmarshal(msg.Data, &job); err == nil {
				fmt.Printf("\n[Node %s] %s qabul qilindi ✅\n> ", node.ID().String()[:8], job.JobID)
				
				go func(j ComputeJob) {
					// 5 soniya davomida ishni bajarish imitatsiyasi
					time.Sleep(5 * time.Second)
					
					fmt.Printf("\n[Node %s] Bajarildi -> natija yuborildi\n> ", node.ID().String()[:8])
					res := JobResult{JobID: j.JobID, NodeID: node.ID().String(), Status: "DONE"}
					b, _ := json.Marshal(res)
					resultTopic.Publish(ctx, b)
				}(job)
			}
		}
	}()

	// Xaridor (Buyer) mantiq
	if *isBuyer {
		// Natijalarni qabul qilish
		go func() {
			for {
				msg, err := resultSub.Next(ctx)
				if err != nil || msg.ReceivedFrom == node.ID() { continue }
				var res JobResult
				if err := json.Unmarshal(msg.Data, &res); err == nil && res.Status == "DONE" {
					buyerActiveJobsMu.Lock()
					if buyerActiveJobs[res.JobID] {
						// Ish yakunlandi, birinchi kelgan natijaga to'laymiz
						buyerActiveJobs[res.JobID] = false
						
						balancesMu.Lock()
						balances[node.ID().String()] -= 0.5
						balances[res.NodeID] += 0.5
						balancesMu.Unlock()

						fmt.Printf("\n[Xaridor] Natija olindi -> 0.5 MWC to'landi ✅ (Node: %s)\n> ", res.NodeID[:8])
					}
					buyerActiveJobsMu.Unlock()
				}
			}
		}()

		// Har 20 soniyada yangi ish e'lon qilish
		go func() {
			time.Sleep(10 * time.Second)
			for {
				jobID := fmt.Sprintf("job-%d", mrand.Intn(10000))
				job := ComputeJob{JobID: jobID, BuyerID: node.ID().String(), CPUCores: 2, DurationSec: 5, PriceMWCoin: 0.5}
				
				buyerActiveJobsMu.Lock()
				buyerActiveJobs[jobID] = true
				buyerActiveJobsMu.Unlock()

				b, _ := json.Marshal(job)
				fmt.Printf("\n[Xaridor] %s PubSub ga yuborildi\n> ", jobID)
				jobTopic.Publish(ctx, b)
				
				time.Sleep(20 * time.Second)
			}
		}()
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Split(line, " ")
		if len(parts) == 0 || parts[0] == "" { fmt.Print("> "); continue }

		cmd := parts[0]
		if cmd == "store" && len(parts) == 2 {
			handleStore(ctx, node, parts[1])
		} else if cmd == "retrieve" && len(parts) == 2 {
			handleRetrieve(ctx, node, parts[1])
		} else {
			fmt.Println("Noma'lum buyruq. Format: store [fayl] yoki retrieve [fayl_id]")
		}
		fmt.Print("> ")
	}
}

// LAYER 4 CORE FUNCTIONS
func handleStore(ctx context.Context, n host.Host, filepath string) {
	data, err := os.ReadFile(filepath)
	if err != nil { fmt.Printf("Xato: %s fayli topilmadi.\n> ", filepath); return }

	key := make([]byte, 32)
	rand.Read(key)
	fileID := fmt.Sprintf("file_%d", time.Now().Unix())

	fileKeysMu.Lock()
	fileKeys[fileID] = key
	fileKeysMu.Unlock()

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	cipherSize := int64(len(ciphertext))
	sizeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeBytes, uint64(cipherSize))
	fullData := append(sizeBytes, ciphertext...)

	enc, err := reedsolomon.New(10, 20)
	if err != nil { fmt.Println("RS xato:", err); return }

	shards, err := enc.Split(fullData)
	if err != nil { fmt.Println("Split xato:", err); return }
	if err = enc.Encode(shards); err != nil { fmt.Println("Encode xato:", err); return }

	fmt.Printf("\n[Storage] %q shifrlandi.\n", filepath)
	fmt.Printf("[Storage] 30 shard yaratildi. Tarmoqqa tarqatilmoqda...\n")

	peers := n.Network().Peers()
	if len(peers) == 0 {
		fmt.Println("[Storage] Tarmoqda boshqa node yo'q! Shardlar faqat o'zingizda saqlanadi.")
		localShardsMu.Lock()
		localShards[fileID] = make(map[int][]byte)
		for i, s := range shards { localShards[fileID][i] = s }
		localShardsMu.Unlock()
		fmt.Printf("[Storage] Fayl ID: %s. Xavfsiz saqlandi.\n> ", fileID)
		return
	}

	dist := make(map[peer.ID]int)
	for i, s := range shards {
		p := peers[i%len(peers)]
		dist[p]++
		stream, err := n.NewStream(ctx, p, "/meshweb/storage/1.0.0")
		if err == nil {
			msg := StorageMsg{Type: "STORE", FileID: fileID, ShardIndex: i, Data: s}
			b, _ := json.Marshal(msg)
			stream.Write(append(b, '\n'))
			stream.Close()
		}
	}

	for p, count := range dist {
		fmt.Printf("[Node %s] %d shard qabul qildi\n", p.String()[:8], count)
	}
	fmt.Printf("[Storage] Yuklash tugadi! Tiklash ID: %s\n> ", fileID)
}

func handleRetrieve(ctx context.Context, n host.Host, fileID string) {
	fmt.Printf("\n[Retrieve] %s faylini tiklash boshlandi...\n", fileID)
	collected := make([][]byte, 30)
	count := 0
	var mu sync.Mutex

	localShardsMu.Lock()
	if myShards, ok := localShards[fileID]; ok {
		for i, data := range myShards {
			collected[i] = data
			count++
		}
	}
	localShardsMu.Unlock()

	peers := n.Network().Peers()
	var wg sync.WaitGroup

	for _, p := range peers {
		wg.Add(1)
		go func(target peer.ID) {
			defer wg.Done()
			stream, err := n.NewStream(ctx, target, "/meshweb/storage/1.0.0")
			if err != nil { return }
			defer stream.Close()

			msg := StorageMsg{Type: "FETCH_REQ", FileID: fileID}
			b, _ := json.Marshal(msg)
			stream.Write(append(b, '\n'))

			reader := bufio.NewReader(stream)
			for {
				line, err := reader.ReadString('\n')
				if err != nil { break }
				var rep StorageMsg
				if json.Unmarshal([]byte(line), &rep) == nil && rep.Type == "FETCH_RES" {
					mu.Lock()
					if collected[rep.ShardIndex] == nil {
						collected[rep.ShardIndex] = rep.Data
						count++
					}
					mu.Unlock()
				}
			}
		}(p)
	}
	
	wg.Wait()

	fmt.Printf("[Retrieve] %d shard yig'ildi.\n", count)
	if count < 10 {
		fmt.Println("[Xato] Faylni tiklash uchun kamida 10 ta shard kerak. Hozirgi tarmoqda yetarli emas.\n> ")
		return
	}

	enc, _ := reedsolomon.New(10, 20)
	err := enc.Reconstruct(collected)
	if err != nil { fmt.Println("[Xato] Shardlarni birlashtirib bo'lmadi:", err); return }

	buf := new(bytes.Buffer)
	enc.Join(buf, collected, len(collected[0])*10)
	joinedData := buf.Bytes()

	cipherSize := binary.LittleEndian.Uint64(joinedData[:8])
	ciphertext := joinedData[8 : 8+cipherSize]

	fileKeysMu.Lock()
	key := fileKeys[fileID]
	fileKeysMu.Unlock()

	if key == nil {
		fmt.Println("[Xato] Bu fayl uchun AES kalit sizning node'da mavjud emas.\n> ")
		return
	}

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize { fmt.Println("[Xato] Noto'g'ri shifrlangan fayl formati.\n> "); return }

	nonce, ciphertextBody := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBody, nil)
	if err != nil { fmt.Println("[Xato] Shifrni ochishda xato:", err); return }

	outName := "restored_" + fileID + ".bin"
	os.WriteFile(outName, plaintext, 0644)
	fmt.Printf("[Retrieve] ✅ Muvaffaqiyatli! %d shard yordamida fayl tiklandi va shifr ochildi.\n", count)
	fmt.Printf("           -> Saqlandi: %s\n> ", outName)
}
