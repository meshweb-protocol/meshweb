package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/klauspost/reedsolomon"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	mh "github.com/multiformats/go-multihash"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const StorageProtocolID protocol.ID = "/meshweb/storage/1.0.0"

type MeshwebFile struct {
	Version      string `json:"version"`
	FileName     string `json:"file_name"`
	FileSize     int64  `json:"file_size"`
	OriginalSize int    `json:"original_size"` // len(ciphertext) before RS padding
	FileID       string `json:"file_id"`
	Shards       int    `json:"shards"`
	MinShards    int    `json:"min_shards"`
	Encryption   string `json:"encryption"`
	KeyHash      string `json:"key_hash"`
	AESKey       string `json:"aes_key,omitempty"`
	CreatedAt    string `json:"created_at"`
	CreatorID    string `json:"creator_id"`
	LocalPath    string `json:"local_path,omitempty"`
}

type DownloadedFile struct {
	FileName     string `json:"file_name"`
	FileID       string `json:"file_id"`
	FileSize     int64  `json:"file_size"`
	LocalPath    string `json:"local_path"`
	DownloadedAt string `json:"downloaded_at"`
}

type ChunkRequest struct {
	FileID string `json:"file_id"`
	Shard  int    `json:"shard"`
}

type ChunkResponse struct {
	FileID string `json:"file_id"`
	Shard  int    `json:"shard"`
	Data   string `json:"data"` // base64
	Error  string `json:"error,omitempty"`
}

func getStorageDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	appDir := filepath.Join(configDir, "meshweb-gui", "storage")
	os.MkdirAll(appDir, 0755)
	return appDir
}

func getDownloadsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dl := filepath.Join(home, "Downloads")
	os.MkdirAll(dl, 0755)
	return dl
}

// uniqueFilePath returns a non-colliding path in dir for the given filename.
// If "rasm.pdf" already exists, it tries "rasm (1).pdf", "rasm (2).pdf", etc.
func uniqueFilePath(dir, name string) string {
	candidate := filepath.Join(dir, name)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 1000; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return candidate
}

