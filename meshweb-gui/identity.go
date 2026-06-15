package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/tyler-smith/go-bip39"
)

type IdentityStorage struct {
	SeedPhrase string `json:"seed_phrase"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

func getIdentityFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	appDir := filepath.Join(configDir, "meshweb-gui")
	os.MkdirAll(appDir, 0755)
	return filepath.Join(appDir, "identity.key")
}

// Kalit yaratish
func (a *App) GenerateIdentity() map[string]interface{} {
	entropy, _ := bip39.NewEntropy(128) // 12 words
	mnemonic, _ := bip39.NewMnemonic(entropy)

	success := a.RestoreIdentity(mnemonic)
	if !success {
		return map[string]interface{}{"success": false, "error": "Failed to derive key from mnemonic"}
	}

	return map[string]interface{}{
		"success":    true,
		"seedPhrase": mnemonic,
	}
}

// Seed phrase dan tiklash
func (a *App) RestoreIdentity(seedPhrase string) bool {
	if !bip39.IsMnemonicValid(seedPhrase) {
		return false
	}

	seed := bip39.NewSeed(seedPhrase, "")
	ed25519Seed := seed[:32]

	priv, pub, err := crypto.GenerateEd25519Key(bytes.NewReader(ed25519Seed))
	if err != nil {
		return false
	}

	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return false
	}

	pubBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return false
	}

	storage := IdentityStorage{
		SeedPhrase: seedPhrase,
		PrivateKey: base64.StdEncoding.EncodeToString(privBytes),
		PublicKey:  base64.StdEncoding.EncodeToString(pubBytes),
	}

	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return false
	}

	err = os.WriteFile(getIdentityFilePath(), data, 0600)
	if err != nil {
		return false
	}

	a.privKey = priv
	return true
}

// Kalit yuklash
func (a *App) LoadIdentity() bool {
	data, err := os.ReadFile(getIdentityFilePath())
	if err != nil {
		return false
	}

	var storage IdentityStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return false
	}

	privBytes, err := base64.StdEncoding.DecodeString(storage.PrivateKey)
	if err != nil {
		return false
	}

	priv, err := crypto.UnmarshalPrivateKey(privBytes)
	if err != nil {
		return false
	}

	a.privKey = priv
	return true
}

// Public key olish
func (a *App) GetPublicKey() string {
	if a.privKey == nil {
		return ""
	}
	id, err := peer.IDFromPrivateKey(a.privKey)
	if err != nil {
		return ""
	}
	return id.String()
}

// Identity export
func (a *App) ExportIdentity() map[string]interface{} {
	data, err := os.ReadFile(getIdentityFilePath())
	if err != nil {
		return map[string]interface{}{"success": false}
	}

	var storage IdentityStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return map[string]interface{}{"success": false}
	}

	return map[string]interface{}{
		"success":    true,
		"seedPhrase": storage.SeedPhrase,
		"privateKey": storage.PrivateKey,
		"publicKey":  storage.PublicKey,
	}
}
