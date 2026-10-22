// Package geo tem as conversões entre coordenadas geográficas e metros.
package geo

import "math"

const earthRadius = 6371008.8 // m, raio médio

// Distance é a distância em metros entre dois pontos (haversine).
func Distance(lat1, lon1, lat2, lon2 float64) float64 {
	p1, p2 := rad(lat1), rad(lat2)
	dp, dl := p2-p1, rad(lon2-lon1)
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * earthRadius * math.Asin(math.Min(1, math.Sqrt(a)))
}

// Local projeta coordenadas num plano tangente (leste, norte em metros) em
// torno de uma origem. Para trechos de poucos km o erro é desprezível.
type Local struct{ lat0, lon0, cos0 float64 }

func NewLocal(lat0, lon0 float64) Local {
	return Local{lat0, lon0, math.Cos(rad(lat0))}
}

func (l Local) ToEN(lat, lon float64) (e, n float64) {
	return rad(lon-l.lon0) * l.cos0 * earthRadius, rad(lat-l.lat0) * earthRadius
}

func (l Local) ToLatLon(e, n float64) (lat, lon float64) {
	return l.lat0 + deg(n/earthRadius), l.lon0 + deg(e/(earthRadius*l.cos0))
}

func rad(d float64) float64 { return d * math.Pi / 180 }
func deg(r float64) float64 { return r * 180 / math.Pi }
