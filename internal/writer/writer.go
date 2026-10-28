// Package writer grava a trilha reconstruída num novo MP4. O arquivo é
// copiado byte a byte e só os campos de GPS (GPS5, GPSF, GPSP e, se preciso,
// GPSU) são sobrescritos no lugar, com o mesmo tamanho: vídeo, áudio, demais
// sensores e metadados ficam idênticos ao original.
package writer

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"gopro-gps-repair/internal/gpmf"
	"gopro-gps-repair/internal/mp4"
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
	// KeepDate mantém no arquivo novo a data de modificação do original e as
	// datas de criação do contêiner MP4. Por padrão o arquivo recebe a data e a
	// hora atuais, para a nuvem da GoPro não o tomar por duplicata do original.
	KeepDate bool
	// Progress, se definido, recebe o andamento das etapas longas (cópia e
	// verificação), em bytes.
	Progress func(stage Stage, done, total int64)
}

// Stage identifica uma etapa longa da gravação.
type Stage string

const (
	StageCopy   Stage = "copia"
	StageVerify Stage = "verificacao"
)

// mp4Epoch é a origem das datas do contêiner MP4 (1904-01-01 UTC).
var mp4Epoch = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)

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
	ContainerDates int     `json:"datas_do_conteiner_regravadas"`
	Created        string  `json:"data_de_criacao,omitempty"`
	BytesWritten   int     `json:"bytes_regravados"`
}

// Write cria out (que não pode existir) a partir de tr.Path.
func Write(tr *telemetry.Track, res *reconstruct.Result, out string, opt Options) (*Summary, error) {
	return WriteContext(context.Background(), tr, res, out, opt)
}

// WriteContext é Write com cancelamento: se ctx for cancelado durante a cópia
// ou a verificação, o arquivo novo é apagado e o erro é ctx.Err().
func WriteContext(ctx context.Context, tr *telemetry.Track, res *reconstruct.Result, out string, opt Options) (*Summary, error) {
	if err := copyFile(ctx, tr.Path, out, opt.Progress); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(out, os.O_RDWR, 0)
	if err != nil {
		os.Remove(out)
		return nil, err
	}
	sum := &Summary{}
	now := time.Now().UTC().Truncate(time.Second)
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
		if !opt.KeepDate {
			if err := putContainerDates(f, tr.Path, now, sum); err != nil {
				return err
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
		err = verifyOnlyPatched(ctx, tr.Path, out, sum.Patches, opt.Progress)
	}
	if err == nil {
		err = copyFileInfo(tr.Path, out, opt.KeepDate, now)
		if err == nil && !opt.KeepDate {
			sum.Created = now.Format(time.RFC3339)
		}
	}
	if err != nil {
		os.Remove(out)
		return nil, err
	}
	return sum, nil
}

// putContainerDates grava a data atual como criação e modificação do contêiner
// (mvhd, tkhd, mdhd). Os campos têm tamanho fixo e entram na lista de trechos
// que a verificação aceita; nenhum metadado da GoPro é tocado.
func putContainerDates(f *os.File, src string, now time.Time, sum *Summary) error {
	fields, err := mp4.TimeFields(src)
	if err != nil {
		return err
	}
	secs := uint64(now.Sub(mp4Epoch) / time.Second)
	for _, tf := range fields {
		buf := make([]byte, tf.Len)
		if tf.Wide {
			for i := 0; i < 2; i++ {
				binary.BigEndian.PutUint64(buf[i*8:], secs)
			}
		} else {
			for i := 0; i < 2; i++ {
				binary.BigEndian.PutUint32(buf[i*4:], uint32(secs))
			}
		}
		if _, err := f.WriteAt(buf, tf.Off); err != nil {
			return err
		}
		sum.Patches = append(sum.Patches, Patch{tf.Off, tf.Len, "datas"})
		sum.BytesWritten += tf.Len
		sum.ContainerDates++
	}
	return nil
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

func copyFile(ctx context.Context, src, dst string, progress func(Stage, int64, int64)) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	r := &ctxReader{ctx: ctx, r: in, total: st.Size(), stage: StageCopy, progress: progress}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}

// ctxReader interrompe a leitura quando ctx é cancelado e informa o progresso.
type ctxReader struct {
	ctx      context.Context
	r        io.Reader
	done     int64
	total    int64
	stage    Stage
	progress func(Stage, int64, int64)
}

func (c *ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := c.r.Read(p)
	c.done += int64(n)
	if c.progress != nil {
		c.progress(c.stage, c.done, c.total)
	}
	return n, err
}
