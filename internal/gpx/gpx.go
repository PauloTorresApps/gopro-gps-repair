// Package gpx exporta a trilha reconstruída em GPX 1.1.
package gpx

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"gopro-gps-repair/internal/reconstruct"
)

// Write grava os pontos em path. O status de cada ponto vai no <desc>.
func Write(path, name string, pts []reconstruct.Point) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="gopro-gps-repair" xmlns="http://www.topografix.com/GPX/1/1">
 <trk><name>%s</name><trkseg>
`, xmlEscape(name))
	for _, p := range pts {
		fmt.Fprintf(w, "  <trkpt lat=\"%.7f\" lon=\"%.7f\"><ele>%.2f</ele><time>%s</time><desc>posição %s, altitude %s</desc></trkpt>\n",
			p.Lat, p.Lon, p.Alt, p.Time.UTC().Format(time.RFC3339Nano), p.PosStatus, p.AltStatus)
	}
	fmt.Fprint(w, " </trkseg></trk>\n</gpx>\n")
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func xmlEscape(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch r {
		case '&':
			out = append(out, []rune("&amp;")...)
		case '<':
			out = append(out, []rune("&lt;")...)
		case '>':
			out = append(out, []rune("&gt;")...)
		default:
			out = append(out, r)
		}
	}
	return string(out)
}
