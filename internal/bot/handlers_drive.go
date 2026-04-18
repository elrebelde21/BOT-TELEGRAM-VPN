package bot

import (
	"fmt"
	"os"
	"strings"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/drive"
	tele "gopkg.in/telebot.v3"
)

func notifyNotConfigured(c tele.Context) error {
	return c.Respond(&tele.CallbackResponse{
		Text:      "⚠️ Google Drive inactivo.\n\nDescarga tu clave JSON de Cuenta de Servicio y envíamela como documento a este chat para habilitar los Backups.",
		ShowAlert: true,
	})
}

func handleDriveBackup(c tele.Context, b *tele.Bot) error {
	if !drive.IsAuthenticated() {
		return notifyNotConfigured(c)
	}
	
	SafeEditCtx(c, b, "⏳ <i>Subiendo copia de seguridad a Google Drive...\nEl proceso tomará unos segundos.</i>", nil)
	
	err := drive.UploadBackup(db.GetDataPath())
	
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver a Ajustes Pro", "menu_admins")))
	
	if err != nil {
		if strings.Contains(err.Error(), "credentials.json no existe") {
			return SafeEditCtx(c, b, "❌ <b>Credenciales Ausentes:</b>\nPor favor, vuelve a enviar el archivo JSON al bot.", markup)
		}
		return SafeEditCtx(c, b, fmt.Sprintf("❌ <b>Error al subir backup:</b>\n%v", err), markup)
	}
	
	return SafeEditCtx(c, b, "✅ <b>Copia de Seguridad subida exitosamente a Drive.</b>\nSe encuentra en la carpeta BotVPN_Backups.", markup)
}

func handleDriveRestore(c tele.Context, b *tele.Bot) error {
	if !drive.IsAuthenticated() {
		return notifyNotConfigured(c)
	}
	
	SafeEditCtx(c, b, "📥 <i>Descargando y aplicando copia de seguridad de Google Drive...</i>", nil)
	
	err := drive.RestoreBackup(db.GetDataPath())
	
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver al Inicio", "menu_admins")))
	
	if err != nil {
		return SafeEditCtx(c, b, fmt.Sprintf("❌ <b>Error al restaurar:</b>\n%v", err), markup)
	}
	
	return SafeEditCtx(c, b, "✅ <b>Base de Datos Restaurada Exitosamente!</b>\nLos IDs de usuario y configuraciones se han cargado.\n\n⚠️ <i>Te recomiendo presionar 'Reiniciar VPS' en Ajustes Pro para aplicar configuraciones.</i>", markup)
}

func handleDocumentUploads(c tele.Context, b *tele.Bot) error {
	if !isAdmin(c.Chat().ID) {
		return nil 
	}

	doc := c.Message().Document
	if doc == nil || !strings.HasSuffix(doc.FileName, ".json") {
		return nil 
	}

	c.Send("⏳ <i>Descargando y verificando archivo de credenciales de Google Drive...</i>", tele.ModeHTML)

	// Descargar temporalmente
	tempPath := "temp_creds.json"
	
	file := c.Message().Document.File
	if err := b.Download(&file, tempPath); err != nil {
		return c.Send(fmt.Sprintf("❌ Error al descargar el documento: %v", err))
	}

	// Verificar si es valido
	err := drive.CheckCredentialsFile(tempPath)
	if err != nil {
		os.Remove(tempPath)
		return c.Send(fmt.Sprintf("❌ <b>El archivo NO es válido para Google Drive:</b>\n%v\n\n<i>Asegúrate de enviar el JSON de una 'Cuenta de Servicio' habilitada.</i>", err), tele.ModeHTML)
	}

	// Sobreescribir viejo credentials si hace falta
	os.Remove(drive.CredentialsFile)
	
	err = os.Rename(tempPath, drive.CredentialsFile)
	if err != nil {
		os.Remove(tempPath)
		return c.Send(fmt.Sprintf("❌ Error interno moviendo el archivo: %v", err))
	}

	return c.Send("✅ <b>¡Google Drive Vinculado Exitosamente!</b>\n\nEl sistema de copias de seguridad automáticas y el menú de restauración están listos para usarse de por vida.", tele.ModeHTML)
}