func encryptData(data []byte) ([]byte, []byte, error) {
	key := make([]byte, 32)
	rand.Read(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, key, nil
}

func decryptData(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func (a *App) setupStorageHandler() {
	a.node.SetStreamHandler(StorageProtocolID, func(s network.Stream) {
		defer s.Close()
		
		scanner := bufio.NewScanner(s)
		if !scanner.Scan() {
			return
		}

		var req ChunkRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			return
		}

		shardPath := filepath.Join(getStorageDir(), req.FileID, fmt.Sprintf("shard_%d", req.Shard))
		data, err := os.ReadFile(shardPath)
		
		res := ChunkResponse{FileID: req.FileID, Shard: req.Shard}
		if err != nil {
			res.Error = "shard not found"
		} else {
			res.Data = base64.StdEncoding.EncodeToString(data)
		}

		b, _ := json.Marshal(res)
		b = append(b, '\n')
		s.Write(b)
	})
}

const (
	rsDataShards   = 10
	rsParityShards = 20
	rsTotalShards  = rsDataShards + rsParityShards // 30
)

// Fayl yuklash (Upload) with Reed-Solomon erasure coding
func (a *App) UploadFile(filePath string) map[string]interface{} {
	a.logEvent(fmt.Sprintf("[Storage] Uploading %s...", filepath.Base(filePath)))

	data, err := os.ReadFile(filePath)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}

	// 1. Encrypt
	ciphertext, aesKey, err := encryptData(data)
	if err != nil {
		return map[string]interface{}{"success": false, "error": "Encryption failed"}
	}

	// 2. Hash & CID
	hash := sha256.Sum256(ciphertext)
	mhash, _ := mh.Encode(hash[:], mh.SHA2_256)
	fileCID := cid.NewCidV1(cid.Raw, mhash).String()

	// 3. Reed-Solomon encode: 10 data + 20 parity = 30 shards
	enc, err := reedsolomon.New(rsDataShards, rsParityShards)
	if err != nil {
		return map[string]interface{}{"success": false, "error": "RS encoder init failed"}
	}

	// Split into 10 equal data shards (RS pads automatically)
	shards, err := enc.Split(ciphertext)
	if err != nil {
		return map[string]interface{}{"success": false, "error": "RS split failed"}
	}

	// Compute parity shards
	if err := enc.Encode(shards); err != nil {
		return map[string]interface{}{"success": false, "error": "RS encode failed"}
	}

	// 4. Save all 30 shards to disk
	fileDir := filepath.Join(getStorageDir(), fileCID)
	os.MkdirAll(fileDir, 0755)
	for i, shard := range shards {
		os.WriteFile(filepath.Join(fileDir, fmt.Sprintf("shard_%d", i)), shard, 0644)
	}

	// 5. Save .meshweb metadata (store original ciphertext size for RS trim)
	keyHash := sha256.Sum256(aesKey)
	meta := MeshwebFile{
		Version:      "1.0",
		FileName:     filepath.Base(filePath),
		FileSize:     int64(len(data)),
		OriginalSize: len(ciphertext), // needed to trim RS padding on download
		FileID:       fileCID,
		Shards:       rsTotalShards,
		MinShards:    rsDataShards,
		Encryption:   "AES-256-GCM",
		KeyHash:      hex.EncodeToString(keyHash[:]),
		AESKey:       base64.StdEncoding.EncodeToString(aesKey),
		CreatedAt:    time.Now().Format("2006-01-02 15:04:05"),
		CreatorID:    "MW-" + a.GetPublicKey()[len(a.GetPublicKey())-8:],
		LocalPath:    filePath,
	}

	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := filepath.Join(getStorageDir(), fileCID+".meshweb")
	os.WriteFile(metaPath, metaBytes, 0644)

	// 6. Announce to DHT
	parsedCid, _ := cid.Decode(fileCID)
	go func() {
		a.logEvent("[Storage] Announcing CID to network...")
		if err := a.idht.Provide(a.ctx, parsedCid, true); err != nil {
			a.logEvent(fmt.Sprintf("[Storage] Announce error: %v", err))
		} else {
			a.logEvent(fmt.Sprintf("[Storage] File announced ✅: %s", fileCID))
		}
	}()

	return map[string]interface{}{
		"success": true,
		"fileId":  fileCID,
		"meta":    meta,
	}
}

func (a *App) GenerateShareLink(fileId string) string {
	metaPath := filepath.Join(getStorageDir(), fileId+".meshweb")
	b, err := os.ReadFile(metaPath)
	if err != nil {
		return ""
	}
	var meta MeshwebFile
	json.Unmarshal(b, &meta)

	// Link contains fileId, base64 AES key, original filename, and OriginalSize
	// so the downloader can trim RS padding correctly.
	return fmt.Sprintf("meshweb://file/%s?k=%s&n=%s&s=%d",
		fileId,
		meta.AESKey,
		base64.URLEncoding.EncodeToString([]byte(meta.FileName)),
		meta.OriginalSize,
	)
}

func (a *App) GenerateMeshwebFile(fileId string) map[string]interface{} {
	metaPath := filepath.Join(getStorageDir(), fileId+".meshweb")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return map[string]interface{}{"success": false, "error": "file not found"}
	}
	return map[string]interface{}{
		"success": true,
		"content": string(data),
	}
}

// Mahalliy yuklangan fayllar ro'yxati
func (a *App) GetMyFiles() []MeshwebFile {
	dir := getStorageDir()
	files, _ := os.ReadDir(dir)
	var list []MeshwebFile

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".meshweb") {
			data, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err == nil {
				var meta MeshwebFile
				json.Unmarshal(data, &meta)
				list = append(list, meta)
			}
		}
	}
	return list
}

func (a *App) DeleteFile(fileId string) {
	os.RemoveAll(filepath.Join(getStorageDir(), fileId))
	os.Remove(filepath.Join(getStorageDir(), fileId+".meshweb"))
}

