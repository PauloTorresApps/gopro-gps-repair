package writer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

// VerifyOnlyPatched compara orig e out byte a byte e falha se houver qualquer
// diferença fora dos trechos regravados, ou se os tamanhos diferirem. Assim,
// metadados da GoPro (ou de qualquer outra fonte), vídeo, áudio e demais
// sensores ficam garantidamente idênticos ao original.
func VerifyOnlyPatched(orig, out string, patches []Patch) error {
	return verifyOnlyPatched(context.Background(), orig, out, patches, nil)
}

func verifyOnlyPatched(ctx context.Context, orig, out string, patches []Patch, progress func(Stage, int64, int64)) error {
	a, err := os.Open(orig)
	if err != nil {
		return err
	}
	defer a.Close()
	b, err := os.Open(out)
	if err != nil {
		return err
	}
	defer b.Close()

	sa, err := a.Stat()
	if err != nil {
		return err
	}
	sb, err := b.Stat()
	if err != nil {
		return err
	}
	if sa.Size() != sb.Size() {
		return fmt.Errorf("verificação: tamanho mudou (%d → %d bytes)", sa.Size(), sb.Size())
	}

	ps := append([]Patch(nil), patches...)
	sort.Slice(ps, func(i, j int) bool { return ps[i].Off < ps[j].Off })
	allowed := func(off int64) bool {
		i := sort.Search(len(ps), func(i int) bool { return ps[i].Off+int64(ps[i].Len) > off })
		return i < len(ps) && ps[i].Off <= off
	}

	const chunk = 1 << 20
	// Só a leitura do original passa pelo ctxReader: ela dita o ritmo e o
	// cancelamento; a do arquivo novo acompanha.
	ca := &ctxReader{ctx: ctx, r: a, total: sa.Size(), stage: StageVerify, progress: progress}
	ra, rb := bufio.NewReaderSize(ca, chunk), bufio.NewReaderSize(b, chunk)
	ba, bb := make([]byte, chunk), make([]byte, chunk)
	var pos int64
	for {
		na, ea := io.ReadFull(ra, ba)
		nb, eb := io.ReadFull(rb, bb)
		if err := ctx.Err(); err != nil {
			return err
		}
		if na != nb {
			return fmt.Errorf("verificação: leitura divergente em %d", pos)
		}
		for i := 0; i < na; i++ {
			if ba[i] != bb[i] && !allowed(pos+int64(i)) {
				return fmt.Errorf("verificação: byte %d mudou fora dos campos de GPS", pos+int64(i))
			}
		}
		pos += int64(na)
		if ea == io.EOF || ea == io.ErrUnexpectedEOF {
			return nil
		}
		if ea != nil {
			return ea
		}
		if eb != nil && eb != io.EOF && eb != io.ErrUnexpectedEOF {
			return eb
		}
	}
}

// copyFileInfo dá ao arquivo novo as permissões do original. A data de
// modificação é a do original só se keepDate; senão é now, para que a nuvem da
// GoPro (e sincronizadores) vejam um arquivo novo, não uma duplicata.
func copyFileInfo(orig, out string, keepDate bool, now time.Time) error {
	st, err := os.Stat(orig)
	if err != nil {
		return err
	}
	if err := os.Chmod(out, st.Mode().Perm()); err != nil {
		return err
	}
	if keepDate {
		return os.Chtimes(out, st.ModTime(), st.ModTime())
	}
	return os.Chtimes(out, now, now)
}
