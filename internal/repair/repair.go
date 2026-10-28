// Package repair reúne o fluxo completo do reparo (leitura, reconstrução,
// gravação, verificação, releitura e relatório). É usado pela linha de comando
// e pela interface gráfica, para que as duas façam exatamente o mesmo.
package repair

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopro-gps-repair/internal/diagnose"
	"gopro-gps-repair/internal/gpx"
	"gopro-gps-repair/internal/reconstruct"
	"gopro-gps-repair/internal/telemetry"
	"gopro-gps-repair/internal/writer"
)

// Valores padrão do DOP declarado no arquivo novo.
const (
	DefaultDOP          = 2.0
	DefaultDOPEstimated = 4.0
)

// Etapas informadas em Options.Progress.
type Stage string

const (
	StageRead        Stage = "leitura"
	StageReconstruct Stage = "reconstrucao"
	StageCopy        Stage = Stage(writer.StageCopy)
	StageVerify      Stage = Stage(writer.StageVerify)
	StageReread      Stage = "releitura"
	StageReport      Stage = "relatorio"
)

// Options controla o reparo.
type Options struct {
	Output       string // arquivo de saída; obrigatório
	Report       string // relatório JSON (padrão: <saída>.relatorio.json)
	GPX          string // trilha GPX opcional
	DOPCorrected float64
	DOPEstimated float64
	KeepTimes    bool // não regrava o GPSU de blocos com horário inconsistente
	KeepDate     bool // mantém a data do original
	// Progress recebe o andamento. Nas etapas de cópia e verificação, done e
	// total são bytes; nas demais, total é 0.
	Progress func(stage Stage, done, total int64)
}

// Range é uma faixa de valores.
type Range struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// Result descreve o reparo feito.
type Result struct {
	Input          string              `json:"entrada"`
	Output         string              `json:"saida"`
	Report         string              `json:"relatorio"`
	GPX            string              `json:"gpx,omitempty"`
	Before         *diagnose.Report    `json:"antes"`
	After          *diagnose.Report    `json:"depois"`
	Reconstruction *reconstruct.Result `json:"reconstrucao"`
	Write          *writer.Summary     `json:"gravacao"`
	Samples        int                 `json:"amostras"`
	AltRange       *Range              `json:"faixa_altitude,omitempty"`
	OrigAltRange   *Range              `json:"faixa_altitude_original,omitempty"`
	KeepDate       bool                `json:"manter_data"`
	DOPCorrected   float64             `json:"dop_corrigido"`
	DOPEstimated   float64             `json:"dop_estimado"`
}

// OutputName dá o nome padrão do arquivo de saída, na pasta dir (ou na pasta
// do original, se dir for vazio).
func OutputName(input, dir string, keepDate bool, now time.Time) string {
	ext := filepath.Ext(input)
	base := strings.TrimSuffix(filepath.Base(input), ext) + "_gps_corrigido"
	if !keepDate {
		base += "_" + now.Format("20060102_150405")
	}
	if dir == "" {
		dir = filepath.Dir(input)
	}
	return filepath.Join(dir, base+ext)
}

// ReportName dá o nome padrão do relatório para a saída out.
func ReportName(out string) string {
	return strings.TrimSuffix(out, filepath.Ext(out)) + ".relatorio.json"
}

