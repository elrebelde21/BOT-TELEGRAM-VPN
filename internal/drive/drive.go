package drive

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	CredentialsFile = "credentials.json"
	FolderName      = "BotVPN_Backups"
	MaxBackups      = 2
)

func getService() (*drive.Service, error) {
	if _, err := os.Stat(CredentialsFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("credentials.json no existe")
	}
	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithCredentialsFile(CredentialsFile), option.WithScopes(drive.DriveFileScope))
	if err != nil {
		return nil, err
	}
	return srv, nil
}

// IsAuthenticated verifica de forma ligera si el archivo existe y es válido
func IsAuthenticated() bool {
	if _, err := os.Stat(CredentialsFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// CheckCredentialsFile prueba un JSON local para asegurar que es válido antes de aplicarlo
func CheckCredentialsFile(path string) error {
	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithCredentialsFile(path), option.WithScopes(drive.DriveFileScope))
	if err != nil {
		return fmt.Errorf("el archivo JSON no es una clave API válida: %v", err)
	}

	// Ping pequeño para ver si autentica de verdad
	_, err = srv.Files.List().PageSize(1).Do()
	if err != nil {
		return fmt.Errorf("el JSON es válido, pero la API de Drive no está activa en tu proyecto: %v", err)
	}

	return nil
}

// UploadBackup toma el un archivo local y lo sube al Drive
func UploadBackup(filePath string) error {
	srv, err := getService()
	if err != nil {
		return err
	}
	
	folderId, err := ensureFolder(srv)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("bot_data_%s.json", time.Now().Format("20060102_150405"))
	
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("imposible leer archivo a respaldar: %v", err)
	}
	defer f.Close()

	driveFile := &drive.File{
		Name:     fileName,
		Parents:  []string{folderId},
	}
	
	_, err = srv.Files.Create(driveFile).Media(f).Do()
	if err != nil {
		return fmt.Errorf("error subiendo archivo a drive: %v", err)
	}

	cleanupOldBackups(srv, folderId)
	return nil
}

// RestoreBackup descarga la ultima versión y reemplaza el archivo local
func RestoreBackup(filePath string) error {
	srv, err := getService()
	if err != nil {
		return err
	}
	folderId, err := ensureFolder(srv)
	if err != nil {
		return err
	}
	
	q := fmt.Sprintf("'%s' in parents and trashed=false", folderId)
	r, err := srv.Files.List().Q(q).OrderBy("createdTime desc").Fields("files(id, name, createdTime)").Do()
	if err != nil {
		return err
	}
	
	if len(r.Files) == 0 {
		return fmt.Errorf("no se encontraron copias de seguridad en Drive")
	}
	
	latest := r.Files[0]
	resp, err := srv.Files.Get(latest.Id).Download()
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, filePath)
}

func ensureFolder(srv *drive.Service) (string, error) {
	q := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and trashed=false", FolderName)
	r, err := srv.Files.List().Q(q).Fields("files(id, name)").Do()
	if err != nil {
		return "", err
	}
	if len(r.Files) > 0 {
		return r.Files[0].Id, nil
	}
	
	f := &drive.File{
		Name:     FolderName,
		MimeType: "application/vnd.google-apps.folder",
	}
	folder, err := srv.Files.Create(f).Do()
	if err != nil {
		return "", err
	}
	return folder.Id, nil
}

func cleanupOldBackups(srv *drive.Service, folderId string) {
	q := fmt.Sprintf("'%s' in parents and trashed=false", folderId)
	r, err := srv.Files.List().Q(q).OrderBy("createdTime desc").Fields("files(id, name, createdTime)").Do()
	if err != nil || len(r.Files) <= MaxBackups {
		return
	}
	
	for i := MaxBackups; i < len(r.Files); i++ {
		srv.Files.Delete(r.Files[i].Id).Do()
	}
}
