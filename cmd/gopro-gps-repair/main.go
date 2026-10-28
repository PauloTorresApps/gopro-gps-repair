// gopro-gps-repair diagnostica e recupera a trilha GPS gravada pela GoPro.
//
//	gopro-gps-repair diagnosticar <video.MP4> [-blocos] [-json]
//	gopro-gps-repair reparar <video.MP4> [-o saida.MP4] [-gpx trilha.gpx] ...
//	gopro-gps-repair versao
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"gopro-gps-repair/internal/diagnose"
	"gopro-gps-repair/internal/repair"
	"gopro-gps-repair/internal/telemetry"
	"gopro-gps-repair/internal/versao"
)

var usage = `gopro-gps-repair ` + versao.Atual() + `: diagnóstico e recuperação do GPS de vídeos GoPro

Uso:
  gopro-gps-repair diagnosticar <video.MP4> [opções]
  gopro-gps-repair reparar      <video.MP4> [opções]
  gopro-gps-repair versao

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
	case "versao", "versão", "version", "-v", "--version":
		fmt.Println("gopro-gps-repair", versao.Atual())
		return
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
	dop := fs.Float64("dop", repair.DefaultDOP, "DOP declarado nos blocos com medição corrigida")
	dopEst := fs.Float64("dop-estimado", repair.DefaultDOPEstimated, "DOP declarado nos blocos majoritariamente estimados")
	keepTimes := fs.Bool("manter-horarios", false, "não regrava o GPSU de blocos com horário inconsistente")
	keepDate := fs.Bool("manter-data", false, "mantém a data do original (por padrão o arquivo novo recebe nome e data de criação novos)")
	file, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if *out == "" {
		*out = repair.OutputName(file, "", *keepDate, time.Now())
	}

	r, err := repair.Run(context.Background(), file, repair.Options{
		Output: *out, Report: *report, GPX: *gpxPath,
		DOPCorrected: *dop, DOPEstimated: *dopEst,
		KeepTimes: *keepTimes, KeepDate: *keepDate,
	})
	if err != nil {
		return err
	}
	before, after, res, sum := r.Before, r.After, r.Reconstruction, r.Write

	fmt.Printf("Arquivo reparado: %s\n\n", r.Output)
	fmt.Printf("Nota do GPS: %d/100 (%s) → %d/100 (%s, pelo que o arquivo declara)\n",
		before.Score, before.Verdict, after.Score, after.Verdict)
	total := r.Samples
	fmt.Printf("Posição:  %d de %d amostras corrigidas, %d estimadas (%d medições rejeitadas como salto)\n",
		total-res.PosEstimated, total, res.PosEstimated, res.PosRejected)
	fmt.Printf("Altitude: %d de %d amostras corrigidas, %d estimadas (%d medições rejeitadas)\n",
		total-res.AltEstimated, total, res.AltEstimated, res.AltRejected)
	fmt.Printf("Maior correção de posição: %s m · incerteza mediana: %s m\n", num(res.MaxShift, 1), num(res.MedianSigma, 1))
	if before.Fix2D+before.Fix3D > 0 && r.AltRange != nil && r.OrigAltRange != nil {
		fmt.Printf("Altitude: faixa %s–%s m (original %s–%s m)\n",
			num(r.AltRange.Min, 0), num(r.AltRange.Max, 0),
			num(r.OrigAltRange.Min, 0), num(r.OrigAltRange.Max, 0))
	}
	fmt.Printf("Regravados: %d blocos, %d bytes; fix ajustado em %d, DOP em %d, horário em %d\n",
		sum.Blocks, sum.BytesWritten, sum.FixRaised, sum.DOPChanged, sum.TimesRewritten)
	if *keepDate {
		fmt.Println("Verificado: fora dos campos de GPS, idêntico ao original (bytes, data de modificação e permissões)")
	} else {
		fmt.Printf("Verificado: fora dos campos de GPS e das datas, idêntico ao original (metadados GoPro intactos)\nNovo nome e data de criação: %s\n", sum.Created)
	}
	if sum.FixMissing > 0 {
		fmt.Printf("Atenção: %d blocos sem campo de fix gravado não puderam ser marcados como 3D\n", sum.FixMissing)
	}
	fmt.Printf("Relatório: %s\n", r.Report)
	if r.GPX != "" {
		fmt.Printf("GPX: %s\n", r.GPX)
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

// num formata com vírgula decimal.
func num(v float64, prec int) string {
	if math.IsNaN(v) {
		return "—"
	}
	return strings.Replace(fmt.Sprintf("%.*f", prec, v), ".", ",", 1)
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