func (a *App) DownloadFile(linkOrPath string) map[string]interface{} {
	var fileId string
	var aesKey []byte
	var meta MeshwebFile

	// 1. Link or Path parse
	if strings.HasPrefix(linkOrPath, "meshweb://file/") {
		// Parse: meshweb://file/<id>?k=<aesKey>&n=<base64FileName>
		without := strings.TrimPrefix(linkOrPath, "meshweb://file/")
		// Split on first '?'
		var queryStr string
		if idx := strings.Index(without, "?"); idx >= 0 {
			fileId = without[:idx]
			queryStr = without[idx+1:]
		} else {
			return map[string]interface{}{"success": false, "error": "invalid link format"}
		}

		// Parse query params manually
		params := make(map[string]string)
		for _, part := range strings.Split(queryStr, "&") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				params[kv[0]] = kv[1]
			}
		}

		// AES key
		aesKey, _ = base64.StdEncoding.DecodeString(params["k"])

		// Original filename (optional — old links don't have it)
		originalName := ""
		if nEncoded, ok := params["n"]; ok && nEncoded != "" {
			nameBytes, err := base64.URLEncoding.DecodeString(nEncoded)
			if err == nil {
				originalName = string(nameBytes)
			}
		}

		// Parse OriginalSize if present for RS trimming
		if sStr, ok := params["s"]; ok {
			fmt.Sscanf(sStr, "%d", &meta.OriginalSize)
		}

		// Build output filename — use original name as-is
		if originalName != "" {
			meta.FileName = originalName
		} else {
			// Fallback: use first 8 chars of fileId (no extension known)
			meta.FileName = fileId[:8]
		}
		meta.FileID = fileId
		meta.Shards = 30
	} else if strings.HasPrefix(linkOrPath, "{") {
		// It's raw JSON content of .meshweb
		json.Unmarshal([]byte(linkOrPath), &meta)
		fileId = meta.FileID
		aesKey, _ = base64.StdEncoding.DecodeString(meta.AESKey)
	} else {
		// It's a file path
		data, err := os.ReadFile(linkOrPath)
		if err != nil {
			return map[string]interface{}{"success": false, "error": "failed to read .meshweb file"}
		}
		json.Unmarshal(data, &meta)
		fileId = meta.FileID
		aesKey, _ = base64.StdEncoding.DecodeString(meta.AESKey)
	}

	a.logEvent(fmt.Sprintf("[Download] Finding providers for %s...", fileId[:8]))

	parsedCid, err := cid.Decode(fileId)
	if err != nil {
		return map[string]interface{}{"success": false, "error": "invalid file ID"}
	}

	// 2. Find Providers
	providers := a.idht.FindProvidersAsync(a.ctx, parsedCid, 5)
	
	var provider peer.AddrInfo
	select {
	case p, ok := <-providers:
		if !ok || p.ID == "" {
			return map[string]interface{}{"success": false, "error": "No seeders found"}
		}
		provider = p
	case <-time.After(15 * time.Second):
		return map[string]interface{}{"success": false, "error": "Timeout finding seeders"}
	}

	a.logEvent(fmt.Sprintf("[Download] Found seeder: %s. Fetching chunks...", provider.ID.String()[:8]))

	// 3. Connect to Provider
	err = a.node.Connect(a.ctx, provider)
	if err != nil {
		a.logEvent(fmt.Sprintf("[Download] Seeder ulanish xatosi: %v", err))
		// Try via relay? Let libp2p handle it
	}

	// 4. Download shards with Reed-Solomon recovery
	//    Try to fetch all rsTotalShards shards; mark missing ones as nil.
	//    RS can reconstruct the original as long as >= rsDataShards are present.
	enc, err := reedsolomon.New(rsDataShards, rsParityShards)
	if err != nil {
		return map[string]interface{}{"success": false, "error": "RS decoder init failed"}
	}

	totalShards := rsTotalShards
	if meta.Shards > 0 {
		totalShards = meta.Shards
	}

	shards := make([][]byte, totalShards)
	receivedCount := 0

	for i := 0; i < totalShards; i++ {
		stream, err := a.node.NewStream(a.ctx, provider.ID, StorageProtocolID)
		if err != nil {
			// Cannot open stream — mark shard as missing
			shards[i] = nil
			continue
		}

		req := ChunkRequest{FileID: fileId, Shard: i}
		b, _ := json.Marshal(req)
		b = append(b, '\n')
		stream.Write(b)

		scanner := bufio.NewScanner(stream)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 10*1024*1024)

		if !scanner.Scan() {
			stream.Close()
			shards[i] = nil
			continue
		}

		var res ChunkResponse
		json.Unmarshal(scanner.Bytes(), &res)
		stream.Close()

		if res.Error != "" || res.Data == "" {
			shards[i] = nil
			continue
		}

		chunkData, decErr := base64.StdEncoding.DecodeString(res.Data)
		if decErr != nil {
			shards[i] = nil
			continue
		}

		shards[i] = chunkData
		receivedCount++
		runtime.EventsEmit(a.ctx, "download-progress", float64(receivedCount)/float64(rsDataShards)*100.0)
	}

	if receivedCount < rsDataShards {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Not enough shards: got %d, need at least %d", receivedCount, rsDataShards),
		}
	}

	// Reconstruct missing shards using Reed-Solomon
	if err := enc.Reconstruct(shards); err != nil {
		return map[string]interface{}{"success": false, "error": fmt.Sprintf("RS reconstruct failed: %v", err)}
	}

	// Join only data shards (first rsDataShards) to rebuild ciphertext
	var ciphertext []byte
	for i := 0; i < rsDataShards; i++ {
		ciphertext = append(ciphertext, shards[i]...)
	}

	// Trim RS zero-padding: enc.Split() pads the last data shard so all
	// shards are equal length. OriginalSize tells us the true ciphertext end.
	if meta.OriginalSize > 0 && meta.OriginalSize < len(ciphertext) {
		ciphertext = ciphertext[:meta.OriginalSize]
	}

	a.logEvent("[Download] Shards reconstructed. Decrypting...")

	// 5. Decrypt
	plaintext, err := decryptData(ciphertext, aesKey)
	if err != nil {
		return map[string]interface{}{"success": false, "error": "Decryption failed (wrong key or corrupted data)"}
	}

	// 6. Resolve output path — use the original filename, auto-number duplicates.
	outName := meta.FileName
	outName = strings.TrimPrefix(outName, "downloaded_")
	if outName == "" {
		outName = fileId[:8]
	}
	outPath := uniqueFilePath(getDownloadsDir(), outName)
	err = os.WriteFile(outPath, plaintext, 0644)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}

	a.logEvent(fmt.Sprintf("[Download] Success ✅ Saved to %s", outPath))

	// Save to downloads.json
	df := DownloadedFile{
		FileName:     outName,
		FileID:       fileId,
		FileSize:     int64(len(plaintext)),
		LocalPath:    outPath,
		DownloadedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	a.saveDownloadedFile(df)

	return map[string]interface{}{
		"success": true,
		"path":    outPath,
	}
}

func (a *App) saveDownloadedFile(df DownloadedFile) {
	downloadsPath := filepath.Join(getStorageDir(), "downloads.json")
	var list []DownloadedFile
	b, err := os.ReadFile(downloadsPath)
	if err == nil {
		json.Unmarshal(b, &list)
	}
	list = append([]DownloadedFile{df}, list...) // push to top
	out, _ := json.MarshalIndent(list, "", "  ")
	os.WriteFile(downloadsPath, out, 0644)
}

func (a *App) GetDownloadedFiles() []DownloadedFile {
	downloadsPath := filepath.Join(getStorageDir(), "downloads.json")
	var list []DownloadedFile
	b, err := os.ReadFile(downloadsPath)
	if err == nil {
		json.Unmarshal(b, &list)
	}
	return list
}

func (a *App) OpenFile(filePath string) bool {
	// Windows only for now
	exec.Command("cmd", "/c", "start", "", filePath).Start()
	return true
}
