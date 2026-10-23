// gopro-gps-repair diagnostica e recupera a trilha GPS gravada pela GoPro.
//
//	gopro-gps-repair diagnosticar <video.MP4> [-blocos] [-json]
//	gopro-gps-repair reparar <video.MP4> [-o saida.MP4] [-gpx trilha.gpx] ...
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"gopro-gps-repair/internal/diagnose"
	"gopro-gps-repair/internal/gpx"
	"gopro-gps-repair/internal/reconstruct"
	"gopro-gps-repair/internal/telemetry"
	"gopro-gps-repair/internal/writer"
)

const usage = `gopro-gps-repair: diagnóstico e recuperação do GPS de vídeos GoPro

Uso:
  gopro-gps-repair diagnosticar <video.MP4> [opções]
  gopro-gps-repair reparar      <video.MP4> [opções]

Use "gopro-gps-repair <comando> -h" para ver as opções.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "diagnosticar", "diagnose":
		err = cmdDiagnose(os.Args[2:])
	case "reparar", "repair":
		err = cmdRepair(os.Args[2:])
	case "-h", "--help", "ajuda", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "comando desconhecido: %s\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

// parseArgs aceita opções antes ou depois do arquivo.
func parseArgs(fs *flag.FlagSet, args []string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() < 1 {
		fs.Usage()
		return "", errors.New("informe o arquivo de vídeo")
	}
	file := fs.Arg(0)
	if err := fs.Parse(fs.Args()[1:]); err != nil {
		return "", err
	}
	if fs.NArg() > 0 {
		return "", fmt.Errorf("argumentos a mais: %v", fs.Args())
	}
	return file, nil
}

func cmdDiagnose(args []string) error {
	fs := flag.NewFlagSet("diagnosticar", flag.ContinueOnError)
	blocks := fs.Bool("blocos", false, "lista cada bloco (~1 s) do GPS")
	asJSON := fs.Bool("json", false, "saída em JSON")
	file, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	tr, err := telemetry.Load(file)
	if err != nil {
		return err
	}
	r := diagnose.Run(tr)
	if *asJSON {
		return printJSON(r)
	}
	printReport(r, *blocks)
	return nil
}

func cmdRepair(args []string) error {
	fs := flag.NewFlagSet("reparar", flag.ContinueOnError)
	out := fs.String("o", "", "arquivo de saída (padrão: <nome>_gps_corrigido.MP4 na mesma pasta)")
	gpxPath := fs.String("gpx", "", "também exporta a trilha reconstruída em GPX")
	report := fs.String("relatorio", "", "relatório JSON (padrão: <saída>.relatorio.json)")
	dop := fs.Float64("dop", 2.0, "DOP declarado nos blocos com medição corrigida")
	dopEst := fs.Float64("dop-estimado", 4.0, "DOP declarado nos blocos majoritariamente estimados")
	keepTimes := fs.Bool("manter-horarios", false, "não regrava o GPSU de blocos com horário inconsistente")
	file, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if *out == "" {
		ext := filepath.Ext(file)
		*out = strings.TrimSuffix(file, ext) + "_gps_corrigido" + ext
	}
	if same(file, *out) {
		return errors.New("a saída não pode ser o próprio arquivo de entrada")
	}
	if _, err := os.Stat(*out); err == nil {
		return fmt.Errorf("%s já existe; apague-o ou use -o", *out)
	}
	if *report == "" {
		*report = strings.TrimSuffix(*out, filepath.Ext(*out)) + ".relatorio.json"
	}

	tr, err := telemetry.Load(file)
	if err != nil {
		return err
	}
	before := diagnose.Run(tr)
	res, err := reconstruct.Run(tr)
	if err != nil {
		return err
	}
	sum, err := writer.Write(tr, res, *out, writer.Options{
		DOPCorrected: *dop, DOPEstimated: *dopEst, FixTimes: !*keepTimes,
	})
	if err != nil {
		return err
	}

	// Relê o arquivo gravado para conferir.
	tr2, err := telemetry.Load(*out)
	if err != nil {
		return fmt.Errorf("arquivo gravado, mas não foi possível relê-lo: %w", err)
	}
	after := diagnose.Run(tr2)

	if *gpxPath != "" {
		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)) + " (GPS reconstruído)"
		if err := gpx.Write(*gpxPath, name, res.Points); err != nil {
			return err
		}
	}
	if err := writeJSON(*report, map[string]any{
		"entrada":       file,
		"saida":         *out,
		"antes":         before,
		"reconstrucao":  res,
		"gravacao":      sum,
		"depois":        after,
		"pontos":        res.Points,
		"observacao":    "Os valores do GPS foram reconstruídos. Pontos com status 'estimado' não têm medição correspondente.",
		"dop_declarado": map[string]float64{"corrigido": *dop, "estimado": *dopEst},
	}); err != nil {
		return err
	}

	fmt.Printf("Arquivo reparado: %s\n\n", *out)
	fmt.Printf("Nota do GPS: %d/100 (%s) → %d/100 (%s, pelo que o arquivo declara)\n",
		before.Score, before.Verdict, after.Score, after.Verdict)
	total := len(res.Points)
	fmt.Printf("Posição:  %d de %d amostras corrigidas, %d estimadas (%d medições rejeitadas como salto)\n",
		total-res.PosEstimated, total, res.PosEstimated, res.PosRejected)
	fmt.Printf("Altitude: %d de %d amostras corrigidas, %d estimadas (%d medições rejeitadas)\n",
		total-res.AltEstimated, total, res.AltEstimated, res.AltRejected)
	fmt.Printf("Maior correção de posição: %s m · incerteza mediana: %s m\n", num(res.MaxShift, 1), num(res.MedianSigma, 1))
	if before.Fix2D+before.Fix3D > 0 {
		fmt.Printf("Altitude: faixa %s–%s m (original %s–%s m)\n",
			num(minAlt(res.Points, true), 0), num(minAlt(res.Points, false), 0),
			num(rawAlt(tr, true), 0), num(rawAlt(tr, false), 0))
	}
	fmt.Printf("Regravados: %d blocos, %d bytes; fix ajustado em %d, DOP em %d, horário em %d\n",
		sum.Blocks, sum.BytesWritten, sum.FixRaised, sum.DOPChanged, sum.TimesRewritten)
	fmt.Println("Verificado: fora dos campos de GPS, idêntico ao original (bytes, data de modificação e permissões)")
	if sum.FixMissing > 0 {
		fmt.Printf("Atenção: %d blocos sem campo de fix gravado não puderam ser marcados como 3D\n", sum.FixMissing)
	}
	fmt.Printf("Relatório: %s\n", *report)
	if *gpxPath != "" {
		fmt.Printf("GPX: %s\n", *gpxPath)
	}
	return nil
}

func printReport(r *diagnose.Report, blocks bool) {
	fmt.Printf("Arquivo: %s\n", r.Path)
	fmt.Printf("GPS: %s → %s UTC (%s s), %d blocos, %d amostras\n",
		r.Start.Format("2006-01-02 15:04:05"), r.End.Format("15:04:05"), num(r.Duration, 1), r.Blocks, r.Samples)
	fmt.Printf("Fix: 3D %d · 2D %d · sem fix %d\n", r.Fix3D, r.Fix2D, r.NoFix)
	fmt.Printf("DOP mediano: %s (amostras com DOP > %s: %d)\n", num(r.MedianDOP, 2), num(diagnose.FairDOP, 0), r.HighDOP)
	fmt.Printf("Saltos de posição: %d · saltos de altitude: %d (maior: %s m)\n", r.Jumps, r.AltJumps, num(r.MaxAltJump, 1))
	if r.TimeSuspect > 0 {
		fmt.Printf("Blocos com horário inconsistente: %d\n", r.TimeSuspect)
	}
	fmt.Printf("Distância com fix 3D: %s m\n", num(r.DistanceM, 0))
	fmt.Printf("\nNota: %d/100 (%s)\n", r.Score, r.Verdict)
	if len(r.Problems) > 0 {
		fmt.Println("Problemas (tempo do vídeo):")
		for _, p := range r.Problems {
			fmt.Println("  •", p)
		}
	}
	if blocks {
		fmt.Println("\n bloco  vídeo(s)  horário UTC    fix   DOP  amostras saltos alt.saltos qualidade")
		for _, b := range r.BlockList {
			flag := ""
			if b.TimeSuspect {
				flag = " horário suspeito"
			}
			fmt.Printf(" %5d  %8s  %s  %3d  %5s  %8d %6d %10d %8d%%%s\n",
				b.Index, num(b.VideoTime, 1), b.Time.Format("15:04:05.000"), b.Fix, num(b.DOP, 2),
				b.Samples, b.Jumps, b.AltJumps, int(math.Round(b.Quality*100)), flag)
		}
	}
}

func minAlt(p []reconstruct.Point, min bool) float64 {
	v := math.Inf(1)
	if !min {
		v = math.Inf(-1)
	}
	for _, x := range p {
		if min {
			v = math.Min(v, x.Alt)
		} else {
			v = math.Max(v, x.Alt)
		}
	}
	return v
}

// rawAlt é a faixa de altitude original, considerando só amostras com fix.
func rawAlt(tr *telemetry.Track, min bool) float64 {
	v := math.Inf(1)
	if !min {
		v = math.Inf(-1)
	}
	for _, s := range tr.Samples {
		if !s.Valid || tr.Blocks[s.Block].Fix < 2 {
			continue
		}
		if min {
			v = math.Min(v, s.Alt)
		} else {
			v = math.Max(v, s.Alt)
		}
	}
	return v
}

// num formata com vírgula decimal.
func num(v float64, prec int) string {
	if math.IsNaN(v) {
		return "—"
	}
	return strings.Replace(fmt.Sprintf("%.*f", prec, v), ".", ",", 1)
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

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
