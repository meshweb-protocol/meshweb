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
	Version      string        `json:"version"`
	Type         string        `json:"type,omitempty"` // "folder" or empty for file
	FileName     string        `json:"file_name"`
	RelativePath string        `json:"relative_path,omitempty"` // Added for proper nested folder reconstruction
	FileSize     int64         `json:"file_size"`
	OriginalSize int           `json:"original_size"` // len(ciphertext) before RS padding
	FileID       string        `json:"file_id"`
	Shards       int           `json:"shards"`
	MinShards    int           `json:"min_shards"`
	Encryption   string        `json:"encryption"`
	KeyHash      string        `json:"key_hash"`
	AESKey       string        `json:"aes_key,omitempty"`
	CreatedAt    string        `json:"created_at"`
	CreatorID    string        `json:"creator_id"`
	LocalPath    string        `json:"local_path,omitempty"`
	Files        []MeshwebFile `json:"files,omitempty"` // for folders
}

type DownloadedFile struct {
	FileName     string        `json:"file_name"`
	FileID       string        `json:"file_id"`
	FileSize     int64         `json:"file_size"`
	LocalPath    string        `json:"local_path"`
	DownloadedAt string        `json:"downloaded_at"`
	Type         string        `json:"type,omitempty"`
	Files        []MeshwebFile `json:"files,omitempty"`
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

type StoreRequest struct {
	Action string `json:"action"` // "fetch" or "store"
	FileID string `json:"file_id"`
	Shard  int    `json:"shard"`
	Data   string `json:"data,omitempty"` // base64, only for store
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
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 10*1024*1024)
		if !scanner.Scan() {
			return
		}

		// Try to parse as StoreRequest first (has "action" field)
		var storeReq StoreRequest
		if err := json.Unmarshal(scanner.Bytes(), &storeReq); err != nil {
			return
		}

		if storeReq.Action == "store" {
			// Incoming shard to store
			shardData, err := base64.StdEncoding.DecodeString(storeReq.Data)
			if err != nil {
				res := ChunkResponse{FileID: storeReq.FileID, Shard: storeReq.Shard, Error: "decode error"}
				b, _ := json.Marshal(res)
				b = append(b, '\n')
				s.Write(b)
				return
			}

			fileDir := filepath.Join(getStorageDir(), storeReq.FileID)
			os.MkdirAll(fileDir, 0755)
			shardPath := filepath.Join(fileDir, fmt.Sprintf("shard_%d", storeReq.Shard))
			os.WriteFile(shardPath, shardData, 0644)

			a.logEvent(fmt.Sprintf("[Storage] Stored shard %d for %s from %s", storeReq.Shard, storeReq.FileID[:8], s.Conn().RemotePeer().String()[:8]))

			// Announce as provider for this CID
			go func() {
				parsedCid, err := cid.Decode(storeReq.FileID)
				if err == nil {
					a.idht.Provide(a.ctx, parsedCid, true)
				}
			}()

			res := ChunkResponse{FileID: storeReq.FileID, Shard: storeReq.Shard}
			b, _ := json.Marshal(res)
			b = append(b, '\n')
			s.Write(b)
			return
		}

		// Default: fetch shard (backward compatible with old ChunkRequest)
		shardPath := filepath.Join(getStorageDir(), storeReq.FileID, fmt.Sprintf("shard_%d", storeReq.Shard))
		data, err := os.ReadFile(shardPath)

		res := ChunkResponse{FileID: storeReq.FileID, Shard: storeReq.Shard}
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

func (a *App) distributeShardsToNetwork(fileID string, shards [][]byte) {
	const relayPeerID = "12D3KooWRdvwz59ErP1e6pxqxpYY6rNFTYPCDYGK8eoH9cd4obfq"

	peers := a.node.Network().Peers()
	if len(peers) == 0 {
		a.logEvent("[Storage] No peers available for shard distribution")
		return
	}

	// Filter out relay and self
	var validPeers []peer.ID
	for _, p := range peers {
		if p.String() != a.myPeerID && p.String() != relayPeerID {
			validPeers = append(validPeers, p)
		}
	}

	if len(validPeers) == 0 {
		a.logEvent("[Storage] No valid peers for distribution")
		return
	}

	a.logEvent(fmt.Sprintf("[Storage] Distributing %d shards to %d peers...", len(shards), len(validPeers)))

	distributed := 0
	for i, shard := range shards {
		targetPeer := validPeers[i%len(validPeers)]

		go func(shardIdx int, shardData []byte, target peer.ID) {
			stream, err := a.node.NewStream(a.ctx, target, StorageProtocolID)
			if err != nil {
				return
			}
			defer stream.Close()

			req := StoreRequest{
				Action: "store",
				FileID: fileID,
				Shard:  shardIdx,
				Data:   base64.StdEncoding.EncodeToString(shardData),
			}
			b, _ := json.Marshal(req)
			b = append(b, '\n')
			stream.Write(b)

			// Wait for ack
			scanner := bufio.NewScanner(stream)
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 10*1024*1024)
			if scanner.Scan() {
				var res ChunkResponse
				json.Unmarshal(scanner.Bytes(), &res)
				if res.Error == "" {
					a.logEvent(fmt.Sprintf("[Storage] Shard %d → %s ✅", shardIdx, target.String()[:8]))
				}
			}
		}(i, shard, targetPeer)

		distributed++
	}

	a.logEvent(fmt.Sprintf("[Storage] Distribution started: %d shards", distributed))
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

	return a.UploadData(data, filepath.Base(filePath), filePath)
}

