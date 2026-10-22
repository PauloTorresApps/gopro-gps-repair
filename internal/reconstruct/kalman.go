package reconstruct

import "math"

// Filtro de Kalman de velocidade constante em d dimensões (estado: d
// posições seguidas de d velocidades), seguido do suavizador RTS
// (Rauch–Tung–Striebel). O suavizador usa as medições futuras também, então
// preenche lacunas por interpolação e não fica "atrasado" como um filtro só
// para frente.

// obs é o que se mediu num instante. pos nil = sem medição de posição.
type obs struct {
	pos      []float64
	posSigma float64
	speed    float64 // módulo da velocidade horizontal (Doppler); < 0 = sem medição
	spSigma  float64
}

type cvResult struct {
	x       [][]float64 // estado suavizado em cada instante
	p       []mat       // covariância suavizada
	usedPos []bool      // a medição de posição foi aceita
}

// cvParams ajusta o filtro.
type cvParams struct {
	dim       int
	accSigma  float64 // m/s², intensidade das manobras
	velSigma0 float64 // m/s, incerteza inicial da velocidade
	gate      float64 // limite do qui-quadrado para rejeitar uma medição
	maxReject int     // após tantas rejeições seguidas, aceita (evita travar)
}

func runCV(ts []float64, ob []obs, prm cvParams) cvResult {
	d, n := prm.dim, 2*prm.dim
	N := len(ts)
	xp := make([]mat, N) // previsto
	pp := make([]mat, N)
	xf := make([]mat, N) // filtrado
	pf := make([]mat, N)
	used := make([]bool, N)

	F := func(dt float64) mat {
		f := eye(n)
		for i := 0; i < d; i++ {
			f[i][d+i] = dt
		}
		return f
	}
	Q := func(dt float64) mat {
		q := zeros(n, n)
		a := prm.accSigma * prm.accSigma
		for i := 0; i < d; i++ {
			q[i][i] = dt * dt * dt * dt / 4 * a
			q[i][d+i] = dt * dt * dt / 2 * a
			q[d+i][i] = q[i][d+i]
			q[d+i][d+i] = dt * dt * a
		}
		return q
	}

	// Estado inicial: primeira posição medida, velocidade zero e incerta.
	x := zeros(n, 1)
	P := eye(n)
	for i := 0; i < d; i++ {
		if ob[0].pos != nil {
			x[i][0] = ob[0].pos[i]
		}
		s := math.Max(ob[0].posSigma, 1)
		P[i][i] = s * s
		P[d+i][d+i] = prm.velSigma0 * prm.velSigma0
	}

	rejects := 0
	for k := 0; k < N; k++ {
		if k > 0 {
			dt := ts[k] - ts[k-1]
			f := F(dt)
			x = f.mul(x)
			P = f.mul(P).mul(f.t()).add(Q(dt))
		}
		xp[k], pp[k] = x, P

		if o := ob[k]; o.pos != nil {
			H := zeros(d, n)
			R := zeros(d, d)
			for i := 0; i < d; i++ {
				H[i][i] = 1
				R[i][i] = o.posSigma * o.posSigma
			}
			nx, nP, chi, ok := update(x, P, H, R, col(o.pos))
			if ok && (chi <= prm.gate || rejects >= prm.maxReject || k == 0) {
				x, P = nx, nP
				used[k] = true
				rejects = 0
			} else {
				rejects++
			}
		}
		if o := ob[k]; o.speed >= 0 && d == 2 {
			// Medição não linear: h(x) = |v|. Linearizada na previsão
			// (filtro estendido); com velocidade quase nula a direção é
			// indefinida e a medição é ignorada.
			ve, vn := x[2][0], x[3][0]
			sp := math.Hypot(ve, vn)
			if sp > 0.5 {
				H := zeros(1, n)
				H[0][2], H[0][3] = ve/sp, vn/sp
				R := mat{{o.spSigma * o.spSigma}}
				innov := o.speed - sp
				// update espera z; como h é linearizado em torno de x, z = H·x + inovação.
				z := H.mul(x)[0][0] + innov
				if nx, nP, chi, ok := update(x, P, H, R, mat{{z}}); ok && chi <= prm.gate {
					x, P = nx, nP
				}
			}
		}
		xf[k], pf[k] = x, P
	}

	// Suavizador RTS, de trás para frente.
	xs := make([]mat, N)
	ps := make([]mat, N)
	xs[N-1], ps[N-1] = xf[N-1], pf[N-1]
	for k := N - 2; k >= 0; k-- {
		f := F(ts[k+1] - ts[k])
		ppInv, ok := pp[k+1].inv()
		if !ok {
			xs[k], ps[k] = xf[k], pf[k]
			continue
		}
		C := pf[k].mul(f.t()).mul(ppInv)
		xs[k] = xf[k].add(C.mul(xs[k+1].sub(xp[k+1])))
		ps[k] = pf[k].add(C.mul(ps[k+1].sub(pp[k+1])).mul(C.t()))
	}

	res := cvResult{x: make([][]float64, N), p: ps, usedPos: used}
	for k := range xs {
		res.x[k] = xs[k].vec()
	}
	return res
}

// update aplica uma medição z = H·x + ruído(R). Devolve também a distância de
// Mahalanobis² da inovação, usada para rejeitar outliers.
func update(x, P, H, R, z mat) (mat, mat, float64, bool) {
	y := z.sub(H.mul(x))
	S := H.mul(P).mul(H.t()).add(R)
	Si, ok := S.inv()
	if !ok {
		return x, P, 0, false
	}
	chi := y.t().mul(Si).mul(y)[0][0]
	K := P.mul(H.t()).mul(Si)
	nx := x.add(K.mul(y))
	nP := eye(len(P)).sub(K.mul(H)).mul(P)
	return nx, nP, chi, true
}
