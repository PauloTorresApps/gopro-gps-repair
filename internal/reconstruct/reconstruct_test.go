package reconstruct

import (
	"math"
	"testing"
	"time"

	"gopro-gps-repair/internal/geo"
	"gopro-gps-repair/internal/telemetry"
)

// Trilha sintética: 30 s a 5 m/s para leste, 18 amostras/s, um bloco por
// segundo. Blocos 0 e 10–12 sem fix; o bloco 20 tem um salto de 80 m.
func synthetic() (*telemetry.Track, geo.Local) {
	proj := geo.NewLocal(-10, -48)
	t0 := time.Date(2025, 8, 16, 20, 0, 0, 0, time.UTC)
	tr := &telemetry.Track{}
	for b := 0; b < 30; b++ {
		blk := telemetry.Block{Index: b, Fix: 3, DOP: 1.5, First: len(tr.Samples), Count: 18, Time: t0.Add(time.Duration(b) * time.Second)}
		if b == 0 || (b >= 10 && b <= 12) {
			blk.Fix, blk.DOP = 0, 99.99
		}
		for j := 0; j < 18; j++ {
			ts := float64(b) + float64(j)/18
			e := 5 * ts
			if b == 20 {
				e += 80
			}
			lat, lon := proj.ToLatLon(e, 0)
			s := telemetry.Sample{Block: b, Time: t0.Add(time.Duration(ts * float64(time.Second))), Lat: lat, Lon: lon, Alt: 300, Speed2D: 5, Valid: true}
			if blk.Fix == 0 {
				s.Lat, s.Lon, s.Speed2D = -10.5, -48.5, 0 // lixo de antes do fix
			}
			tr.Samples = append(tr.Samples, s)
		}
		tr.Blocks = append(tr.Blocks, blk)
	}
	return tr, proj
}

func TestReconstructFillsGapsAndRejectsJump(t *testing.T) {
	tr, proj := synthetic()
	res, err := Run(tr)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range res.Points {
		ts := p.Time.Sub(tr.Samples[0].Time).Seconds()
		e, n := proj.ToEN(p.Lat, p.Lon)
		if err := math.Hypot(e-5*ts, n); err > 2 {
			t.Fatalf("amostra %d (t=%.2f, bloco %d, %s): erro %.1f m", i, ts, tr.Samples[i].Block, p.PosStatus, err)
		}
		if math.Abs(p.Speed2D-5) > 0.5 {
			t.Fatalf("amostra %d: velocidade %.2f", i, p.Speed2D)
		}
	}
	gap := res.Points[11*18]
	if gap.PosStatus != Estimated {
		t.Errorf("lacuna deveria estar marcada como estimada, veio %s", gap.PosStatus)
	}
	if res.PosRejected < 10 {
		t.Errorf("o salto de 80 m deveria ser rejeitado (rejeitadas: %d)", res.PosRejected)
	}
}