// Run repara input. Se ctx for cancelado, para na próxima oportunidade e não
// deixa arquivo de saída para trás.
func Run(ctx context.Context, input string, opt Options) (*Result, error) {
	if opt.Output == "" {
		return nil, errors.New("arquivo de saída não informado")
	}
	if opt.DOPCorrected == 0 {
		opt.DOPCorrected = DefaultDOP
	}
	if opt.DOPEstimated == 0 {
		opt.DOPEstimated = DefaultDOPEstimated
	}
	if opt.Report == "" {
		opt.Report = ReportName(opt.Output)
	}
	if same(input, opt.Output) {
		return nil, errors.New("a saída não pode ser o próprio arquivo de entrada")
	}
	for _, p := range []string{opt.Output, opt.Report, opt.GPX} {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return nil, fmt.Errorf("%s já existe; apague-o ou use -o", p)
		}
	}
	progress := throttle(opt.Progress)

	progress(StageRead, 0, 0)
	tr, err := telemetry.Load(input)
	if err != nil {
		return nil, err
	}
	before := diagnose.Run(tr)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	progress(StageReconstruct, 0, 0)
	res, err := reconstruct.Run(tr)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sum, err := writer.WriteContext(ctx, tr, res, opt.Output, writer.Options{
		DOPCorrected: opt.DOPCorrected, DOPEstimated: opt.DOPEstimated,
		FixTimes: !opt.KeepTimes, KeepDate: opt.KeepDate,
		Progress: func(s writer.Stage, done, total int64) { progress(Stage(s), done, total) },
	})
	if err != nil {
		return nil, err
	}

	// A partir daqui o arquivo já existe; se algo falhar ou o usuário abortar,
	// ele é apagado.
	ok := false
	defer func() {
		if !ok {
			os.Remove(opt.Output)
			os.Remove(opt.Report)
			if opt.GPX != "" {
				os.Remove(opt.GPX)
			}
		}
	}()

	progress(StageReread, 0, 0)
	tr2, err := telemetry.Load(opt.Output)
	if err != nil {
		return nil, fmt.Errorf("arquivo gravado, mas não foi possível relê-lo: %w", err)
	}
	after := diagnose.Run(tr2)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r := &Result{
		Input: input, Output: opt.Output, Report: opt.Report, GPX: opt.GPX,
		Before: before, After: after, Reconstruction: res, Write: sum,
		Samples:  len(res.Points),
		AltRange: pointsAltRange(res.Points), OrigAltRange: rawAltRange(tr),
		KeepDate: opt.KeepDate, DOPCorrected: opt.DOPCorrected, DOPEstimated: opt.DOPEstimated,
	}

	progress(StageReport, 0, 0)
	if opt.GPX != "" {
		name := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input)) + " (GPS reconstruído)"
		if err := gpx.Write(opt.GPX, name, res.Points); err != nil {
			return nil, err
		}
	}
	if err := writeJSON(opt.Report, map[string]any{
		"entrada":       input,
		"saida":         opt.Output,
		"antes":         before,
		"reconstrucao":  res,
		"gravacao":      sum,
		"depois":        after,
		"pontos":        res.Points,
		"observacao":    "Os valores do GPS foram reconstruídos. Pontos com status 'estimado' não têm medição correspondente.",
		"dop_declarado": map[string]float64{"corrigido": opt.DOPCorrected, "estimado": opt.DOPEstimated},
	}); err != nil {
		return nil, err
	}
	ok = true
	return r, nil
}

// throttle limita os avisos de progresso em bytes a ~20 por segundo; mudanças
// de etapa e o fim de cada etapa sempre passam.
func throttle(f func(Stage, int64, int64)) func(Stage, int64, int64) {
	if f == nil {
		return func(Stage, int64, int64) {}
	}
	var last time.Time
	var lastStage Stage
	return func(s Stage, done, total int64) {
		now := time.Now()
		if s == lastStage && done < total && now.Sub(last) < 50*time.Millisecond {
			return
		}
		last, lastStage = now, s
		f(s, done, total)
	}
}

func pointsAltRange(p []reconstruct.Point) *Range {
	r := Range{math.Inf(1), math.Inf(-1)}
	for _, x := range p {
		r.Min, r.Max = math.Min(r.Min, x.Alt), math.Max(r.Max, x.Alt)
	}
	return finite(r)
}

// rawAltRange é a faixa de altitude original, só com amostras com fix.
func rawAltRange(tr *telemetry.Track) *Range {
	r := Range{math.Inf(1), math.Inf(-1)}
	for _, s := range tr.Samples {
		if !s.Valid || tr.Blocks[s.Block].Fix < 2 {
			continue
		}
		r.Min, r.Max = math.Min(r.Min, s.Alt), math.Max(r.Max, s.Alt)
	}
	return finite(r)
}

func finite(r Range) *Range {
	if math.IsInf(r.Min, 0) || math.IsInf(r.Max, 0) || math.IsNaN(r.Min) || math.IsNaN(r.Max) {
		return nil
	}
	return &r
}

func same(a, b string) bool {
	sa, err1 := os.Stat(a)
	sb, err2 := os.Stat(b)
	if err1 == nil && err2 == nil {
		return os.SameFile(sa, sb)
	}
	aa, _ := filepath.Abs(a)
	bb, _ := filepath.Abs(b)
	return aa == bb
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
