package sys

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
)

const (
	bannerDir    = "/etc/ssh_banners"
	profileScript = "/etc/profile.d/depwise_banner.sh"
)

// GenerateUserBanner genera el contenido del banner para un usuario SSH específico
func GenerateUserBanner(username, title string, limit int, expireDate string) string {
	if title == "" {
		title = "INTERNET ILIMITADO"
	}

	// Calcular días restantes
	daysLeft := 0
	parsed, err := time.Parse("2006-01-02", expireDate)
	if err == nil {
		daysLeft = int(math.Ceil(time.Until(parsed).Hours() / 24))
		if daysLeft < 0 {
			daysLeft = 0
		}
	}

	limitStr := fmt.Sprintf("%d", limit)
	if limit <= 0 {
		limitStr = "Ilimitado"
	}

	var b strings.Builder

	b.WriteString("\033[1;32m") // Verde brillante
	b.WriteString("═══════════════════════════════════════════\n")
	b.WriteString("      ____  _____ ______        _____ ____  _____ \n")
	b.WriteString("     |  _ \\| ____|  _ \\ \\      / /_ _/ ___|| ____|\n")
	b.WriteString("     | | | |  _| | |_) \\ \\ /\\ / / | |\\___ \\|  _|  \n")
	b.WriteString("     | |_| | |___|  __/ \\ V  V /  | | ___) | |___ \n")
	b.WriteString("     |____/|_____|_|     \\_/\\_/  |___|____/|_____|\n")
	b.WriteString("\033[0m") // Reset color
	b.WriteString("\033[1;36m") // Cyan brillante
	b.WriteString("═══════════════════════════════════════════\n")

	// Título centrado
	padding := (43 - len(title)) / 2
	if padding < 0 {
		padding = 0
	}
	b.WriteString(fmt.Sprintf("\033[1;35m%s⚡ %s ⚡\033[0m\n", strings.Repeat(" ", padding), title))

	b.WriteString("\033[1;36m═══════════════════════════════════════════\033[0m\n")

	// Datos de la cuenta
	b.WriteString(fmt.Sprintf("  \033[1;33m👤 Usuario:       \033[1;37m%s\033[0m\n", username))
	b.WriteString(fmt.Sprintf("  \033[1;33m📅 Vence:         \033[1;37m%s\033[0m\n", expireDate))
	b.WriteString(fmt.Sprintf("  \033[1;33m⏳ Días Restant.: \033[1;37m%d\033[0m\n", daysLeft))
	b.WriteString(fmt.Sprintf("  \033[1;33m💻 Límite:        \033[1;37m%s dispositivos\033[0m\n", limitStr))

	b.WriteString("\033[1;36m═══════════════════════════════════════════\033[0m\n")

	// Promoción
	b.WriteString("  \033[1;35m🔥 ¡SE VENDEN SERVIDORES PREMIUM! 🔥\033[0m\n")
	b.WriteString("  \033[1;33m📢 Canal:\033[0m  \033[1;36m@Depwise2\033[0m\n")
	b.WriteString("  \033[1;33m👤 Soporte:\033[0m \033[1;36m@Dan3651\033[0m\n")

	b.WriteString("\033[1;36m═══════════════════════════════════════════\033[0m\n")

	// Reglas
	b.WriteString("  \033[1;31m⚠️  REGLAS DEL SERVIDOR\033[0m\n")
	b.WriteString("  \033[0;37m🚫 NO Torrent / P2P\033[0m\n")
	b.WriteString("  \033[0;37m🚫 NO Spam / Fraude\033[0m\n")
	b.WriteString("  \033[0;37m🚫 NO Ataques DDoS\033[0m\n")
	b.WriteString("  \033[0;31m⛔ El incumplimiento genera ban automático\033[0m\n")

	b.WriteString("\033[1;36m═══════════════════════════════════════════\033[0m\n")
	b.WriteString("  \033[1;32m✅ CREADO CON: @Depwise_bot\033[0m\n")
	b.WriteString("\033[1;36m═══════════════════════════════════════════\033[0m\n\n")

	return b.String()
}

// WriteUserBanner genera y escribe el banner de un usuario en /etc/ssh_banners/
func WriteUserBanner(username, title string, limit int, expireDate string) error {
	// Crear directorio si no existe
	if err := os.MkdirAll(bannerDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio de banners: %v", err)
	}

	content := GenerateUserBanner(username, title, limit, expireDate)
	path := filepath.Join(bannerDir, username+".banner")
	return os.WriteFile(path, []byte(content), 0644)
}

// RemoveUserBanner elimina el banner de un usuario
func RemoveUserBanner(username string) error {
	path := filepath.Join(bannerDir, username+".banner")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // No existe, nada que hacer
	}
	return os.Remove(path)
}

// EnsureBannerSystem instala el script /etc/profile.d/depwise_banner.sh
// que muestra el banner correcto según el usuario que se conecta
func EnsureBannerSystem() error {
	script := `#!/bin/bash
# Banner dinámico Depwise — NO EDITAR (generado automáticamente)
BANNER_FILE="/etc/ssh_banners/$(whoami).banner"
if [ -f "$BANNER_FILE" ]; then
    cat "$BANNER_FILE"
fi
`
	if err := os.MkdirAll(bannerDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(profileScript, []byte(script), 0755); err != nil {
		return err
	}

	// Asegurar permisos de ejecución
	exec.Command("chmod", "+x", profileScript).Run()

	return nil
}

// RefreshAllBanners regenera los banners de todos los usuarios SSH activos
// para actualizar los días restantes
func RefreshAllBanners() {
	data, err := db.Load()
	if err != nil {
		return
	}

	// Solo regenerar si el directorio de banners existe (sistema activo)
	if _, err := os.Stat(bannerDir); os.IsNotExist(err) {
		return
	}

	for user, expire := range data.SSHTimeUsers {
		title := ""
		if data.SSHBannerTitles != nil {
			title = data.SSHBannerTitles[user]
		}
		limit := GetUserMaxLogins(user)
		WriteUserBanner(user, title, limit, expire)
	}
}
