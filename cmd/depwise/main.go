package main

import (
	"log"
	"time"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/bot"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/drive"
)

func main() {
	log.Println("Iniciando Depwise SSH VPN Manager...")

	// Hilo de backup de 24 hrs
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if drive.IsAuthenticated() {
				log.Println("Iniciando backup automático programado...")
				err := drive.UploadBackup(db.GetDataPath())
				if err != nil {
					log.Printf("Error en backup automático: %v\n", err)
				} else {
					log.Println("Backup automático subido a Drive.")
				}
			}
		}
	}()

	// Iniciar servidor del bot (bloqueante)
	bot.StartBot()
}

