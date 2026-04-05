package sys

import (
	"os/exec"
	"strings"
	"time"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/vpn"
	tele "gopkg.in/telebot.v3"
)

// CountZivpnActive returns true if any UDP session exists for zivpn
func CountZivpnActive() bool {
	out, err := exec.Command("sh", "-c", "ss -u -n -p | grep 'zivpn' | wc -l").Output()
	if err != nil {
		return false
	}
	count := strings.TrimSpace(string(out))
	return count != "" && count != "0"
}

// AutoCleanupLoop corre en un hilo separado ejecutando la limpieza de Iptables
// y usuarios excedidos cada cierto tiempo.
func AutoCleanupLoop(b *tele.Bot) {

	tick := 0
	for {
		// Revisar límites de conexión activa cada 14 segundos (2 ticks)
		if tick%2 == 0 {
			EnforceConnectionLimits()
		}

		// 1. Limpieza de usuarios vencidos de forma periódica
		if tick >= 9 { // Cada 60-70 segundos aprox
			// Guardar el tráfico en DB para que persista tras reiniciar la VPS
			GetGlobalTraffic()

			db.Update(func(data *db.ConfigData) error {
				now := time.Now().Format("2006-01-02")

				// Revisar SSH
				for user, expire := range data.SSHTimeUsers {
					if now > expire {
						DeleteSSHUser(user)
						delete(data.SSHTimeUsers, user)
						delete(data.SSHOwners, user)
						delete(data.SSHLastActive, user)
					}
				}

				// Revisar ZiVPN - auto-expiración por fecha
				for pass, expire := range data.ZivpnUsers {
					if now > expire {
						vpn.RemoveZivpnUser(pass)
						delete(data.ZivpnUsers, pass)
						delete(data.ZivpnOwners, pass)
						delete(data.ZivpnLastActive, pass)
					}
				}

				return nil
			})

			// Nueva Ejecución: Limpieza cada 60s terminada
			tick = 0
		}

		// Ejecución Crítica: Cuotas y Conexiones (Más frecuente)
		if tick%2 == 0 {
			EnforceConnectionLimits()
		}

		// EnforceConnectionLimits revisa las conexiones activas cada tick
		EnforceConnectionLimits()

		tick++
		time.Sleep(7 * time.Second)
	}
}
