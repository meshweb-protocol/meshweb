package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
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
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ResourceAnnouncement struct {
	PeerID string  `json:"peer_id"`
	CPU    float64 `json:"cpu_percent"`
	RAM    float64 `json:"ram_percent"`
}



type App struct {
	ctx             context.Context
	node            host.Host
	idht            *dht.IpfsDHT
	ps              *pubsub.PubSub
	jobTopic        *pubsub.Topic
	resultTopic     *pubsub.Topic
	isConnected     bool
	myPeerID        string
	inviteLink      string
	balance         float64
	todayIncome     float64
	totalIncome     float64
	mu              sync.Mutex
	offerResources  bool
	privKey         crypto.PrivKey
	
	availableNodes  map[string]ResourceAnnouncement
	activeRentals   map[string]*RentalJob
}

func NewApp() *App {
	return &App{
		balance:         0.0,
		offerResources:  false,
		availableNodes:  make(map[string]ResourceAnnouncement),
		activeRentals:   make(map[string]*RentalJob),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadBalance()
}

func (a *App) saveBalance() {
	configDir, _ := os.UserConfigDir()
	path := filepath.Join(configDir, "meshweb-gui", "balance.json")
	data, _ := json.Marshal(map[string]float64{
		"balance": a.balance,
		"today":   a.todayIncome,
		"total":   a.totalIncome,
	})
	os.WriteFile(path, data, 0644)
}

func (a *App) loadBalance() {
	configDir, _ := os.UserConfigDir()
	path := filepath.Join(configDir, "meshweb-gui", "balance.json")
	data, err := os.ReadFile(path)
	if err == nil {
		var b map[string]float64
		json.Unmarshal(data, &b)
		a.balance = b["balance"]
		a.todayIncome = b["today"]
		a.totalIncome = b["total"]
	}
}

func (a *App) logEvent(msg string) {
	runtime.EventsEmit(a.ctx, "activity-log", msg)
}

// ---------------------------------------------
// Wails Bindings for React
// ---------------------------------------------

func (a *App) StartNewNetwork() map[string]interface{} {
	if a.isConnected {
		return map[string]interface{}{"success": true, "inviteLink": a.inviteLink}
	}
	if a.balance == 0 {
		a.balance = 100.0 // Xaridor boshlang'ich balansi
		a.saveBalance()
	}
	err := a.initNode("")
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}
	return map[string]interface{}{"success": true, "inviteLink": a.inviteLink}
}

func (a *App) ConnectToNetwork(invite string) map[string]interface{} {
	if a.isConnected {
		return map[string]interface{}{"success": true, "inviteLink": a.inviteLink}
	}
	err := a.initNode(invite)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}
	
	// Kutamiz (Relay dan Reservation olinishi uchun)
	a.logEvent("[System] Waiting for Relay reservation (5s)...")
	time.Sleep(5 * time.Second)
	a.logEvent("[System] Invite link generated ✅")
	
	return map[string]interface{}{"success": true, "inviteLink": a.inviteLink}
}

func (a *App) FindAvailableNodes(cpuCores int, ramGB int, hasGPU bool) map[string]interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()

	const relayPeerID = "12D3KooWRdvwz59ErP1e6pxqxpYY6rNFTYPCDYGK8eoH9cd4obfq"
	var nodes []map[string]interface{}

	// First: use PubSub-announced nodes (these are real meshweb peers that
	// have broadcast their resource info on the meshweb-nodes topic).
	for peerID, info := range a.availableNodes {
		if peerID == a.myPeerID || peerID == relayPeerID {
			continue
		}
		latency := 20 + int(info.CPU)%60
		nodes = append(nodes, map[string]interface{}{
			"peer_id": peerID,
			"latency": latency,
			"cpu":     info.CPU,
			"ram":     info.RAM,
		})
	}

	// Fallback: if PubSub announcements haven't arrived yet, list all
	// peers we're actually connected to (relay-connected meshweb nodes)
	// so the user can at least attempt a rent request.
	if len(nodes) == 0 && a.node != nil {
		for _, p := range a.node.Network().Peers() {
			pID := p.String()
			if pID == a.myPeerID || pID == relayPeerID {
				continue
			}
			nodes = append(nodes, map[string]interface{}{
				"peer_id": pID,
				"latency": 45,
				"cpu":     0.0,
				"ram":     0.0,
			})
		}
	}

	return map[string]interface{}{
		"success": true,
		"nodes":   nodes,
	}
}

