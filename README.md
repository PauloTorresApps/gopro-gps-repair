# gopro-gps-repair

*[English version below](#english)*

Em atividades como mountain bike ou trilhas sob cobertura de árvores, o GPS de
câmeras de ação (como a GoPro Hero 7 Black) frequentemente sofre perdas de
sinal, gerando falhas nos metadados e impedindo a inserção dos medidores de
telemetria no GoPro Quik.

Esta aplicação em Go foi desenvolvida para solucionar esse problema. O software
lê o fluxo de metadados do arquivo de vídeo original, detecta lacunas nas
coordenadas e reconstrói os pontos faltantes por meio de interpolação
matemática, restaurando a integridade dos dados para uso no ecossistema da
GoPro.

Diagnóstico e recuperação dos pontos GPS em vídeos gravados por câmeras GoPro.

A GoPro grava a telemetria (GPS, acelerômetro, giroscópio...) dentro do próprio
MP4, numa trilha `gpmd` no formato GPMF. Quando a câmera começa a gravar antes de
o GPS estabilizar, ou grava com poucos satélites, a trilha fica com trechos sem
fix, fix só 2D, precisão ruim (DOP alto), saltos de posição e altitude oscilando.
Este programa mede esses defeitos e gera um **novo** MP4 com o GPS reconstruído.

Ele recupera vídeos cujo GPS foi gravado mal ou ficou degradado, para que a
telemetria (mapa, velocidade, altitude) possa ser adicionada ao vídeo pelo app
**Quik da GoPro**. O vídeo reparado mantém os metadados originais da GoPro.

**Resultado na prática:** depois de reparar um vídeo original com GPS
degradado, o app Quik da GoPro conseguiu ler os dados de GPS do arquivo novo e
adicionar as figurinhas de telemetria (mapa, velocidade, altitude etc.).

A matemática usada (filtro de Kalman, suavizador RTS, rejeição de saltos etc.)
está descrita em [docs/tecnicas-matematicas.md](docs/tecnicas-matematicas.md)
(em português).

## Uso

### Pré-requisitos

- Go 1.25 ou mais recente (só para compilar);
- vídeo de câmera GoPro com GPS gravado no formato `GPS5` (HERO5 a HERO10).

Não é preciso ffmpeg nem outra dependência: o programa lê e grava o MP4 sozinho.
A interface gráfica tem requisitos próprios (veja [Interface gráfica](#interface-gráfica)).

### Compilação

```sh
make build    # gera bin/gopro-gps-repair
make test     # roda os testes
```

Os comandos também aceitam os nomes em inglês `diagnose` e `repair`.
`bin/gopro-gps-repair versao` (ou `-v`) mostra a versão, que também aparece
no cabeçalho da interface gráfica.

### Interface gráfica

Além da linha de comando, há um app de janela (Wails v3, React e Tailwind) em
`gui/`, com textos em inglês (padrão), português, espanhol e mandarim. O fluxo:

1. escolha o vídeo (botão ou arrastar o MP4 para a janela) e clique em
   **Restaurar dados de GPS**;
2. o app faz a avaliação do GPS (a mesma do `diagnosticar`) e mostra a nota, a
   qualidade ao longo do vídeo e os trechos com problema;
3. confirme a restauração ou aborte; durante a restauração também há um botão
   de abortar, e nesse caso nenhum arquivo fica gravado;
4. ao final, um relatório mostra o que foi restaurado (posições e altitudes
   `corrigido`/`estimado`, medições rejeitadas, campos regravados) e como.

O vídeo restaurado e o relatório JSON são gravados na pasta
`~/gopro-gps-repair` (no diretório pessoal do usuário), com nome e data de
criação novos. O app usa o mesmo código do `reparar`, inclusive a verificação
byte a byte.

Para compilar são necessários, além do Go, o Node.js com npm, a CLI do Wails v3
(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16`) e, no
Linux, gcc e os pacotes de desenvolvimento do GTK 4 e do WebKitGTK 6
(`wails3 doctor` lista o que falta).

```sh
make gui        # gera gui/bin/gopro-gps-repair-gui
make gui-dev    # modo de desenvolvimento, com recarga automática
```

### Diagnosticar

Avalia o GPS do vídeo sem alterar nada.

```sh
bin/gopro-gps-repair diagnosticar ~/Downloads/GH019736.MP4 [-blocos] [-json]

# ou, pelo Makefile (já inclui -blocos):
make diagnosticar VIDEO=~/Downloads/GH019736.MP4
```

| opção | |
|---|---|
| `-blocos` | lista cada bloco do GPS (~1 s): tempo no vídeo, horário UTC, fix, DOP, amostras, saltos e qualidade |
| `-json` | imprime o diagnóstico completo em JSON, para uso por outros programas |

Exemplo de saída:

```text
Arquivo: /home/usuario/Downloads/GH019736.MP4
GPS: 2025-08-16 20:18:09 → 20:18:32 UTC (22,5 s), 23 blocos, 407 amostras
Fix: 3D 382 · 2D 18 · sem fix 7
DOP mediano: 7,28 (amostras com DOP > 5: 382)
Saltos de posição: 2 · saltos de altitude: 1 (maior: 34,2 m)
Distância com fix 3D: 62 m

Nota: 48/100 (ruim)
Problemas (tempo do vídeo):
  • sem fix: 0,0–1,0 s
  • só fix 2D (altitude não confiável): 1,0–2,0 s
  • precisão ruim (DOP > 5): 2,0–23,0 s
  • saltos de posição: 1,0–2,0 s
  • saltos de altitude: 1,0–2,0 s
```

**Como ler a nota.** Cada amostra recebe um peso: 1 com fix 3D e DOP até 2;
0,8 com DOP até 5; 0,5 com DOP até 10; 0,2 com DOP acima de 10; 0,3 com fix
só 2D; 0 sem fix ou em salto de posição. Saltos de altitude cortam o peso pela
metade. A nota é a média dos pesos × 100:

| nota | veredito | o que fazer |
|---|---|---|
| 85–100 | bom | GPS confiável; reparar é opcional |
| 50–84 | degradado | utilizável, mas o reparo é recomendado |
| 0–49 | ruim | reparar antes de usar |

Os trechos em "Problemas" estão em **tempo do vídeo** (segundos desde o início
da gravação), para localizar o defeito no editor.

### Reparar

```sh
bin/gopro-gps-repair reparar ~/Downloads/GH019736.MP4 [-o saida.MP4] [-gpx trilha.gpx]

# ou, pelo Makefile:
make reparar VIDEO=~/Downloads/GH019736.MP4 ARGS="-gpx trilha.gpx"
```

Opções do `reparar`:

| opção | padrão | |
|---|---|---|
| `-o` | `<nome>_gps_corrigido.MP4` | arquivo de saída (nunca sobrescreve) |
| `-gpx` | — | exporta também a trilha reconstruída em GPX |
| `-relatorio` | `<saída>.relatorio.json` | relatório com antes/depois e cada ponto |
| `-dop` | `2.0` | DOP declarado nos blocos com medição corrigida |
| `-dop-estimado` | `4.0` | DOP declarado nos blocos majoritariamente estimados |
| `-manter-horarios` | não | não regrava o GPSU de blocos com horário inconsistente |
| `-manter-data` | não | mantém a data do original; sem ela o nome de saída ganha data/hora (`_gps_corrigido_AAAAMMDD_HHMMSS`) e o arquivo recebe data de criação nova |

As opções podem vir antes ou depois do nome do vídeo.

Exemplo de saída:

```text
Arquivo reparado: /home/usuario/Downloads/GH019736_gps_corrigido.MP4

Nota do GPS: 48/100 (ruim) → 100/100 (bom, pelo que o arquivo declara)
Posição:  400 de 407 amostras corrigidas, 7 estimadas (0 medições rejeitadas como salto)
Altitude: 382 de 407 amostras corrigidas, 25 estimadas (0 medições rejeitadas)
Maior correção de posição: 10,8 m · incerteza mediana: 2,3 m
Altitude: faixa 357–371 m (original 353–387 m)
Regravados: 23 blocos, 8278 bytes; fix ajustado em 2, DOP em 22, horário em 0
Verificado: fora dos campos de GPS, idêntico ao original (bytes, data de modificação e permissões)
Relatório: /home/usuario/Downloads/GH019736_gps_corrigido.relatorio.json
GPX: /home/usuario/Downloads/GH019736_gps_corrigido.gpx
```

A nota "depois" é a que o arquivo novo **declara** (fix 3D e o DOP pedido), não
uma nova medição. Para saber o quanto confiar em cada trecho, veja o status e a
incerteza de cada ponto no relatório.

### Reprocessar um vídeo

O programa nunca sobrescreve arquivos. Se o arquivo de saída já existir, ele
para com o erro `... já existe; apague-o ou use -o`. Para gerar de novo, apague
os arquivos da versão anterior (vídeo, relatório e GPX) ou escolha outro nome
com `-o`.

### Arquivos gerados

| arquivo | conteúdo |
|---|---|
| `<nome>_gps_corrigido.MP4` | o vídeo com o GPS reconstruído; todo o resto idêntico ao original |
| `<nome>_gps_corrigido.relatorio.json` | relatório do reparo (sempre gerado) |
| arquivo do `-gpx` | a trilha reconstruída, para ferramentas de mapa ou de overlay que aceitem GPX |

O **relatório JSON** tem:

- `entrada` e `saida`: caminhos dos arquivos;
- `antes` e `depois`: o diagnóstico completo do original e do arquivo novo;
- `reconstrucao`: resumo (posições e altitudes estimadas, medições rejeitadas,
  maior correção, incerteza mediana);
- `gravacao`: o que foi regravado (blocos, bytes, fix, DOP, horários);
- `dop_declarado`: os valores de `-dop` e `-dop-estimado` usados;
- `pontos`: uma entrada por amostra GPS (~18 por segundo), como esta:

```json
{
  "horario": "2025-08-16T20:18:15.739Z",
  "lat": -10.169893965,
  "lon": -48.279545072,
  "alt": 367.35,
  "velocidade_2d": 2.31,
  "velocidade_3d": 2.41,
  "status_posicao": "corrigido",
  "status_altitude": "corrigido",
  "incerteza_posicao_m": 1.83,
  "incerteza_altitude_m": 3.58,
  "correcao_m": 1.68
}
```

Velocidades em m/s; altitude, incertezas e `correcao_m` (distância até a
posição original) em metros. Incerteza `-1` significa desconhecida.

O **GPX** tem um `<trkpt>` por amostra, com altitude e horário UTC; o status de
cada ponto vai no `<desc>` (por exemplo, `posição corrigido, altitude estimado`).

## Como funciona

1. **Leitura** (`internal/mp4`, `internal/gpmf`, `internal/telemetry`): localiza
   as amostras `gpmd` pelo índice do MP4 e decodifica os blocos GPS (`GPS5`,
   `GPSF`, `GPSP`, `GPSU`, `SCAL`), guardando a posição de cada campo no arquivo.
   O horário de cada bloco é conferido contra o relógio do vídeo.
2. **Diagnóstico** (`internal/diagnose`): fix, DOP, saltos de posição (além do
   que a velocidade explica) e de altitude; nota ponderada por amostra.
3. **Reconstrução** (`internal/reconstruct`): filtro de Kalman de velocidade
   constante com suavizador RTS, usando a posição e a velocidade Doppler (o dado
   mais confiável do receptor). Medições incoerentes são rejeitadas, lacunas são
   interpoladas e as pontas extrapoladas por até 5 s. A altitude usa só fix 3D,
   com suavização forte.
4. **Gravação** (`internal/writer`): copia o MP4 e sobrescreve **no lugar**, com
   o mesmo tamanho, só os campos de GPS. Vídeo, áudio, demais sensores e
   metadados ficam idênticos ao original.
5. **Verificação**: o arquivo novo é comparado byte a byte com o original. Se
   qualquer byte fora dos campos de GPS tiver mudado, ou o tamanho for outro, o
   arquivo novo é apagado e o reparo falha. O arquivo novo recebe um **nome e
   uma data/hora de criação novos** (data de modificação do arquivo e datas
   `mvhd`/`tkhd`/`mdhd` do contêiner MP4), para a nuvem da GoPro não o tomar por
   duplicata do original; as permissões são as do original. Com
   `-manter-data` a data de modificação do original é mantida.

### Metadados preservados

Os metadados da GoPro ou de qualquer outra fonte (firmware,
número de série da câmera, timecode, configurações de gravação, acelerômetro,
giroscópio, metadados adicionados por outros programas etc.) ficam **idênticos**
ao original. Só mudam os campos de GPS: `GPS5` (posição, altitude e
velocidades), `GPSF` (fix), `GPSP` (DOP) e, nos blocos com horário
inconsistente, `GPSU` (desligável com `-manter-horarios`). As únicas outras
mudanças são o nome do arquivo e as datas de criação/modificação do contêiner
(desligáveis com `-manter-data`).

Cada ponto do relatório é marcado como `corrigido` (havia medição) ou
`estimado` (não havia), com a incerteza em metros.

## Honestidade dos dados

O arquivo reparado declara fix 3D e um DOP melhor que o original, para que os
programas de edição aceitem a trilha. Os valores são reconstruídos, não
medidos: guarde o original e o relatório JSON. A nota "depois" do `reparar`
reflete o que o arquivo declara, não uma nova medição.

## Pacotes e lojas

Nas lojas, o app aparece como **GPS Repair for GoPro** (projeto independente,
sem vínculo com a GoPro, Inc.). Todos os pacotes trazem a interface
(`gopro-gps-repair-gui`) e a linha de comando (`gopro-gps-repair`).

| distribuição | formato | onde fica |
|---|---|---|
| Ubuntu 24.04+, Debian 13+ e derivados | `.deb` | `make pacotes` → `gui/bin/` |
| Fedora, RHEL 10 e derivados | `.rpm` | `make pacotes` → `gui/bin/` |
| Arch, Manjaro, EndeavourOS... | `.pkg.tar.zst` e AUR | `make pacotes`; `packaging/aur/PKGBUILD` |
| qualquer distro | AppImage | `make pacotes` → `gui/bin/` |
| Flathub (GNOME Software, KDE Discover; Fedora, Mint, Pop!_OS...) | Flatpak | `packaging/flatpak/` (`make flatpak`) |
| Snap Store (App Center do Ubuntu) | Snap | `snap/snapcraft.yaml` |

`make pacotes` gera os quatro primeiros. O identificador do app
(AppStream, `.desktop`, Flatpak) é `io.github.PauloTorresApps.gopro_gps_repair`;
os metadados das lojas ficam em `gui/build/linux/share/`.

**Publicar uma versão.** O workflow `.github/workflows/packages.yml` gera todos
os pacotes (Linux, Flatpak, Snap, Windows e macOS) e cria uma Release em
rascunho no GitHub quando uma tag `vX.Y.Z` é enviada. A versão fica só no
arquivo `VERSION`, na raiz: os pacotes Linux, o Snap, o macOS e o Windows a
leem dele. Ao lançar uma versão, mude o `VERSION` e escreva à mão as duas
cópias que as lojas exigem: a entrada nova em `<releases>` no metainfo (com
data e novidades) e o `pkgver` do `packaging/aur/PKGBUILD` e do `.SRCINFO`.
`make conferir-versao` (também rodado pelo CI) acusa o que faltar, e o CI
recusa uma tag diferente do `VERSION`.

- **Flathub:** o Flathub compila sem internet; as dependências Go e npm ficam
  listadas em `packaging/flatpak/go-sources.json` e `npm-sources.json`. Depois de
  mudar `gui/go.mod` ou `package-lock.json`, rode `make fontes-flatpak`. Para
  enviar, siga <https://docs.flathub.org/docs/for-app-authors/submission>,
  trocando a fonte `dir` do manifesto pela tag do GitHub (há um comentário no
  arquivo).
- **AUR:** depois de criar a tag, rode `updpkgsums` e
  `makepkg --printsrcinfo > .SRCINFO` em `packaging/aur/` e envie os dois
  arquivos ao repositório `gopro-gps-repair` do AUR.
- **Snap Store:** registre o nome (`snapcraft register gopro-gps-repair`) e
  envie o `.snap` gerado pelo workflow (`snapcraft upload --release=stable`).

## macOS e Windows

As tarefas de build do Wails para os dois sistemas já estão em
`gui/build/darwin` e `gui/build/windows`. Em cada sistema, instale Go 1.25+,
Node.js 22+ e a CLI do Wails
(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16`), e rode na
pasta `gui/`:

| sistema | requisitos extras | comando | resultado |
|---|---|---|---|
| macOS 11+ | Xcode Command Line Tools (`xcode-select --install`) | `wails3 task darwin:package:universal` | `bin/gopro-gps-repair-gui.app` (Intel e Apple Silicon) |
| Windows 10/11 | NSIS (`winget install NSIS.NSIS`), para o instalador | `wails3 task windows:package` | instalador em `bin/` |

No Windows o app usa o WebView2, que já vem no Windows 11; o instalador o
instala no Windows 10 se faltar. O app do macOS sai sem assinatura: na primeira
vez, abra com o botão direito → **Abrir**. Para assinar e notarizar, veja
`wails3 task darwin:sign:notarize` e `wails3 setup signing`.

## Limitações atuais

- só o formato `GPS5` (HERO5 a HERO10); o `GPS9` da HERO11+ ainda não;
- cada arquivo é tratado isoladamente (sem juntar GH01…, GH02… da mesma gravação);
- altitude só a partir do próprio GPS (sem modelo de terreno);
- sem fontes externas (Strava, GPX/FIT) e sem uso do acelerômetro/giroscópio.

## Licença

Apache License 2.0. Veja o arquivo [LICENSE](LICENSE).

---

<a id="english"></a>

# gopro-gps-repair (English)

In activities such as mountain biking or trails under tree cover, the GPS of
action cameras (such as the GoPro Hero 7 Black) often loses signal, causing
gaps in the metadata and preventing the telemetry gauges from being added in
GoPro Quik.

This Go application was built to solve that problem. The software reads the
metadata stream of the original video file, detects gaps in the coordinates and
rebuilds the missing points through mathematical interpolation, restoring the
integrity of the data for use in the GoPro ecosystem.

Diagnosis and recovery of the GPS recorded by GoPro cameras.

GoPro cameras record telemetry (GPS, accelerometer, gyroscope...) inside the MP4
itself, in a `gpmd` track using the GPMF format. When the camera starts recording
before the GPS has settled, or records with few satellites, the track ends up
with stretches without a fix, 2D-only fix, poor precision (high DOP), position
jumps and oscillating altitude. This program measures these defects and creates
a **new** MP4 with the reconstructed GPS.

It recovers videos whose GPS was poorly recorded or degraded, so that the
telemetry (map, speed, altitude) can be added to the video with the **GoPro
Quik** app. The repaired video keeps the original GoPro metadata.

**Result in practice:** after repairing an original video with degraded GPS,
the GoPro Quik app was able to read the GPS data in the new file and add the
telemetry stickers (map, speed, altitude, etc.).

The mathematics used (Kalman filter, RTS smoother, jump rejection, etc.) is
described in [docs/tecnicas-matematicas.md](docs/tecnicas-matematicas.md)
(in Portuguese).

## Usage

### Requirements

- Go 1.25 or newer (only to build);
- a video from a GoPro camera with GPS recorded in the `GPS5` format (HERO5 to HERO10).

No ffmpeg or other dependency is needed: the program reads and writes the MP4 by itself.
The graphical interface has its own requirements (see [Graphical interface](#graphical-interface)).

### Building

```sh
make build    # creates bin/gopro-gps-repair
make test     # runs the tests
```

The commands also accept the English aliases `diagnose` and `repair`.
`bin/gopro-gps-repair versao` (or `-v`) prints the version, which is also shown
in the header of the graphical interface.

### Graphical interface

Besides the command line, there is a desktop app (Wails v3, React and Tailwind)
in `gui/`, available in English (default), Portuguese, Spanish and Mandarin.
The flow:

1. choose the video (button, or drag the MP4 onto the window) and click
   **Restore GPS data**;
2. the app assesses the GPS (the same assessment as `diagnosticar`) and shows
   the score, the quality along the video and the problem stretches;
3. confirm the restoration or abort; there is also an abort button during the
   restoration, and if it is used no file is left behind;
4. at the end, a report shows what was restored (`corrigido`/`estimado`
   positions and altitudes, rejected measurements, rewritten fields) and how.

The restored video and the JSON report are saved to the `~/gopro-gps-repair`
folder (in the user's home directory), with a new name and creation date. The
app uses the same code as `reparar`, including the byte-by-byte verification.

Building it requires, besides Go, Node.js with npm, the Wails v3 CLI
(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16`) and, on
Linux, gcc and the GTK 4 and WebKitGTK 6 development packages (`wails3 doctor`
lists what is missing).

```sh
make gui        # creates gui/bin/gopro-gps-repair-gui
make gui-dev    # development mode, with live reload
```

### Diagnosing

Evaluates the video's GPS without changing anything.

```sh
bin/gopro-gps-repair diagnosticar ~/Downloads/GH019736.MP4 [-blocos] [-json]

# or, through the Makefile (already includes -blocos):
make diagnosticar VIDEO=~/Downloads/GH019736.MP4
```

| option | |
|---|---|
| `-blocos` | lists each GPS block (~1 s): video time, UTC time, fix, DOP, samples, jumps and quality |
| `-json` | prints the full diagnosis as JSON, for use by other programs |

Example output (the program prints in Portuguese):

```text
Arquivo: /home/usuario/Downloads/GH019736.MP4
GPS: 2025-08-16 20:18:09 → 20:18:32 UTC (22,5 s), 23 blocos, 407 amostras
Fix: 3D 382 · 2D 18 · sem fix 7
DOP mediano: 7,28 (amostras com DOP > 5: 382)
Saltos de posição: 2 · saltos de altitude: 1 (maior: 34,2 m)
Distância com fix 3D: 62 m

Nota: 48/100 (ruim)
Problemas (tempo do vídeo):
  • sem fix: 0,0–1,0 s
  • só fix 2D (altitude não confiável): 1,0–2,0 s
  • precisão ruim (DOP > 5): 2,0–23,0 s
  • saltos de posição: 1,0–2,0 s
  • saltos de altitude: 1,0–2,0 s
```

**Reading the score.** Each sample gets a weight: 1 with a 3D fix and DOP up to
2; 0.8 with DOP up to 5; 0.5 with DOP up to 10; 0.2 with DOP above 10; 0.3 with
a 2D-only fix; 0 without a fix or in a position jump. Altitude jumps cut the
weight in half. The score is the average weight × 100:

| score | verdict | what to do |
|---|---|---|
| 85–100 | `bom` (good) | reliable GPS; repairing is optional |
| 50–84 | `degradado` (degraded) | usable, but repairing is recommended |
| 0–49 | `ruim` (poor) | repair before using |

The stretches under "Problemas" (problems) are in **video time** (seconds since
the start of the recording), so the defect can be found in the editor.

### Repairing

```sh
bin/gopro-gps-repair reparar ~/Downloads/GH019736.MP4 [-o output.MP4] [-gpx track.gpx]

# or, through the Makefile:
make reparar VIDEO=~/Downloads/GH019736.MP4 ARGS="-gpx track.gpx"
```

`reparar` options:

| option | default | |
|---|---|---|
| `-o` | `<name>_gps_corrigido.MP4` | output file (never overwrites) |
| `-gpx` | — | also exports the reconstructed track as GPX |
| `-relatorio` | `<output>.relatorio.json` | report with before/after and every point |
| `-dop` | `2.0` | DOP declared in blocks with corrected measurements |
| `-dop-estimado` | `4.0` | DOP declared in mostly estimated blocks |
| `-manter-horarios` | no | does not rewrite the GPSU of blocks with inconsistent time |
| `-manter-data` | no | keeps the original's date; without it the output name gets a timestamp (`_gps_corrigido_YYYYMMDD_HHMMSS`) and the file gets a new creation date |

Options can come before or after the video name.

Example output:

```text
Arquivo reparado: /home/usuario/Downloads/GH019736_gps_corrigido.MP4

Nota do GPS: 48/100 (ruim) → 100/100 (bom, pelo que o arquivo declara)
Posição:  400 de 407 amostras corrigidas, 7 estimadas (0 medições rejeitadas como salto)
Altitude: 382 de 407 amostras corrigidas, 25 estimadas (0 medições rejeitadas)
Maior correção de posição: 10,8 m · incerteza mediana: 2,3 m
Altitude: faixa 357–371 m (original 353–387 m)
Regravados: 23 blocos, 8278 bytes; fix ajustado em 2, DOP em 22, horário em 0
Verificado: fora dos campos de GPS, idêntico ao original (bytes, data de modificação e permissões)
Relatório: /home/usuario/Downloads/GH019736_gps_corrigido.relatorio.json
GPX: /home/usuario/Downloads/GH019736_gps_corrigido.gpx
```

The "after" score is the one the new file **declares** (3D fix and the requested
DOP), not a new measurement. To know how much to trust each stretch, check each
point's status and uncertainty in the report.

### Reprocessing a video

The program never overwrites files. If the output file already exists, it stops
with the error `... já existe; apague-o ou use -o` ("already exists; delete it
or use -o"). To generate it again, delete the previous files (video, report and
GPX) or choose another name with `-o`.

### Generated files

| file | content |
|---|---|
| `<name>_gps_corrigido.MP4` | the video with the reconstructed GPS; everything else identical to the original |
| `<name>_gps_corrigido.relatorio.json` | repair report (always created) |
| the `-gpx` file | the reconstructed track, for map or overlay tools that accept GPX |

The **JSON report** has:

- `entrada` and `saida`: input and output paths;
- `antes` and `depois`: the full diagnosis of the original and of the new file;
- `reconstrucao`: summary (estimated positions and altitudes, rejected
  measurements, largest correction, median uncertainty);
- `gravacao`: what was rewritten (blocks, bytes, fix, DOP, times);
- `dop_declarado`: the `-dop` and `-dop-estimado` values used;
- `pontos`: one entry per GPS sample (~18 per second), like this one:

```json
{
  "horario": "2025-08-16T20:18:15.739Z",
  "lat": -10.169893965,
  "lon": -48.279545072,
  "alt": 367.35,
  "velocidade_2d": 2.31,
  "velocidade_3d": 2.41,
  "status_posicao": "corrigido",
  "status_altitude": "corrigido",
  "incerteza_posicao_m": 1.83,
  "incerteza_altitude_m": 3.58,
  "correcao_m": 1.68
}
```

Speeds in m/s; altitude, uncertainties and `correcao_m` (distance to the
original position) in meters. An uncertainty of `-1` means unknown.

The **GPX** has one `<trkpt>` per sample, with altitude and UTC time; each
point's status goes in `<desc>` (for example, `posição corrigido, altitude
estimado`, meaning corrected position, estimated altitude).

## How it works

1. **Reading** (`internal/mp4`, `internal/gpmf`, `internal/telemetry`): locates
   the `gpmd` samples through the MP4 index and decodes the GPS blocks (`GPS5`,
   `GPSF`, `GPSP`, `GPSU`, `SCAL`), keeping the position of each field in the
   file. Each block's time is checked against the video clock.
2. **Diagnosis** (`internal/diagnose`): fix, DOP, position jumps (beyond what
   the speed explains) and altitude jumps; score weighted per sample.
3. **Reconstruction** (`internal/reconstruct`): constant-velocity Kalman filter
   with an RTS smoother, using position and Doppler speed (the receiver's most
   reliable data). Inconsistent measurements are rejected, gaps are
   interpolated and the ends are extrapolated for up to 5 s. Altitude uses only
   3D fixes, with strong smoothing.
4. **Writing** (`internal/writer`): copies the MP4 and overwrites **in place**,
   with the same size, only the GPS fields. Video, audio, other sensors and
   metadata stay identical to the original.
5. **Verification**: the new file is compared byte by byte with the original.
   If any byte outside the GPS fields has changed, or the size differs, the new
   file is deleted and the repair fails. The new file gets a **new name and
   creation date/time** (file modification time and the container's
   `mvhd`/`tkhd`/`mdhd` dates), so GoPro's cloud does not take it for a
   duplicate of the original; permissions are the original's. With
   `-manter-data` the original's modification time is kept.

### Preserved metadata

Metadata from GoPro or any other source (firmware, camera
serial number, timecode, recording settings, accelerometer, gyroscope, metadata
added by other software, etc.) stays **identical** to the original. Only the
GPS fields change: `GPS5` (position, altitude and speeds), `GPSF` (fix),
`GPSP` (DOP) and, in blocks with inconsistent time, `GPSU` (can be disabled
with `-manter-horarios`). The only other changes are the file name and the
container's creation/modification dates (can be disabled with `-manter-data`).

Each point in the report is marked as `corrigido` (corrected: there was a
measurement) or `estimado` (estimated: there was none), with the uncertainty in
meters.

## Data honesty

The repaired file declares a 3D fix and a better DOP than the original, so that
editing software accepts the track. The values are reconstructed, not
measured: keep the original and the JSON report. The "after" score shown by
`reparar` reflects what the file declares, not a new measurement.

## Packages and stores

In the stores, the app is listed as **GPS Repair for GoPro** (an independent
project, not affiliated with GoPro, Inc.). Every package ships both the
graphical app (`gopro-gps-repair-gui`) and the command line
(`gopro-gps-repair`).

| distribution | format | where |
|---|---|---|
| Ubuntu 24.04+, Debian 13+ and derivatives | `.deb` | `make pacotes` → `gui/bin/` |
| Fedora, RHEL 10 and derivatives | `.rpm` | `make pacotes` → `gui/bin/` |
| Arch, Manjaro, EndeavourOS... | `.pkg.tar.zst` and AUR | `make pacotes`; `packaging/aur/PKGBUILD` |
| any distro | AppImage | `make pacotes` → `gui/bin/` |
| Flathub (GNOME Software, KDE Discover; Fedora, Mint, Pop!_OS...) | Flatpak | `packaging/flatpak/` (`make flatpak`) |
| Snap Store (Ubuntu App Center) | Snap | `snap/snapcraft.yaml` |

`make pacotes` builds the first four. The app ID (AppStream,
`.desktop`, Flatpak) is `io.github.PauloTorresApps.gopro_gps_repair`; the store
metadata lives in `gui/build/linux/share/`.

**Publishing a release.** The `.github/workflows/packages.yml` workflow builds
every package (Linux, Flatpak, Snap, Windows and macOS) and creates a draft
GitHub Release when a `vX.Y.Z` tag is pushed. The version lives only in the
`VERSION` file at the repository root: the Linux packages, the Snap, macOS and
Windows read it from there. When releasing, change `VERSION` and hand-write the
two copies the stores require: a new `<releases>` entry in the metainfo (with
date and changes) and the `pkgver` in `packaging/aur/PKGBUILD` and `.SRCINFO`.
`make conferir-versao` (also run by CI) reports anything missing, and CI
rejects a tag that differs from `VERSION`.

- **Flathub:** Flathub builds without internet access; the Go and npm
  dependencies are listed in `packaging/flatpak/go-sources.json` and
  `npm-sources.json`. After changing `gui/go.mod` or `package-lock.json`, run
  `make fontes-flatpak`. To submit, follow
  <https://docs.flathub.org/docs/for-app-authors/submission>, replacing the
  manifest's `dir` source with the GitHub tag (there is a comment in the file).
- **AUR:** after tagging, run `updpkgsums` and `makepkg --printsrcinfo > .SRCINFO`
  in `packaging/aur/` and push both files to the AUR `gopro-gps-repair`
  repository.
- **Snap Store:** register the name (`snapcraft register gopro-gps-repair`) and
  upload the `.snap` built by the workflow (`snapcraft upload --release=stable`).

## macOS and Windows

The Wails build tasks for both systems are in `gui/build/darwin` and
`gui/build/windows`. On each system, install Go 1.25+, Node.js 22+ and the
Wails CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16`),
then run in the `gui/` folder:

| system | extra requirements | command | result |
|---|---|---|---|
| macOS 11+ | Xcode Command Line Tools (`xcode-select --install`) | `wails3 task darwin:package:universal` | `bin/gopro-gps-repair-gui.app` (Intel and Apple Silicon) |
| Windows 10/11 | NSIS (`winget install NSIS.NSIS`), for the installer | `wails3 task windows:package` | installer in `bin/` |

On Windows the app uses WebView2, which ships with Windows 11; the installer
adds it on Windows 10 if missing. The macOS app is unsigned: the first time,
right-click it → **Open**. To sign and notarize, see
`wails3 task darwin:sign:notarize` and `wails3 setup signing`.

## Current limitations

- only the `GPS5` format (HERO5 to HERO10); HERO11+ `GPS9` is not supported yet;
- each file is handled on its own (GH01…, GH02… of the same recording are not joined);
- altitude comes only from the GPS itself (no terrain model);
- no external sources (Strava, GPX/FIT) and no use of the accelerometer/gyroscope.

## License

Apache License 2.0. See the [LICENSE](LICENSE) file.
