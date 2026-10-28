// Interface gráfica do gopro-gps-repair (Wails v3 + React).
//
// A tela escolhe um vídeo, mostra a avaliação do GPS, pede confirmação e
// restaura os dados, gravando o vídeo novo em ~/gopro-gps-repair. Diagnóstico e
// reparo são os mesmos da linha de comando (pacotes internal/diagnose e
// internal/repair).
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

// AppName é o nome exibido. "for GoPro" indica compatibilidade: o app não é
// um produto oficial da GoPro.
const AppName = "GPS Repair for GoPro"

func init() {
	application.RegisterEvent[Progress](EventProgress)
	application.RegisterEvent[string](EventDropped)
}

func main() {
	app := application.New(application.Options{
		Name:        AppName,
		Description: "Diagnosis and recovery of GPS data in GoPro videos",
		Services: []application.Service{
			application.NewService(&RepairService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Linux: application.LinuxOptions{ProgramName: "gopro-gps-repair-gui"},
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            AppName,
		Width:            1040,
		Height:           760,
		MinWidth:         720,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(14, 14, 15),
		EnableFileDrop:   true,
		URL:              "/",
	})

	// Vídeo arrastado para a área de entrada: repassa o caminho à interface.
	win.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
		if files := e.Context().DroppedFiles(); len(files) > 0 {
			app.Event.Emit(EventDropped, files[0])
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
