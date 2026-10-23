// Package writer grava a trilha reconstruída num novo MP4. O arquivo é
// copiado byte a byte e só os campos de GPS (GPS5, GPSF, GPSP e, se preciso,
// GPSU) são sobrescritos no lugar, com o mesmo tamanho: vídeo, áudio, demais
// sensores e metadados ficam idênticos ao original.
package writer

import (
	"fmt"
	"io"
	"math"
	"os"

	"gopro-gps-repair/internal/gpmf"
	"gopro-gps-repair/internal/reconstruct"
	"gopro-gps-repair/internal/telemetry"
)

// Options controla o que é declarado no arquivo novo.
type Options struct {
	// DOP gravado nos blocos com medição corrigida e nos blocos majoritariamente
	// estimados. O valor declarado não deve fingir mais precisão do que há.
	DOPCorrected float64
	DOPEstimated float64
	// FixTimes regrava o GPSU dos blocos com horário inconsistente.
	FixTimes bool
}

// Patch é um trecho do arquivo que foi sobrescrito.
type Patch struct {
	Off int64
	Len int
	Key string
}

// Summary descreve o que foi gravado.
type Summary struct {
	Patches        []Patch `json:"-"`
	Blocks         int     `json:"blocos_regravados"`
	FixRaised      int     `json:"blocos_com_fix_ajustado"`
	FixMissing     int     `json:"blocos_sem_campo_fix"`
	DOPChanged     int     `json:"blocos_com_dop_ajustado"`
	TimesRewritten int     `json:"horarios_regravados"`
	BytesWritten   int     `json:"bytes_regravados"`
}

// Write cria out (que não pode existir) a partir de tr.Path.
func Write(tr *telemetry.Track, res *reconstruct.Result, out string, opt Options) (*Summary, error) {
	if err := copyFile(tr.Path, out); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(out, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	sum := &Summary{}
	put := func(fd *telemetry.Field, key string, data []byte) error {
		if len(data) != fd.Len {
			return fmt.Errorf("%s: tamanho %d diferente do original %d", key, len(data), fd.Len)
		}
		if _, err := f.WriteAt(data, fd.Off); err != nil {
			return err
		}
		sum.Patches = append(sum.Patches, Patch{fd.Off, fd.Len, key})
		sum.BytesWritten += fd.Len
		return nil
	}

	err = func() error {
		for _, b := range tr.Blocks {
			// GPS5: 5 inteiros de 32 bits por amostra, na escala do bloco.
			buf := make([]byte, b.GPS5.Len)
			estimated := 0
			for j := 0; j < b.Count; j++ {
				p := res.Points[b.First+j]
				if p.PosStatus == reconstruct.Estimated {
					estimated++
				}
				vals := [5]float64{p.Lat, p.Lon, p.Alt, p.Speed2D, p.Speed3D}
				for c, v := range vals {
					raw := int64(math.Round(v * b.Scale[c]))
					if err := gpmf.PutInt('l', buf[j*20+c*4:], raw); err != nil {
						return fmt.Errorf("bloco %d: %w", b.Index, err)
					}
				}
			}
			if err := put(&b.GPS5, "GPS5", buf); err != nil {
				return err
			}
			sum.Blocks++

			if b.FixField != nil {
				if b.Fix != 3 {
					sum.FixRaised++
				}
				if err := putInt(put, b.FixField, "GPSF", 3); err != nil {
					return err
				}
			} else if b.Fix != 3 {
				sum.FixMissing++
			}

			if b.DOPField != nil {
				dop := opt.DOPCorrected
				if estimated*2 > b.Count {
					dop = opt.DOPEstimated
				}
				// Se a precisão original já é melhor que a declarada, mantém a original.
				if !math.IsNaN(b.DOP) && b.DOP < dop {
					dop = b.DOP
				}
				if math.Abs(dop-b.DOP) > 0.005 {
					sum.DOPChanged++
				}
				if err := putInt(put, b.DOPField, "GPSP", int64(math.Round(dop*100))); err != nil {
					return err
				}
			}

			if opt.FixTimes && b.TimeSuspect && b.GPSUField != nil {
				s := []byte(b.Time.UTC().Format("060102150405.000"))
				if len(s) <= b.GPSUField.Len {
					data := make([]byte, b.GPSUField.Len)
					copy(data, s)
					if err := put(b.GPSUField, "GPSU", data); err != nil {
						return err
					}
					sum.TimesRewritten++
				}
			}
		}
		return f.Sync()
	}()
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	// Garantia: fora dos campos de GPS regravados, o arquivo novo tem que ser
	// idêntico ao original. Se não for, ele é descartado.
	if err == nil {
		err = VerifyOnlyPatched(tr.Path, out, sum.Patches)
	}
	if err == nil {
		err = copyFileInfo(tr.Path, out)
	}
	if err != nil {
		os.Remove(out)
		return nil, err
	}
	return sum, nil
}

func putInt(put func(*telemetry.Field, string, []byte) error, fd *telemetry.Field, key string, v int64) error {
	data := make([]byte, fd.Len)
	w := fd.Len // um único valor; se houver mais, todos recebem v
	switch fd.Type {
	case 'b', 'B':
		w = 1
	case 's', 'S':
		w = 2
	case 'l', 'L':
		w = 4
	}
	for off := 0; off+w <= len(data); off += w {
		if err := gpmf.PutInt(fd.Type, data[off:], v); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	return put(fd, key, data)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}