func (a *App) ToggleOfferResources(offer bool) {
	a.mu.Lock()
	a.offerResources = offer
	a.mu.Unlock()
}

func (a *App) GetStartupFile() string {
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if strings.HasSuffix(arg, ".meshweb") || strings.HasPrefix(arg, "meshweb://file/") {
			return arg
		}
	}
	return ""
}

func (a *App) SelectFile() string {
	path, _ := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Fayl tanlang",
	})
	return path
}

func (a *App) SelectFolder() string {
	result, _ := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder to Upload",
	})
	return result
}

func (a *App) uploadDir(dirPath string, isRoot bool) (MeshwebFile, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return MeshwebFile{}, err
	}

	var files []MeshwebFile
	var totalSize int64

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			subFolder, err := a.uploadDir(fullPath, false)
			if err == nil {
				files = append(files, subFolder)
				totalSize += subFolder.FileSize
			}
		} else {
			result := a.UploadFile(fullPath)
			if result["success"].(bool) {
				meta := result["meta"].(MeshwebFile)
				files = append(files, meta)
				totalSize += meta.FileSize
				os.Remove(filepath.Join(getStorageDir(), meta.FileID+".meshweb"))
			}
		}
	}

	folderMeta := MeshwebFile{
		Version:   "1.0",
		Type:      "folder",
		FileName:  filepath.Base(dirPath),
		FileSize:  totalSize,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		CreatorID: "MW-" + a.GetPublicKey()[len(a.GetPublicKey())-8:],
		LocalPath: dirPath,
		Files:     files,
	}

	folderMetaBytes, _ := json.MarshalIndent(folderMeta, "", "  ")
	res := a.UploadData(folderMetaBytes, filepath.Base(dirPath), dirPath)
	if res["success"].(bool) {
		finalMeta := res["meta"].(MeshwebFile)
		finalMeta.Type = "folder"
		finalMeta.FileName = filepath.Base(dirPath)
		finalMeta.Files = files

		if isRoot {
			finalMetaBytes, _ := json.MarshalIndent(finalMeta, "", "  ")
			os.WriteFile(filepath.Join(getStorageDir(), finalMeta.FileID+".meshweb"), finalMetaBytes, 0644)
		} else {
			os.Remove(filepath.Join(getStorageDir(), finalMeta.FileID+".meshweb"))
		}
		return finalMeta, nil
	}

	return folderMeta, nil
}

func (a *App) UploadFolder(folderPath string) map[string]interface{} {
	a.logEvent(fmt.Sprintf("[Storage] Uploading folder: %s", filepath.Base(folderPath)))

	folderMeta, err := a.uploadDir(folderPath, true)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}

	a.logEvent(fmt.Sprintf("[Storage] Folder uploaded ✅"))

	return map[string]interface{}{
		"success": true,
		"meta":    folderMeta,
	}
}

func (a *App) GetDashboardStats() map[string]interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()

	c, _ := cpu.Percent(0, false)
	cpuVal := 0.0
	if len(c) > 0 {
		cpuVal = c[0]
	}
	m, _ := mem.VirtualMemory()
	ramVal := 0.0
	if m != nil {
		ramVal = m.UsedPercent
	}

	// Count meshweb peers — exclude the relay server itself and only count
	// peers that are not the well-known relay ID.
	const relayPeerID = "12D3KooWRdvwz59ErP1e6pxqxpYY6rNFTYPCDYGK8eoH9cd4obfq"
	peers := 0
	if a.node != nil {
		for _, p := range a.node.Network().Peers() {
			if p.String() != relayPeerID {
				peers++
			}
		}
	}
	activeJobs := 0
	for _, r := range a.activeRentals {
		if r.IsActive {
			activeJobs++
		}
	}

	return map[string]interface{}{
		"connected":      a.isConnected,
		"peerId":         a.myPeerID,
		"inviteLink":     a.inviteLink,
		"balance":        a.balance,
		"todayIncome":    a.todayIncome,
		"totalIncome":    a.totalIncome,
		"cpu":            cpuVal,
		"ram":            ramVal,
		"connectedPeers": peers,
		"activeJobs":     activeJobs,
	}
}

