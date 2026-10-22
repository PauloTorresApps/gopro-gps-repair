package reconstruct

import "math"

// Matrizes pequenas (até 4×4) para o filtro de Kalman.
type mat [][]float64

func zeros(r, c int) mat {
	m := make(mat, r)
	for i := range m {
		m[i] = make([]float64, c)
	}
	return m
}

func eye(n int) mat {
	m := zeros(n, n)
	for i := range m {
		m[i][i] = 1
	}
	return m
}

func (a mat) mul(b mat) mat {
	out := zeros(len(a), len(b[0]))
	for i := range a {
		for k := range b {
			if a[i][k] == 0 {
				continue
			}
			for j := range b[0] {
				out[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return out
}

func (a mat) t() mat {
	out := zeros(len(a[0]), len(a))
	for i := range a {
		for j := range a[0] {
			out[j][i] = a[i][j]
		}
	}
	return out
}

func (a mat) add(b mat) mat {
	out := zeros(len(a), len(a[0]))
	for i := range a {
		for j := range a[0] {
			out[i][j] = a[i][j] + b[i][j]
		}
	}
	return out
}

func (a mat) sub(b mat) mat {
	out := zeros(len(a), len(a[0]))
	for i := range a {
		for j := range a[0] {
			out[i][j] = a[i][j] - b[i][j]
		}
	}
	return out
}

// inv inverte por Gauss-Jordan com pivotamento parcial.
func (a mat) inv() (mat, bool) {
	n := len(a)
	m := zeros(n, 2*n)
	for i := range a {
		copy(m[i], a[i])
		m[i][n+i] = 1
	}
	for c := 0; c < n; c++ {
		p := c
		for r := c + 1; r < n; r++ {
			if math.Abs(m[r][c]) > math.Abs(m[p][c]) {
				p = r
			}
		}
		if math.Abs(m[p][c]) < 1e-12 {
			return nil, false
		}
		m[c], m[p] = m[p], m[c]
		piv := m[c][c]
		for j := range m[c] {
			m[c][j] /= piv
		}
		for r := 0; r < n; r++ {
			if r != c && m[r][c] != 0 {
				f := m[r][c]
				for j := range m[r] {
					m[r][j] -= f * m[c][j]
				}
			}
		}
	}
	out := zeros(n, n)
	for i := range out {
		copy(out[i], m[i][n:])
	}
	return out, true
}

func col(v []float64) mat {
	m := zeros(len(v), 1)
	for i, x := range v {
		m[i][0] = x
	}
	return m
}

func (a mat) vec() []float64 {
	out := make([]float64, len(a))
	for i := range a {
		out[i] = a[i][0]
	}
	return out
}
