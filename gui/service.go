package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	goprogpsrepair "gopro-gps-repair"
	"gopro-gps-repair/internal/diagnose"
	"gopro-gps-repair/internal/repair"
	"gopro-gps-repair/internal/telemetry"
)

// OutputDirName é a pasta, dentro do diretório pessoal do usuário, onde os
// vídeos restaurados são gravados.
const OutputDirName = "gopro-gps-repair"

// Eventos enviados à interface.
const (
	EventProgress = "reparo:progresso"
	EventDropped  = "arquivo:solto"
)

// Progress é o andamento do reparo enviado à interface.
type Progress struct {
	Stage string `json:"etapa"`
	Done  int64  `json:"feito"`
	Total int64  `json:"total"`
}

// RepairService é o que a interface chama. Diagnóstico e reparo usam os
// mesmos pacotes da linha de comando.
type RepairService struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

// ServiceStartup roda dentro de app.Run, depois de o Wails criar a aplicação
// GTK e antes de registrá-la: é o momento de ajustar o ID no Linux.
func (s *RepairService) ServiceStartup(context.Context, application.ServiceOptions) error {
	setLinuxAppID()
	return nil
}

// Version devolve a versão do app (arquivo VERSION, embutido na compilação).
func (s *RepairService) Version() string {
	return goprogpsrepair.Version
}

// OutputDir devolve (e cria, se preciso) a pasta de saída: ~/gopro-gps-repair.
func (s *RepairService) OutputDir() (string, error) {
	// No Snap, HOME aponta para ~/snap/<app>/<revisão>; o diretório pessoal
	// de verdade vem em SNAP_REAL_HOME.
	home := os.Getenv("SNAP_REAL_HOME")
	if home == "" {
		var err error
		if home, err = os.UserHomeDir(); err != nil {
			return "", err
		}
	}
	dir := filepath.Join(home, OutputDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// SelectVideo abre o seletor de arquivos e devolve o vídeo escolhido ("" se
// o usuário cancelar).
func (s *RepairService) SelectVideo(title string) (string, error) {
	d := application.Get().Dialog.OpenFile().
		SetTitle(title).
		CanChooseFiles(true).
		AddFilter("MP4", "*.mp4;*.MP4")
	if w := application.Get().Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	return d.PromptForSingleSelection()
}

// Diagnose avalia o GPS do vídeo sem alterar nada.
func (s *RepairService) Diagnose(path string) (*diagnose.Report, error) {
	if err := checkVideo(path); err != nil {
		return nil, err
	}
	tr, err := telemetry.Load(path)
	if err != nil {
		return nil, err
	}
	r := diagnose.Run(tr)
	r.BlockList = compact(r.BlockList)
	return r, nil
}

// Repair restaura o GPS do vídeo e grava o resultado em ~/gopro-gps-repair.
// Pode ser interrompido por Abort; nesse caso nenhum arquivo fica gravado.
func (s *RepairService) Repair(ctx context.Context, path string) (*repair.Result, error) {
	if err := checkVideo(path); err != nil {
		return nil, err
	}
	dir, err := s.OutputDir()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		cancel()
		return nil, errors.New("já há uma restauração em andamento")
	}
	s.cancel = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.cancel = nil
		s.mu.Unlock()
		cancel()
	}()

	app := application.Get()
	r, err := repair.Run(ctx, path, repair.Options{
		Output: repair.OutputName(path, dir, false, time.Now()),
		Progress: func(st repair.Stage, done, total int64) {
			app.Event.Emit(EventProgress, Progress{Stage: string(st), Done: done, Total: total})
		},
	})
	if errors.Is(err, context.Canceled) {
		return nil, errors.New("cancelado")
	}
	if err != nil {
		return nil, err
	}
	r.Before.BlockList = compact(r.Before.BlockList)
	r.After.BlockList = compact(r.After.BlockList)
	return r, nil
}

// Abort interrompe a restauração em andamento, se houver.
func (s *RepairService) Abort() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// ShowInFolder abre o gerenciador de arquivos com o arquivo selecionado.
func (s *RepairService) ShowInFolder(path string) error {
	return application.Get().Env.OpenFileManager(path, true)
}

func checkVideo(path string) error {
	if path == "" {
		return errors.New("nenhum vídeo escolhido")
	}
	if ext := strings.ToLower(filepath.Ext(path)); ext != ".mp4" {
		return errors.New("o arquivo não é um vídeo MP4")
	}
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return errors.New("o caminho é uma pasta, não um vídeo")
	}
	return nil
}

// compact reduz a lista de blocos a no máximo 600 itens, o bastante para a
// faixa de qualidade da interface em vídeos longos; cada item fica com a pior
// qualidade do grupo que representa.
func compact(blocks []diagnose.BlockReport) []diagnose.BlockReport {
	const max = 600
	if len(blocks) <= max {
		return blocks
	}
	step := (len(blocks) + max - 1) / max
	out := make([]diagnose.BlockReport, 0, max)
	for i := 0; i < len(blocks); i += step {
		g := blocks[i]
		for j := i + 1; j < i+step && j < len(blocks); j++ {
			if blocks[j].Quality < g.Quality {
				q, idx, vt := blocks[j].Quality, g.Index, g.VideoTime
				g = blocks[j]
				g.Quality, g.Index, g.VideoTime = q, idx, vt
			}
		}
		out = append(out, g)
	}
	return out
}