// ---------------------------------------------
// Meshweb Core Logic
// ---------------------------------------------

func getPublicIP() string {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(ip))
}

func generateInviteLink(node host.Host, port int) string {
	ip := getPublicIP()
	if ip == "" {
		ip = "127.0.0.1"
	}
	return fmt.Sprintf("meshweb://%s:%d/%s", ip, port, node.ID().String())
}

func parseInviteLink(link string) string {
	link = strings.TrimPrefix(link, "meshweb://")
	
	if !strings.Contains(link, "/") && !strings.Contains(link, ":") {
		return "/p2p/" + link
	}

	parts := strings.Split(link, "/")
	if len(parts) == 2 {
		ipPort := strings.Split(parts[0], ":")
		if len(ipPort) == 2 {
			// IP ni tashlab yuboramiz, faqat Peer ID orqali Relay tarmog'idan izlashga majburlaymiz!
			return "/p2p/" + parts[1]
		}
	}
	return ""
}

func loadBootstrapNodes() []string {
	configDir, _ := os.UserConfigDir()
	bootstrapPath := filepath.Join(configDir, "meshweb-gui", "bootstrap.json")
	file, err := os.ReadFile(bootstrapPath)
	if err == nil {
		var addrs []string
		if json.Unmarshal(file, &addrs) == nil && len(addrs) > 0 {
			return addrs
		}
	}
	defaultBootstraps := []string{
		"/ip4/185.177.116.13/tcp/443/p2p/12D3KooWRdvwz59ErP1e6pxqxpYY6rNFTYPCDYGK8eoH9cd4obfq",
		"/ip4/185.177.116.13/udp/443/quic-v1/p2p/12D3KooWRdvwz59ErP1e6pxqxpYY6rNFTYPCDYGK8eoH9cd4obfq",
	}
	data, _ := json.MarshalIndent(defaultBootstraps, "", "  ")
	os.WriteFile(bootstrapPath, data, 0644)
	return defaultBootstraps
}

func addToBootstrap(maddr string) {
	bootstraps := loadBootstrapNodes()
	for _, b := range bootstraps {
		if b == maddr {
			return
		}
	}
	bootstraps = append(bootstraps, maddr)
	data, _ := json.MarshalIndent(bootstraps, "", "  ")
	configDir, _ := os.UserConfigDir()
	bootstrapPath := filepath.Join(configDir, "meshweb-gui", "bootstrap.json")
	os.WriteFile(bootstrapPath, data, 0644)
}

func (a *App) startBootstrapSweeper() {
	go func() {
		for {
			time.Sleep(30 * time.Second)
			bootstraps := loadBootstrapNodes()
			var activeBootstraps []string
			changed := false

			for _, bAddrStr := range bootstraps {
				// Asosiy Relay tugunini o'chirmaymiz
				if strings.Contains(bAddrStr, "185.177.116.13") || strings.Contains(bAddrStr, "bootstrap.libp2p.io") {
					activeBootstraps = append(activeBootstraps, bAddrStr)
					continue
				}

				maddr, err := multiaddr.NewMultiaddr(bAddrStr)
				if err != nil {
					changed = true
					continue
				}
				addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
				if err != nil {
					changed = true
					continue
				}

				// 10 soniya ichida ulanib ko'ramiz
				ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
				err = a.node.Connect(ctx, *addrInfo)
				cancel()

				if err != nil {
					changed = true
					a.logEvent(fmt.Sprintf("[Network] Inactive node removed: %s", addrInfo.ID.String()[:8]))
				} else {
					activeBootstraps = append(activeBootstraps, bAddrStr)
				}
			}

			if changed {
				data, _ := json.MarshalIndent(activeBootstraps, "", "  ")
				configDir, _ := os.UserConfigDir()
				bootstrapPath := filepath.Join(configDir, "meshweb-gui", "bootstrap.json")
				os.WriteFile(bootstrapPath, data, 0644)
			}
		}
	}()
}

