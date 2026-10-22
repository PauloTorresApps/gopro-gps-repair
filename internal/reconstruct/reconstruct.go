// Package reconstruct reconstrói a trilha GPS a partir das medições
// aproveitáveis: suaviza o ruído, rejeita saltos e estima os trechos sem
// medição (lacunas no meio por interpolação; pontas por extrapolação curta).
package reconstruct

import (
	"fmt"
	"math"
	"sort"
	"time"

	"gopro-gps-repair/internal/geo"
	"gopro-gps-repair/internal/telemetry"
)

// Status diz de onde veio um valor reconstruído.
type Status string

const (
	Corrected Status = "corrigido" // havia medição aceita; o valor foi suavizado
	Estimated Status = "estimado"  // sem medição utilizável; valor inferido
)

// Point é uma amostra reconstruída, na mesma ordem de Track.Samples.
type Point struct {
	Time      time.Time `json:"horario"`
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Alt       float64   `json:"alt"`
	Speed2D   float64   `json:"velocidade_2d"`
	Speed3D   float64   `json:"velocidade_3d"`
	PosStatus Status    `json:"status_posicao"`
	AltStatus Status    `json:"status_altitude"`
	PosSigma  float64   `json:"incerteza_posicao_m"`
	AltSigma  float64   `json:"incerteza_altitude_m"` // -1: desconhecida
	Shift     float64   `json:"correcao_m"`           // distância até a posição original
}

// Result é a trilha reconstruída e um resumo.
type Result struct {
	Points       []Point `json:"-"`
	PosEstimated int     `json:"posicoes_estimadas"`
	AltEstimated int     `json:"altitudes_estimadas"`
	PosRejected  int     `json:"posicoes_rejeitadas"`
	AltRejected  int     `json:"altitudes_rejeitadas"`
	MaxShift     float64 `json:"maior_correcao_m"`
	MedianSigma  float64 `json:"incerteza_mediana_m"`
}

// MaxExtrapolation limita quanto tempo as pontas são estendidas a partir da
// última velocidade conhecida; além disso, a posição fica parada.
const MaxExtrapolation = 5.0 // s

// Incerteza das medições em função do DOP informado pela câmera. Os fatores
// são empíricos: o DOP da GoPro é um indicador relativo, não um erro em
// metros.
func posSigma(dop float64) float64   { return 1 + 0.8*dopOr(dop, 5) }
func altSigma(dop float64) float64   { return 3 + 1.5*dopOr(dop, 5) }
func speedSigma(dop float64) float64 { return 0.3 + 0.05*dopOr(dop, 5) }

func dopOr(d, def float64) float64 {
	if math.IsNaN(d) {
		return def
	}
	return d
}

// maxUsableDOP: acima disso a câmera está reportando "sem precisão" (99,99).
const maxUsableDOP = 50.0

