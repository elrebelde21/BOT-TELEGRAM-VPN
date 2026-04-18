package bot

import (
	"fmt"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/drive"
	tele "gopkg.in/telebot.v3"
)

func handleAuthDrive(c tele.Context, b *tele.Bot) error {
	if !isAdmin(c.Chat().ID) {
		return c.Send("⛔ Solo administradores pueden usar este comando.", tele.ModeHTML)
	}

	args := c.Args()
	if len(args) == 0 {
		url, err := drive.GetAuthURL()
		if err != nil {
			return c.Send(fmt.Sprintf("❌ Error obteniendo URL de Google:\n%v\n\n¿Has puesto credentials.json en la carpeta?", err))
		}
		texto := "🔐 <b>Autorización Google Drive</b>\n\n" +
			"1. Abre este enlace: <a href='" + url + "'>Click Aquí</a>\n" +
			"2. Inicia sesión con tu cuenta de Google.\n" +
			"3. Copia el código que te da la pantalla.\n" +
			"4. Envíamelo usando el comando:\n" +
			"<code>/authdrive TU_CODIGO_AQUI</code>"
		return c.Send(texto, tele.ModeHTML)
	}

	code := args[0]
	err := drive.SaveToken(code)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Error guardando el Token:\n%v", err))
	}

	return c.Send("✅ <b>¡Identificación con Google correcta!</b>\n\nAhora puedes usar los botones de Backup y Restaurar en el menú de Ajustes Pro.", tele.ModeHTML)
}

func handleDriveBackup(c tele.Context, b *tele.Bot) error {
	if !drive.IsAuthenticated() {
		return c.Respond(&tele.CallbackResponse{Text: "⚠️ Primero vincula tu cuenta usando /authdrive", ShowAlert: true})
	}
	
	SafeEditCtx(c, b, "⏳ <i>Subiendo copia de seguridad a Google Drive...\nEl proceso tomará unos segundos.</i>", nil)
	
	err := drive.UploadBackup(db.GetDataPath())
	
	// Restauramos el menú para que no se quede bloqueado
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver a Ajustes Pro", "menu_admins")))
	
	if err != nil {
		return SafeEditCtx(c, b, fmt.Sprintf("❌ <b>Error al subir backup:</b>\n%v", err), markup)
	}
	
	return SafeEditCtx(c, b, "✅ <b>Copia de Seguridad subida exitosamente a Drive.</b>\nSe encuentra en la carpeta BotVPN_Backups.", markup)
}

func handleDriveRestore(c tele.Context, b *tele.Bot) error {
	if !drive.IsAuthenticated() {
		return c.Respond(&tele.CallbackResponse{Text: "⚠️ Primero vincula tu cuenta usando /authdrive", ShowAlert: true})
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