func (a *App) initNode(invite string) error {
	var idht *dht.IpfsDHT
	var err error

	relayAddrStr := "/ip4/185.177.116.13/tcp/443/p2p/12D3KooWRdvwz59ErP1e6pxqxpYY6rNFTYPCDYGK8eoH9cd4obfq"
	relayMaddr, _ := multiaddr.NewMultiaddr(relayAddrStr)
	relayAddrInfo, _ := peer.AddrInfoFromP2pAddr(relayMaddr)

	opts := []libp2p.Option{
		libp2p.Identity(a.privKey),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/tcp/0/ws",
			"/ip4/0.0.0.0/udp/0/quic-v1",
		),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
		libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{*relayAddrInfo}),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err = dht.New(a.ctx, h)
			return idht, err
		}),
	}

	node, err := libp2p.New(
		append(opts, libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/443",
			"/ip4/0.0.0.0/tcp/443/ws",
			"/ip4/0.0.0.0/udp/443/quic-v1",
		))...,
	)
	if err != nil {
		// Port 443 band bo'lsa yoki admin ruxsati yo'q bo'lsa random portga o'tamiz
		node, err = libp2p.New(
			append(opts, libp2p.ListenAddrStrings(
				"/ip4/0.0.0.0/tcp/0",
				"/ip4/0.0.0.0/tcp/0/ws",
				"/ip4/0.0.0.0/udp/0/quic-v1",
			))...,
		)
	}

	if err != nil {
		return err
	}

	node.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			if c.RemotePeer() == relayAddrInfo.ID {
				a.logEvent("[Relay] Connected to server, Reservation in progress...")
			}
		},
	})

	a.mu.Lock()
	a.node = node
	a.idht = idht
	a.myPeerID = node.ID().String()
	a.inviteLink = generateInviteLink(node, 443)
	a.isConnected = true
	a.mu.Unlock()

	// Aniq Relay ulanishi va Reservation (Avtomatik emas, majburiy)
	go func() {
		ctxTimeout, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		
		a.logEvent("[Relay] Connecting to server...")
		if err := a.node.Connect(ctxTimeout, *relayAddrInfo); err != nil {
			a.logEvent(fmt.Sprintf("[Relay] Server connection error: %v", err))
			return
		}
		a.logEvent("[Relay] Connected ✅, Requesting Reservation...")
		
		reservation, err := client.Reserve(a.ctx, a.node, *relayAddrInfo)
		if err != nil {
			a.logEvent(fmt.Sprintf("[Relay] Reservation error: %v", err))
		} else {
			a.logEvent(fmt.Sprintf("[Relay] Reservation acquired ✅ (Until: %v)", reservation.Expiration))
		}
	}()

	a.logEvent(fmt.Sprintf("[Node Started] Peer ID: %s", a.myPeerID[:8]))
	a.logEvent("[Node] AutoRelay and HolePunching enabled ✅")
	a.logEvent("[Node] WebSocket transport enabled ✅")
	a.logEvent("[Node] QUIC transport enabled ✅")

	a.setupStorageHandler()
	a.setupRentHandler()
	go a.rentBillingLoop()

	a.startBootstrapSweeper()

	go a.networkLoop(invite)
	return nil
}

