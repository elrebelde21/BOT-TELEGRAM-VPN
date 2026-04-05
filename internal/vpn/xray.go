package vpn

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

const xrayConfigPath = "/usr/local/etc/xray/config.json"

// InstallXray instala el núcleo de Xray y configura el archivo JSON inicial
func InstallXray() error {
	// 1. Descargar e instalar Xray desde el script oficial de GitHub
	cmd := exec.Command("bash", "-c", "bash -c \"$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)\" @ install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falló la instalación de xray core: %v", err)
	}

	// 2. Crear configuración base de VMess WS
	baseConfig := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "warning",
		},
		"inbounds": []map[string]interface{}{
			{
				"port":     10002, // Puerto local fijo para enlazar con HAProxy
				"listen":   "127.0.0.1",
				"protocol": "vmess",
				"settings": map[string]interface{}{
					"clients": []map[string]interface{}{},
				},
				"streamSettings": map[string]interface{}{
					"network": "ws",
					"wsSettings": map[string]interface{}{
						"path": "/vmess",
					},
				},
				"sniffing": map[string]interface{}{
					"enabled": true,
					"destOverride": []string{"http", "tls"},
				},
			},
		},
		"outbounds": []map[string]interface{}{
			{
				"protocol": "freedom",
				"tag":      "direct",
			},
			{
				"protocol": "blackhole",
				"tag":      "block",
			},
		},
	}

	raw, err := json.MarshalIndent(baseConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("error generando JSON base: %v", err)
	}

	if err := os.WriteFile(xrayConfigPath, raw, 0644); err != nil {
		return fmt.Errorf("error escribiendo config.json de xray: %v", err)
	}

	if err := exec.Command("systemctl", "restart", "xray").Run(); err != nil {
		return fmt.Errorf("error reiniciando xray.service: %v", err)
	}

	// Asegurarse de que arranque al iniciar el sistema
	exec.Command("systemctl", "enable", "xray").Run()

	return nil
}

// RemoveXray detiene y borra el núcleo
func RemoveXray() error {
	exec.Command("systemctl", "stop", "xray").Run()
	exec.Command("systemctl", "disable", "xray").Run()
	exec.Command("bash", "-c", "bash -c \"$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)\" @ remove").Run()
	os.RemoveAll("/usr/local/etc/xray")
	return nil
}

// loadXrayConfig lee la config JSON existente
func loadXrayConfig() (map[string]interface{}, error) {
	raw, err := os.ReadFile(xrayConfigPath)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	err = json.Unmarshal(raw, &data)
	return data, err
}

// saveXrayConfig escribe la config JSON al sistema y reinicia el demonio
func saveXrayConfig(data map[string]interface{}) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(xrayConfigPath, raw, 0644); err != nil {
		return err
	}
	// Reinicio silencioso
	return exec.Command("systemctl", "restart", "xray").Run()
}

// AddXrayUser inyecta el nuevo usuario VMess al archivo y reinicia el core
func AddXrayUser(uuid, email string) error {
	cfg, err := loadXrayConfig()
	if err != nil {
		return err
	}

	inbounds, ok := cfg["inbounds"].([]interface{})
	if !ok || len(inbounds) == 0 {
		return fmt.Errorf("formato inbounds inválido en config.json")
	}

	inbound0 := inbounds[0].(map[string]interface{})
	settings := inbound0["settings"].(map[string]interface{})
	
	var clients []interface{}
	if settings["clients"] != nil {
		clients = settings["clients"].([]interface{})
	}

	newUser := map[string]interface{}{
		"id":    uuid,
		"level": 0,
		"email": email, // Guardamos el alias o chatid para identificarlo
	}
	clients = append(clients, newUser)
	settings["clients"] = clients

	return saveXrayConfig(cfg)
}

// RemoveXrayUser busca el UUID y lo elimina de la lista de clientes.
func RemoveXrayUser(uuid string) error {
	cfg, err := loadXrayConfig()
	if err != nil {
		return err
	}

	inbounds, ok := cfg["inbounds"].([]interface{})
	if !ok || len(inbounds) == 0 {
		return fmt.Errorf("formato inbounds inválido en config.json")
	}

	inbound0 := inbounds[0].(map[string]interface{})
	settings := inbound0["settings"].(map[string]interface{})
	
	if settings["clients"] == nil {
		return nil // no hay clientes
	}
	clients := settings["clients"].([]interface{})

	var newClients []interface{}
	for _, c := range clients {
		clientMap := c.(map[string]interface{})
		if clientMap["id"] != uuid {
			newClients = append(newClients, c)
		}
	}
	settings["clients"] = newClients

	return saveXrayConfig(cfg)
}

// GenerateVmessLink crea el texto base64 para importar el perfil en v2rayNG / HTTP Custom
func GenerateVmessLink(alias, uuid, domain string) string {
	vmessObj := map[string]interface{}{
		"v":    "2",
		"ps":   alias,
		"add":  domain,
		"port": "443", // Puerto SSL Tunnel (HAProxy)
		"id":   uuid,
		"aid":  "0",
		"scy":  "auto",
		"net":  "ws",
		"type": "none",
		"host": domain,
		"path": "/vmess",
		"tls":  "tls",
		"sni":  domain,
		"alpn": "",
	}
	
	raw, _ := json.Marshal(vmessObj)
	encoded := base64.StdEncoding.EncodeToString(raw)
	return "vmess://" + encoded
}