// Run reconstrói a trilha.
func Run(tr *telemetry.Track) (*Result, error) {
	s := tr.Samples
	N := len(s)
	posOK := make([]bool, N)
	altOK := make([]bool, N)
	first, firstAlt := -1, -1
	for i := range s {
		b := tr.Blocks[s[i].Block]
		dopOK := math.IsNaN(b.DOP) || b.DOP < maxUsableDOP
		posOK[i] = s[i].Valid && b.Fix >= 2 && dopOK
		altOK[i] = s[i].Valid && b.Fix >= 3 && dopOK
		if posOK[i] && first < 0 {
			first = i
		}
		if altOK[i] && firstAlt < 0 {
			firstAlt = i
		}
	}
	if first < 0 {
		return nil, fmt.Errorf("nenhuma amostra com fix: não há o que reconstruir")
	}

	proj := geo.NewLocal(s[first].Lat, s[first].Lon)
	t0 := s[0].Time
	ts := make([]float64, N)
	for i := range s {
		ts[i] = s[i].Time.Sub(t0).Seconds()
	}

	// O erro do receptor muda devagar: as ~18 amostras de um segundo não são
	// medições independentes, e o filtro, que as trata como tal, subestima a
	// incerteza. A incerteza reportada é multiplicada por √(amostras/s), o que
	// equivale a contar cada segundo como uma medição independente. (Inflar o
	// ruído das próprias medições tiraria peso demais da posição: nas curvas a
	// trilha passaria a seguir reto.)
	corr := 1.0
	if span := ts[N-1] - ts[0]; span > 0 {
		corr = math.Max(1, math.Min(5, math.Sqrt(float64(N)/span)))
	}

	// Horizontal: posição + velocidade Doppler.
	hob := make([]obs, N)
	for i := range s {
		hob[i].speed = -1
		if !posOK[i] {
			continue
		}
		dop := tr.Blocks[s[i].Block].DOP
		e, n := proj.ToEN(s[i].Lat, s[i].Lon)
		hob[i] = obs{pos: []float64{e, n}, posSigma: posSigma(dop), speed: s[i].Speed2D, spSigma: speedSigma(dop)}
	}
	h := fitRange(ts, hob, posOK, cvParams{dim: 2, accSigma: 1.5, velSigma0: 10, gate: 16, maxReject: 36})

	// Vertical: só altitude de fix 3D. Manobras verticais são lentas, então a
	// suavização é bem mais forte que na horizontal.
	res := &Result{Points: make([]Point, N)}
	var v fitted
	if firstAlt >= 0 {
		vob := make([]obs, N)
		for i := range s {
			vob[i].speed = -1
			if altOK[i] {
				vob[i] = obs{pos: []float64{s[i].Alt}, posSigma: altSigma(tr.Blocks[s[i].Block].DOP), speed: -1}
			}
		}
		v = fitRange(ts, vob, altOK, cvParams{dim: 1, accSigma: 0.15, velSigma0: 2, gate: 9, maxReject: 54})
	}

	var sigmas []float64
	for i := range s {
		hs := h.at(i, ts)
		lat, lon := proj.ToLatLon(hs.pos[0], hs.pos[1])
		sp2 := math.Hypot(hs.vel[0], hs.vel[1])
		p := Point{
			Time: s[i].Time, Lat: lat, Lon: lon, Speed2D: sp2, Speed3D: sp2,
			PosStatus: Estimated, AltStatus: Estimated,
			PosSigma: hs.sigma * corr,
		}
		if h.used[i] {
			p.PosStatus = Corrected
			p.Shift = geo.Distance(s[i].Lat, s[i].Lon, lat, lon)
			res.MaxShift = math.Max(res.MaxShift, p.Shift)
		} else {
			res.PosEstimated++
			if posOK[i] {
				res.PosRejected++
			}
		}
		if firstAlt >= 0 {
			vs := v.at(i, ts)
			p.Alt, p.AltSigma = vs.pos[0], vs.sigma*corr
			p.Speed3D = math.Hypot(sp2, vs.vel[0])
			if v.used[i] {
				p.AltStatus = Corrected
			} else if altOK[i] {
				res.AltRejected++
			}
		} else {
			// Sem nenhuma altitude 3D: mantém a gravada, que é o melhor que há.
			p.Alt, p.AltSigma = s[i].Alt, -1 // incerteza desconhecida
		}
		if p.AltStatus == Estimated {
			res.AltEstimated++
		}
		sigmas = append(sigmas, p.PosSigma)
		res.Points[i] = p
	}
	res.MedianSigma = median(sigmas)
	return res, nil
}

// fitted guarda o resultado do filtro no trecho entre a primeira e a última
// medição e sabe estender esse resultado para as pontas.
type fitted struct {
	first, last int
	res         cvResult
	used        []bool
	dim         int
}

type state struct {
	pos, vel []float64
	sigma    float64
}

func fitRange(ts []float64, ob []obs, ok []bool, prm cvParams) fitted {
	first, last := -1, -1
	for i := range ok {
		if ok[i] {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	f := fitted{first: first, last: last, used: make([]bool, len(ts)), dim: prm.dim}
	f.res = runCV(ts[first:last+1], ob[first:last+1], prm)
	copy(f.used[first:], f.res.usedPos)
	return f
}

// at devolve o estado no instante da amostra i. Fora do trecho filtrado,
// extrapola com a velocidade da ponta por até MaxExtrapolation segundos.
func (f fitted) at(i int, ts []float64) state {
	d := f.dim
	k := i
	switch {
	case i < f.first:
		k = f.first
	case i > f.last:
		k = f.last
	}
	x := f.res.x[k-f.first]
	P := f.res.p[k-f.first]
	st := state{pos: append([]float64(nil), x[:d]...), vel: append([]float64(nil), x[d:]...)}
	var pv, vv float64
	for j := 0; j < d; j++ {
		pv += P[j][j]
		vv += P[d+j][d+j]
	}
	st.sigma = math.Sqrt(pv / float64(d))

	if dt := ts[i] - ts[k]; dt != 0 {
		ext := math.Max(-MaxExtrapolation, math.Min(MaxExtrapolation, dt))
		for j := 0; j < d; j++ {
			st.pos[j] += st.vel[j] * ext
		}
		if math.Abs(dt) > MaxExtrapolation {
			for j := range st.vel {
				st.vel[j] = 0
			}
		}
		// A incerteza cresce com o tempo extrapolado.
		st.sigma = math.Hypot(st.sigma, math.Sqrt(vv/float64(d))*math.Abs(dt)+0.75*math.Abs(dt)*math.Abs(dt))
	}
	return st
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[len(c)/2]
}