func (a *App) networkLoop(invite string) {
	bootstraps := loadBootstrapNodes()
	var relayFound bool
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, bAddrStr := range bootstraps {
		wg.Add(1)
		go func(addrStr string) {
			defer wg.Done()
			maddr, err := multiaddr.NewMultiaddr(addrStr)
			if err != nil {
				return
			}
			ctxTimeout, cancel := context.WithTimeout(a.ctx, 60*time.Second)
			defer cancel()

			addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
			if err != nil {
				return
			}

			if addrInfo.ID != a.node.ID() {
				hostName := strings.Split(addrStr, "/p2p")[0]
				err := a.node.Connect(ctxTimeout, *addrInfo)
				if err == nil {
					mu.Lock()
					if !relayFound {
						relayFound = true
						a.logEvent(fmt.Sprintf("[Node] Successfully found Relay ✅ (%s)", hostName))
					}
					mu.Unlock()
				}
			}
		}(bAddrStr)
	}
	
	// Wait for bootstraps to connect before proceeding
	wg.Wait()

	// Endi DHT ni bootstrap qilamiz
	a.idht.Bootstrap(a.ctx)

	routingDiscovery := drouting.NewRoutingDiscovery(a.idht)
	util.Advertise(a.ctx, routingDiscovery, "meshweb-network")

	// Create GossipSub with DHT-based discovery so relay-connected peers
	// are added to the mesh automatically via WithDiscovery option.
	ps, _ := pubsub.NewGossipSub(a.ctx, a.node,
		pubsub.WithDiscovery(routingDiscovery),
	)
	a.ps = ps

	// Jobs pubsub
	jobTopic, _ := ps.Join("meshweb-jobs")
	a.jobTopic = jobTopic

	resultTopic, _ := ps.Join("meshweb-results")
	a.resultTopic = resultTopic

	// Nodes pubsub
	nodesTopic, _ := ps.Join("meshweb-nodes")
	nodesSub, _ := nodesTopic.Subscribe()

	// Discovery logic
	go func() {
		for {
			msg, err := nodesSub.Next(a.ctx)
			if err != nil || msg.ReceivedFrom == a.node.ID() {
				continue
			}
			var ann ResourceAnnouncement
			if err := json.Unmarshal(msg.Data, &ann); err == nil {
				a.mu.Lock()
				a.availableNodes[msg.ReceivedFrom.String()] = ann
				a.mu.Unlock()
			}
		}
	}()

	// Announce helper – publishes this node's info to the meshweb-nodes topic
	announceNode := func() {
		c, _ := cpu.Percent(0, false)
		v, _ := mem.VirtualMemory()
		cpuVal := 0.0
		if len(c) > 0 {
			cpuVal = c[0]
		}
		ann := ResourceAnnouncement{
			PeerID: a.node.ID().String(),
			CPU:    cpuVal,
			RAM:    v.UsedPercent,
		}
		b, _ := json.Marshal(ann)
		nodesTopic.Publish(a.ctx, b)
	}

	// React to any new peer connecting: wait 3s for GossipSub mesh to form,
	// then announce twice to ensure the remote peer receives our info.
	a.node.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			go func() {
				time.Sleep(3 * time.Second)
				a.logEvent(fmt.Sprintf("[PubSub] Node discovered: %s", c.RemotePeer().String()[:8]))
				announceNode()
				announceNode() // Send twice for reliability
			}()
		},
	})

	// Periodic announce every 5s so nodes stay visible quickly after joining.
	go func() {
		announceNode() // announce immediately
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			announceNode()
		}
	}()

	// Job pubsub and result pubsub logic removed as part of simulation cleanup

	time.Sleep(3 * time.Second)

	if invite != "" {
		maddrStr := parseInviteLink(invite)
		if maddrStr != "" {
			maddr, err := multiaddr.NewMultiaddr(maddrStr)
			if err == nil {
				addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
				if err == nil && addrInfo.ID != a.node.ID() {
					a.logEvent("[Node] Connecting...")
					a.logEvent("[Node] HolePunch coordination...")
					
					if err := a.node.Connect(a.ctx, *addrInfo); err == nil {
						a.logEvent("[Node] Connected directly ✅")
						a.logEvent(fmt.Sprintf("[PubSub] Node %s discovered ✅", addrInfo.ID.String()[:8]))
						addToBootstrap(invite)
					} else {
						a.logEvent("[Node] Direct connection blocked. Connecting via Relay fallback...")
						
						relayAddrStr := "/ip4/185.177.116.13/tcp/443/p2p/12D3KooWRdvwz59ErP1e6pxqxpYY6rNFTYPCDYGK8eoH9cd4obfq/p2p-circuit/p2p/" + addrInfo.ID.String()
						relayMaddr, _ := multiaddr.NewMultiaddr(relayAddrStr)
						
						relayAddrInfo := peer.AddrInfo{
							ID:    addrInfo.ID,
							Addrs: []multiaddr.Multiaddr{relayMaddr},
						}
						
						if err := a.node.Connect(a.ctx, relayAddrInfo); err == nil {
							a.logEvent("[Node] Connected via Relay ✅")
							a.logEvent(fmt.Sprintf("[PubSub] Node %s discovered ✅", addrInfo.ID.String()[:8]))
							addToBootstrap(invite)
						} else {
							a.logEvent(fmt.Sprintf("[Error] Failed to connect: %v", err))
						}
					}
				}
			}
		}
	}
}
