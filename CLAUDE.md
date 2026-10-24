# CLAUDE.md

## README bilíngue

O `README.md` é mantido em **português e inglês**. Toda edição dele deve ser
feita nos dois idiomas:

- o texto em português fica no topo e nunca é apagado;
- a tradução em inglês fica abaixo da âncora `<a id="english"></a>`;
- as duas metades seguem a mesma ordem de seções;
- nomes de comandos, opções, arquivos e status do programa (`diagnosticar`,
  `-blocos`, `corrigido`, `estimado`...) não são traduzidos.

## Comandos

Use os alvos do `Makefile`:

- `make build`: compila em `bin/gopro-gps-repair`;
- `make test` e `make vet`;
- `make diagnosticar VIDEO=<arquivo>`;
- `make reparar VIDEO=<arquivo> [ARGS="..."]`.

## Convenções

- Comentários, mensagens e saída da linha de comando são em português.
- O arquivo de entrada nunca é alterado: o reparo grava um MP4 novo e só
  sobrescreve, no lugar e com o mesmo tamanho, os campos de GPS.
- **Metadados intocáveis:** os vídeos corrigidos devem manter os metadados da
  GoPro ou de qualquer outra fonte idênticos ao original. Só os campos de GPS
  (`GPS5`, `GPSF`, `GPSP`, `GPSU`) podem mudar. Nada de remux (ffmpeg etc.),
  reescrita de boxes do MP4 ou mudança de tamanho. A verificação byte a byte
  (`writer.VerifyOnlyPatched`) e a cópia de data/permissões
  (`copyFileInfo`) não podem ser removidas nem contornadas.
- **Nome e data novos:** a única exceção são o nome do arquivo e as datas de
  criação/modificação (arquivo e `mvhd`/`tkhd`/`mdhd`), que ficam novos por
  padrão para a nuvem da GoPro não ver o reparo como duplicata. Essas datas
  entram na lista de trechos aceitos pela verificação; `-manter-data` as
  preserva.
- Valores reconstruídos são sempre marcados como `corrigido` ou `estimado` no
  relatório. Não declarar no arquivo uma precisão melhor do que a opção pedida.