func (a *App) UploadData(data []byte, fileName string, localPath string) map[string]interface{} {
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
		FileName:     fileName,
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
		LocalPath:    localPath,
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

	// 7. Distribute shards to network peers
	go a.distributeShardsToNetwork(fileCID, shards)

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

func (a *App) removeRecursive(folder MeshwebFile, fileId string) (bool, MeshwebFile) {
	removedAny := false
	var newFiles []MeshwebFile
	var newSize int64

	for _, f := range folder.Files {
		if f.FileID == fileId {
			removedAny = true
			continue
		}
		if f.Type == "folder" {
			removedSub, newSubFolder := a.removeRecursive(f, fileId)
			if removedSub {
				removedAny = true
				f = newSubFolder
			}
		}
		newFiles = append(newFiles, f)
		newSize += f.FileSize
	}

	if removedAny {
		folder.Files = newFiles
		folder.FileSize = newSize
	}
	return removedAny, folder
}

func (a *App) removeInnerFileFromFolders(fileId string) {
	files, err := os.ReadDir(getStorageDir())
	if err != nil {
		return
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".meshweb") {
			metaPath := filepath.Join(getStorageDir(), f.Name())
			data, err := os.ReadFile(metaPath)
			if err == nil {
				var meta MeshwebFile
				json.Unmarshal(data, &meta)
				if meta.Type == "folder" {
					if removed, newMeta := a.removeRecursive(meta, fileId); removed {
						b, _ := json.MarshalIndent(newMeta, "", "  ")
						os.WriteFile(metaPath, b, 0644)
					}
				}
			}
		}
	}
}

func (a *App) DeleteFile(fileId string) {
	os.RemoveAll(filepath.Join(getStorageDir(), fileId))
	err := os.Remove(filepath.Join(getStorageDir(), fileId+".meshweb"))
	if err != nil {
		a.removeInnerFileFromFolders(fileId)
	} else {
		a.removeInnerFileFromFolders(fileId)
	}
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

	plaintext, err := a.fetchAndDecrypt(fileId, aesKey, meta.OriginalSize)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}

	var folderMeta MeshwebFile
	if err := json.Unmarshal(plaintext, &folderMeta); err == nil && folderMeta.Type == "folder" {
		folderName := folderMeta.FileName
		folderName = strings.TrimSuffix(folderName, ".folder.json")
		if folderName == "" {
			folderName = fileId[:8]
		}
		outPath := uniqueFilePath(getDownloadsDir(), folderName)
		os.MkdirAll(outPath, 0755)

		err = a.downloadFolderRecursive(folderMeta, outPath)
		if err != nil {
			return map[string]interface{}{"success": false, "error": err.Error()}
		}

		a.logEvent(fmt.Sprintf("[Download] Folder Success ✅ Saved to %s", outPath))

		df := DownloadedFile{
			FileName:     folderName,
			FileID:       fileId,
			FileSize:     folderMeta.FileSize,
			LocalPath:    outPath,
			DownloadedAt: time.Now().Format("2006-01-02 15:04:05"),
			Type:         "folder",
			Files:        folderMeta.Files,
		}
		a.saveDownloadedFile(df)

		return map[string]interface{}{
			"success": true,
			"path":    outPath,
		}
	}

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

