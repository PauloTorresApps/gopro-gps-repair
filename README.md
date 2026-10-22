# gopro-gps-repair

Diagnóstico e recuperação do GPS gravado por câmeras GoPro.

A GoPro grava a telemetria (GPS, acelerômetro, giroscópio...) dentro do próprio
MP4, numa trilha `gpmd` no formato GPMF. Quando a câmera começa a gravar antes de
o GPS estabilizar, ou grava com poucos satélites, a trilha fica com trechos sem
fix, fix só 2D, precisão ruim (DOP alto), saltos de posição e altitude oscilando.
Este programa mede esses defeitos e gera um **novo** MP4 com o GPS reconstruído.

## Uso

```sh
make build

# Diagnóstico: nota de 0 a 100 e trechos com problema (tempo do vídeo)
bin/gopro-gps-repair diagnosticar ~/Downloads/GH019736.MP4 [-blocos] [-json]

# Reparo: grava <nome>_gps_corrigido.MP4 ao lado do original
bin/gopro-gps-repair reparar ~/Downloads/GH019736.MP4 [-o saida.MP4] [-gpx trilha.gpx]
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

Cada ponto do relatório é marcado como `corrigido` (havia medição) ou
`estimado` (não havia), com a incerteza em metros.

## Honestidade dos dados

O arquivo reparado declara fix 3D e um DOP melhor que o original, para que os
programas de edição aceitem a trilha. Os valores são reconstruídos, não
medidos: guarde o original e o relatório JSON. A nota "depois" do `reparar`
reflete o que o arquivo declara, não uma nova medição.

## Limitações atuais

- só o formato `GPS5` (HERO5 a HERO10); o `GPS9` da HERO11+ ainda não;
- cada arquivo é tratado isoladamente (sem juntar GH01…, GH02… da mesma gravação);
- altitude só a partir do próprio GPS (sem modelo de terreno);
- sem fontes externas (Strava, GPX/FIT) e sem uso do acelerômetro/giroscópio.
