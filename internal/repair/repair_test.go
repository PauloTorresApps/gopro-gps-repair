package repair

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopro-gps-repair/internal/diagnose"
	"gopro-gps-repair/internal/telemetry"
	"gopro-gps-repair/internal/testvideo"
)

func sample(t *testing.T, opt testvideo.Options) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "GH010001.MP4")
	if err := testvideo.Write(p, opt); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRun(t *testing.T) {
	in := sample(t, testvideo.Options{})
	tr, err := telemetry.Load(in)
	if err != nil {
		t.Fatal(err)
	}
	d := diagnose.Run(tr)
	if d.Verdict == "bom" || len(d.Issues) == 0 {
		t.Fatalf("o vídeo de teste deveria ter GPS degradado: nota %d, %v", d.Score, d.Issues)
	}

	out := OutputName(in, t.TempDir(), false, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	if filepath.Base(out) != "GH010001_gps_corrigido_20260102_030405.MP4" {
		t.Errorf("nome de saída inesperado: %s", out)
	}
	var stages []Stage
	r, err := Run(context.Background(), in, Options{
		Output:   out,
		Progress: func(s Stage, _, _ int64) { stages = append(stages, s) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.After.Score <= r.Before.Score {
		t.Errorf("nota não melhorou: %d → %d", r.Before.Score, r.After.Score)
	}
	if r.Reconstruction.PosEstimated == 0 {
		t.Error("o início sem fix deveria gerar posições estimadas")
	}
	for _, p := range []string{r.Output, r.Report} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("arquivo não gerado: %v", err)
		}
	}
	if len(stages) == 0 || stages[0] != StageRead || stages[len(stages)-1] != StageReport {
		t.Errorf("etapas inesperadas: %v", stages)
	}

	// Não sobrescreve.
	if _, err := Run(context.Background(), in, Options{Output: out}); err == nil {
		t.Error("deveria recusar saída existente")
	}
}

func TestRunCancel(t *testing.T) {
	in := sample(t, testvideo.Options{Filler: 64 << 20})
	out := filepath.Join(t.TempDir(), "saida.MP4")
	ctx, cancel := context.WithCancel(context.Background())
	_, err := Run(ctx, in, Options{
		Output: out,
		Progress: func(s Stage, done, _ int64) {
			if s == StageCopy && done > 1<<20 {
				cancel()
			}
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("esperava cancelamento, veio %v", err)
	}
	for _, p := range []string{out, ReportName(out)} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s deveria ter sido apagado", p)
		}
	}
}