func (a *App) fetchAndDecrypt(fileId string, aesKey []byte, originalSize int) ([]byte, error) {
	parsedCid, err := cid.Decode(fileId)
	if err != nil {
		return nil, fmt.Errorf("invalid file ID")
	}

	providersChan := a.idht.FindProvidersAsync(a.ctx, parsedCid, 20)
	var providerList []peer.AddrInfo
	timeout := time.After(15 * time.Second)
collecting:
	for {
		select {
		case p, ok := <-providersChan:
			if !ok {
				break collecting
			}
			if p.ID != "" && p.ID != a.node.ID() {
				providerList = append(providerList, p)
			}
		case <-timeout:
			break collecting
		}
	}

	for _, p := range a.node.Network().Peers() {
		alreadyIn := false
		for _, existing := range providerList {
			if existing.ID == p {
				alreadyIn = true
				break
			}
		}
		if !alreadyIn && p != a.node.ID() {
			providerList = append(providerList, peer.AddrInfo{ID: p})
		}
	}

	if len(providerList) == 0 {
		return nil, fmt.Errorf("No seeders found")
	}

	for _, p := range providerList {
		a.node.Connect(a.ctx, p)
	}

	enc, err := reedsolomon.New(rsDataShards, rsParityShards)
	if err != nil {
		return nil, fmt.Errorf("RS decoder init failed")
	}

	totalShards := rsTotalShards
	shards := make([][]byte, totalShards)
	receivedCount := 0

	for i := 0; i < totalShards && receivedCount < rsDataShards; i++ {
		for _, p := range providerList {
			stream, err := a.node.NewStream(a.ctx, p.ID, StorageProtocolID)
			if err != nil {
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
				continue
			}

			var res ChunkResponse
			json.Unmarshal(scanner.Bytes(), &res)
			stream.Close()

			if res.Error != "" || res.Data == "" {
				continue
			}

			chunkData, decErr := base64.StdEncoding.DecodeString(res.Data)
			if decErr != nil {
				continue
			}

			shards[i] = chunkData
			receivedCount++
			
			// Main progress event
			progress := float64(receivedCount) / float64(rsDataShards) * 100.0
			if progress > 100 {
				progress = 100
			}
			runtime.EventsEmit(a.ctx, "download-progress", progress)
			break
		}
	}

	if receivedCount < rsDataShards {
		return nil, fmt.Errorf("Not enough shards: got %d, need at least %d", receivedCount, rsDataShards)
	}

	if err := enc.Reconstruct(shards); err != nil {
		return nil, fmt.Errorf("RS reconstruct failed: %v", err)
	}

	var ciphertext []byte
	for i := 0; i < rsDataShards; i++ {
		ciphertext = append(ciphertext, shards[i]...)
	}

	if originalSize > 0 && originalSize < len(ciphertext) {
		ciphertext = ciphertext[:originalSize]
	}

	plaintext, err := decryptData(ciphertext, aesKey)
	if err != nil {
		return nil, fmt.Errorf("Decryption failed (wrong key or corrupted data)")
	}

	return plaintext, nil
}

func (a *App) downloadFolderRecursive(folder MeshwebFile, basePath string) error {
	for _, child := range folder.Files {
		if child.Type == "folder" {
			var subPath string
			if child.RelativePath != "" {
				relPath := filepath.FromSlash(child.RelativePath)
				subPath = filepath.Join(basePath, relPath)
			} else {
				subPath = filepath.Join(basePath, child.FileName)
			}
			os.MkdirAll(subPath, 0755)
			if err := a.downloadFolderRecursive(child, basePath); err != nil {
				a.logEvent(fmt.Sprintf("[Download] Error in subfolder %s: %v", child.FileName, err))
			}
		} else {
			a.logEvent(fmt.Sprintf("[Download] Fetching file %s...", child.FileName))
			
			var outPath string
			if child.RelativePath != "" {
				relPath := filepath.FromSlash(child.RelativePath)
				outPath = filepath.Join(basePath, relPath)
				os.MkdirAll(filepath.Dir(outPath), 0755)
			} else {
				outPath = filepath.Join(basePath, child.FileName)
			}

			aesKeyBytes, _ := base64.StdEncoding.DecodeString(child.AESKey)
			plaintext, err := a.fetchAndDecrypt(child.FileID, aesKeyBytes, child.OriginalSize)
			if err != nil {
				a.logEvent(fmt.Sprintf("[Download] Error fetching %s: %v", child.FileName, err))
				continue
			}
			
			os.WriteFile(outPath, plaintext, 0644)
			
			if child.RelativePath != "" {
				a.logEvent(fmt.Sprintf("[Download] ✅ %s", child.RelativePath))
			} else {
				a.logEvent(fmt.Sprintf("[Download] ✅ %s", child.FileName))
			}
		}
	}
	return nil
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
