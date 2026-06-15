package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

func (a *App) RegisterFileAssociation() map[string]interface{} {
	exePath, err := os.Executable()
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}

	extKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\.meshweb`, registry.ALL_ACCESS)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}
	defer extKey.Close()

	err = extKey.SetStringValue("", "MeshwebFile")
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}

	meshwebKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\MeshwebFile`, registry.ALL_ACCESS)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}
	defer meshwebKey.Close()

	err = meshwebKey.SetStringValue("", "Meshweb Shared File")
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}

	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\MeshwebFile\shell\open\command`, registry.ALL_ACCESS)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}
	defer cmdKey.Close()

	command := fmt.Sprintf(`"%s" "%%1"`, exePath)
	err = cmdKey.SetStringValue("", command)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}

	return map[string]interface{}{"success": true}
}
