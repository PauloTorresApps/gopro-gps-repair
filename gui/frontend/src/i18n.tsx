import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";

// Idiomas da interface. O inglês é o padrão.
export const languages = [
  { code: "en", label: "English", locale: "en-US" },
  { code: "pt", label: "Português", locale: "pt-BR" },
  { code: "es", label: "Español", locale: "es-ES" },
  { code: "zh", label: "中文（简体）", locale: "zh-CN" },
] as const;

export type Lang = (typeof languages)[number]["code"];

const en = {
  "app.tagline": "Recover degraded GPS so GoPro Quik can add telemetry gauges",
  "app.language": "Language",
  "app.version": "Version",

  "input.title": "Choose a GoPro video",
  "input.hint": "Drag an MP4 here or browse your files. The original is never changed.",
  "input.browse": "Browse…",
  "input.change": "Change video",
  "input.dialogTitle": "Choose a GoPro video",
  "input.selected": "Selected video",
  "input.process": "Restore GPS data",
  "input.output": "Restored videos are saved to {dir}",
  "input.supported": "Cameras with GPS5 telemetry (HERO5 to HERO10)",

  "diag.running": "Evaluating the GPS data…",
  "diag.title": "GPS assessment",
  "diag.subtitle": "This is the real state of the GPS recorded in the video. Nothing has been changed yet.",
  "diag.score": "Score",
  "diag.verdict.bom": "Good",
  "diag.verdict.degradado": "Degraded",
  "diag.verdict.ruim": "Poor",
  "diag.advice.bom": "The GPS is reliable. Restoring is optional, but it can still smooth the track.",
  "diag.advice.degradado": "The GPS is usable but flawed. Restoring is recommended before using it in Quik.",
  "diag.advice.ruim": "The GPS is poor. Restore it before adding gauges in Quik.",
  "diag.timeline": "Quality along the video",
  "diag.timelineHint": "Each bar is about one second of GPS",
  "diag.recorded": "Recorded",
  "diag.duration": "GPS duration",
  "diag.samples": "Samples",
  "diag.fix": "Fix (3D / 2D / none)",
  "diag.dop": "Median DOP",
  "diag.highDop": "Samples with DOP > 5",
  "diag.jumps": "Position jumps",
  "diag.altJumps": "Altitude jumps",
  "diag.maxAltJump": "Largest altitude jump",
  "diag.timeSuspect": "Blocks with inconsistent time",
  "diag.distance": "Distance with 3D fix",
  "diag.problems": "Problems found (video time)",
  "diag.noProblems": "No problems found.",
  "diag.confirmQuestion": "Restore the GPS of this video?",
  "diag.confirmNote": "A new video will be created in {dir}. Only the GPS fields change; the original stays untouched.",
  "diag.confirm": "Confirm restoration",
  "diag.abort": "Abort",

  "issue.sem_fix": "No fix",
  "issue.fix_2d": "2D fix only (altitude unreliable)",
  "issue.dop_alto": "Poor precision (DOP > 5)",
  "issue.saltos_posicao": "Position jumps",
  "issue.saltos_altitude": "Altitude jumps",
  "issue.horario_inconsistente": "Inconsistent GPS time (GPSU)",

  "run.title": "Restoring GPS data",
  "run.subtitle": "You can abort at any time. Nothing is kept if you do.",
  "run.abort": "Abort restoration",
  "run.aborting": "Aborting…",
  "stage.leitura": "Reading the telemetry",
  "stage.reconstrucao": "Rebuilding the track",
  "stage.copia": "Copying the video",
  "stage.verificacao": "Verifying byte by byte",
  "stage.releitura": "Re-reading the new file",
  "stage.relatorio": "Saving the report",

  "done.title": "GPS restored",
  "done.subtitle": "The new video is ready to import into GoPro Quik.",
  "done.file": "Restored video",
  "done.report": "Report (JSON)",
  "done.showFolder": "Show in folder",
  "done.another": "Restore another video",
  "done.scoreBefore": "Before",
  "done.scoreAfter": "After (declared)",
  "done.what": "What was restored",
  "done.positions": "Positions",
  "done.altitudes": "Altitudes",
  "done.corrected": "corrected",
  "done.estimated": "estimated",
  "done.rejected": "Measurements rejected as jumps",
  "done.rejectedValue": "{pos} positions · {alt} altitudes",
  "done.maxShift": "Largest position correction",
  "done.sigma": "Median uncertainty",
  "done.altRange": "Altitude range",
  "done.altRangeValue": "{min}–{max} m (original {omin}–{omax} m)",
  "done.written": "What was written to the file",
  "done.blocks": "GPS blocks rewritten",
  "done.fixRaised": "Blocks with fix raised to 3D",
  "done.fixMissing": "Blocks without a fix field (left as they were)",
  "done.dopChanged": "Blocks with DOP adjusted",
  "done.timesRewritten": "GPS times rewritten",
  "done.bytes": "Bytes rewritten",
  "done.created": "New creation date",
  "done.how": "How it was restored",
  "done.how1.t": "Reading",
  "done.how1.d": "The GPS blocks (GPS5, GPSF, GPSP, GPSU) were read from the video's telemetry track, and each block's time was checked against the video clock.",
  "done.how2.t": "Filtering",
  "done.how2.d": "A constant-velocity Kalman filter with an RTS smoother combined position and Doppler speed, weighting each measurement by its DOP.",
  "done.how3.t": "Jump rejection",
  "done.how3.d": "Measurements that moved farther than the speed explains were discarded ({n} positions).",
  "done.how4.t": "Gaps",
  "done.how4.d": "Stretches without a usable measurement were interpolated; the ends were extrapolated for at most 5 s ({n} estimated positions).",
  "done.how5.t": "Altitude",
  "done.how5.d": "Altitude used only 3D-fix samples, with strong smoothing ({n} estimated altitudes).",
  "done.how6.t": "In-place writing",
  "done.how6.d": "The video was copied and only the GPS fields were overwritten, with the same size. Declared DOP: {dop} for corrected blocks, {dopEst} for mostly estimated ones.",
  "done.how7.t": "Verification",
  "done.how7.d": "The new file was compared byte by byte with the original: outside the GPS fields and the creation dates, it is identical. GoPro metadata is intact.",
  "done.how8.t": "New name and date",
  "done.how8.d": "The file got a new name and creation date so the GoPro cloud does not treat it as a duplicate of the original.",
  "done.honesty": "The values are reconstructed, not measured. The \"after\" score is what the file declares, not a new measurement. Keep the original video and the JSON report, which marks every point as corrected or estimated.",
  "done.quality": "Quality along the video",

  "unit.s": "{v} s",
  "unit.m": "{v} m",

  "err.title": "Something went wrong",
  "err.dismiss": "Dismiss",
  "err.aborted": "Restoration aborted. No file was written.",
  "err.noGps5": "This video has no GPS5 track. The camera may have had GPS off, or it records GPS9 (HERO11 or newer), which is not supported yet.",
  "err.noTelemetry": "This video has no GoPro telemetry track (gpmd).",
  "err.notMp4": "The file is not an MP4 video.",
  "err.invalidMp4": "The file is not a valid MP4.",
  "err.exists": "An output file with this name already exists.",
  "err.noGpsTime": "No GPS block has a valid time (GPSU).",
  "err.busy": "A restoration is already running.",
  "err.generic": "The operation failed.",
  "err.details": "Details",
};

export type Key = keyof typeof en;
type Dict = Record<Key, string>;

const pt: Dict = {
  "app.tagline": "Recupere o GPS degradado para o GoPro Quik adicionar os medidores de telemetria",
  "app.language": "Idioma",
  "app.version": "Versão",

  "input.title": "Escolha um vídeo da GoPro",
  "input.hint": "Arraste um MP4 para cá ou procure nos seus arquivos. O original nunca é alterado.",
  "input.browse": "Procurar…",
  "input.change": "Trocar vídeo",
  "input.dialogTitle": "Escolha um vídeo da GoPro",
  "input.selected": "Vídeo escolhido",
  "input.process": "Restaurar dados de GPS",
  "input.output": "Os vídeos restaurados são gravados em {dir}",
  "input.supported": "Câmeras com telemetria GPS5 (HERO5 a HERO10)",

  "diag.running": "Avaliando os dados de GPS…",
  "diag.title": "Avaliação do GPS",
  "diag.subtitle": "Esta é a situação real do GPS gravado no vídeo. Nada foi alterado ainda.",
  "diag.score": "Nota",
  "diag.verdict.bom": "Bom",
  "diag.verdict.degradado": "Degradado",
  "diag.verdict.ruim": "Ruim",
  "diag.advice.bom": "O GPS é confiável. Restaurar é opcional, mas ainda pode suavizar a trilha.",
  "diag.advice.degradado": "O GPS é utilizável, mas tem falhas. Recomenda-se restaurar antes de usar no Quik.",
  "diag.advice.ruim": "O GPS está ruim. Restaure antes de adicionar os medidores no Quik.",
  "diag.timeline": "Qualidade ao longo do vídeo",
  "diag.timelineHint": "Cada barra é cerca de um segundo de GPS",
  "diag.recorded": "Gravado em",
  "diag.duration": "Duração do GPS",
  "diag.samples": "Amostras",
  "diag.fix": "Fix (3D / 2D / sem fix)",
  "diag.dop": "DOP mediano",
  "diag.highDop": "Amostras com DOP > 5",
  "diag.jumps": "Saltos de posição",
  "diag.altJumps": "Saltos de altitude",
  "diag.maxAltJump": "Maior salto de altitude",
  "diag.timeSuspect": "Blocos com horário inconsistente",
  "diag.distance": "Distância com fix 3D",
  "diag.problems": "Problemas encontrados (tempo do vídeo)",
  "diag.noProblems": "Nenhum problema encontrado.",
  "diag.confirmQuestion": "Restaurar o GPS deste vídeo?",
  "diag.confirmNote": "Um vídeo novo será criado em {dir}. Só os campos de GPS mudam; o original fica intacto.",
  "diag.confirm": "Confirmar restauração",
  "diag.abort": "Abortar",

  "issue.sem_fix": "Sem fix",
  "issue.fix_2d": "Só fix 2D (altitude não confiável)",
  "issue.dop_alto": "Precisão ruim (DOP > 5)",
  "issue.saltos_posicao": "Saltos de posição",
  "issue.saltos_altitude": "Saltos de altitude",
  "issue.horario_inconsistente": "Horário GPS (GPSU) inconsistente",

  "run.title": "Restaurando os dados de GPS",
  "run.subtitle": "Você pode abortar a qualquer momento. Nesse caso, nada é gravado.",
  "run.abort": "Abortar restauração",
  "run.aborting": "Abortando…",
  "stage.leitura": "Lendo a telemetria",
  "stage.reconstrucao": "Reconstruindo a trilha",
  "stage.copia": "Copiando o vídeo",
  "stage.verificacao": "Verificando byte a byte",
  "stage.releitura": "Relendo o arquivo novo",
  "stage.relatorio": "Salvando o relatório",

  "done.title": "GPS restaurado",
  "done.subtitle": "O vídeo novo está pronto para importar no GoPro Quik.",
  "done.file": "Vídeo restaurado",
  "done.report": "Relatório (JSON)",
  "done.showFolder": "Mostrar na pasta",
  "done.another": "Restaurar outro vídeo",
  "done.scoreBefore": "Antes",
  "done.scoreAfter": "Depois (declarada)",
  "done.what": "O que foi restaurado",
  "done.positions": "Posições",
  "done.altitudes": "Altitudes",
  "done.corrected": "corrigidas",
  "done.estimated": "estimadas",
  "done.rejected": "Medições rejeitadas como salto",
  "done.rejectedValue": "{pos} posições · {alt} altitudes",
  "done.maxShift": "Maior correção de posição",
  "done.sigma": "Incerteza mediana",
  "done.altRange": "Faixa de altitude",
  "done.altRangeValue": "{min}–{max} m (original {omin}–{omax} m)",
  "done.written": "O que foi gravado no arquivo",
  "done.blocks": "Blocos de GPS regravados",
  "done.fixRaised": "Blocos com fix ajustado para 3D",
  "done.fixMissing": "Blocos sem campo de fix (mantidos como estavam)",
  "done.dopChanged": "Blocos com DOP ajustado",
  "done.timesRewritten": "Horários GPS regravados",
  "done.bytes": "Bytes regravados",
  "done.created": "Nova data de criação",
  "done.how": "Como foi restaurado",
  "done.how1.t": "Leitura",
  "done.how1.d": "Os blocos de GPS (GPS5, GPSF, GPSP, GPSU) foram lidos da trilha de telemetria do vídeo, e o horário de cada bloco foi conferido com o relógio do vídeo.",
  "done.how2.t": "Filtragem",
  "done.how2.d": "Um filtro de Kalman de velocidade constante com suavizador RTS combinou posição e velocidade Doppler, pesando cada medição pelo seu DOP.",
  "done.how3.t": "Rejeição de saltos",
  "done.how3.d": "Medições que se deslocaram mais do que a velocidade explica foram descartadas ({n} posições).",
  "done.how4.t": "Lacunas",
  "done.how4.d": "Trechos sem medição utilizável foram interpolados; as pontas foram extrapoladas por no máximo 5 s ({n} posições estimadas).",
  "done.how5.t": "Altitude",
  "done.how5.d": "A altitude usou só amostras com fix 3D, com suavização forte ({n} altitudes estimadas).",
  "done.how6.t": "Gravação no lugar",
  "done.how6.d": "O vídeo foi copiado e só os campos de GPS foram sobrescritos, com o mesmo tamanho. DOP declarado: {dop} nos blocos corrigidos, {dopEst} nos majoritariamente estimados.",
  "done.how7.t": "Verificação",
  "done.how7.d": "O arquivo novo foi comparado byte a byte com o original: fora dos campos de GPS e das datas de criação, é idêntico. Os metadados da GoPro estão intactos.",
  "done.how8.t": "Nome e data novos",
  "done.how8.d": "O arquivo recebeu nome e data de criação novos para a nuvem da GoPro não o tratar como duplicata do original.",
  "done.honesty": "Os valores são reconstruídos, não medidos. A nota \"depois\" é a que o arquivo declara, não uma nova medição. Guarde o vídeo original e o relatório JSON, que marca cada ponto como corrigido ou estimado.",
  "done.quality": "Qualidade ao longo do vídeo",

  "unit.s": "{v} s",
  "unit.m": "{v} m",

  "err.title": "Algo deu errado",
  "err.dismiss": "Fechar",
  "err.aborted": "Restauração abortada. Nenhum arquivo foi gravado.",
  "err.noGps5": "Este vídeo não tem trilha GPS5. A câmera pode ter gravado com o GPS desligado, ou grava em GPS9 (HERO11 ou mais nova), que ainda não é suportado.",
  "err.noTelemetry": "Este vídeo não tem trilha de telemetria da GoPro (gpmd).",
  "err.notMp4": "O arquivo não é um vídeo MP4.",
  "err.invalidMp4": "O arquivo não é um MP4 válido.",
  "err.exists": "Já existe um arquivo de saída com este nome.",
  "err.noGpsTime": "Nenhum bloco de GPS tem horário (GPSU) válido.",
  "err.busy": "Já há uma restauração em andamento.",
  "err.generic": "A operação falhou.",
  "err.details": "Detalhes",
};

const es: Dict = {
  "app.tagline": "Recupera el GPS degradado para que GoPro Quik pueda añadir los indicadores de telemetría",
  "app.language": "Idioma",
  "app.version": "Versión",

  "input.title": "Elige un vídeo de GoPro",
  "input.hint": "Arrastra un MP4 aquí o búscalo en tus archivos. El original nunca se modifica.",
  "input.browse": "Buscar…",
  "input.change": "Cambiar vídeo",
  "input.dialogTitle": "Elige un vídeo de GoPro",
  "input.selected": "Vídeo elegido",
  "input.process": "Restaurar datos GPS",
  "input.output": "Los vídeos restaurados se guardan en {dir}",
  "input.supported": "Cámaras con telemetría GPS5 (HERO5 a HERO10)",

  "diag.running": "Evaluando los datos GPS…",
  "diag.title": "Evaluación del GPS",
  "diag.subtitle": "Este es el estado real del GPS grabado en el vídeo. Todavía no se ha cambiado nada.",
  "diag.score": "Puntuación",
  "diag.verdict.bom": "Bueno",
  "diag.verdict.degradado": "Degradado",
  "diag.verdict.ruim": "Malo",
  "diag.advice.bom": "El GPS es fiable. Restaurar es opcional, aunque puede suavizar la ruta.",
  "diag.advice.degradado": "El GPS se puede usar, pero tiene fallos. Se recomienda restaurarlo antes de usarlo en Quik.",
  "diag.advice.ruim": "El GPS es malo. Restáuralo antes de añadir los indicadores en Quik.",
  "diag.timeline": "Calidad a lo largo del vídeo",
  "diag.timelineHint": "Cada barra es aproximadamente un segundo de GPS",
  "diag.recorded": "Grabado",
  "diag.duration": "Duración del GPS",
  "diag.samples": "Muestras",
  "diag.fix": "Fix (3D / 2D / sin fix)",
  "diag.dop": "DOP mediano",
  "diag.highDop": "Muestras con DOP > 5",
  "diag.jumps": "Saltos de posición",
  "diag.altJumps": "Saltos de altitud",
  "diag.maxAltJump": "Mayor salto de altitud",
  "diag.timeSuspect": "Bloques con hora incoherente",
  "diag.distance": "Distancia con fix 3D",
  "diag.problems": "Problemas encontrados (tiempo del vídeo)",
  "diag.noProblems": "No se encontraron problemas.",
  "diag.confirmQuestion": "¿Restaurar el GPS de este vídeo?",
  "diag.confirmNote": "Se creará un vídeo nuevo en {dir}. Solo cambian los campos GPS; el original queda intacto.",
  "diag.confirm": "Confirmar restauración",
  "diag.abort": "Abortar",

  "issue.sem_fix": "Sin fix",
  "issue.fix_2d": "Solo fix 2D (altitud poco fiable)",
  "issue.dop_alto": "Precisión baja (DOP > 5)",
  "issue.saltos_posicao": "Saltos de posición",
  "issue.saltos_altitude": "Saltos de altitud",
  "issue.horario_inconsistente": "Hora GPS (GPSU) incoherente",

  "run.title": "Restaurando los datos GPS",
  "run.subtitle": "Puedes abortar en cualquier momento. Si lo haces, no se guarda nada.",
  "run.abort": "Abortar restauración",
  "run.aborting": "Abortando…",
  "stage.leitura": "Leyendo la telemetría",
  "stage.reconstrucao": "Reconstruyendo la ruta",
  "stage.copia": "Copiando el vídeo",
  "stage.verificacao": "Verificando byte a byte",
  "stage.releitura": "Releyendo el archivo nuevo",
  "stage.relatorio": "Guardando el informe",

  "done.title": "GPS restaurado",
  "done.subtitle": "El vídeo nuevo está listo para importarlo en GoPro Quik.",
  "done.file": "Vídeo restaurado",
  "done.report": "Informe (JSON)",
  "done.showFolder": "Mostrar en la carpeta",
  "done.another": "Restaurar otro vídeo",
  "done.scoreBefore": "Antes",
  "done.scoreAfter": "Después (declarada)",
  "done.what": "Qué se restauró",
  "done.positions": "Posiciones",
  "done.altitudes": "Altitudes",
  "done.corrected": "corregidas",
  "done.estimated": "estimadas",
  "done.rejected": "Mediciones descartadas como saltos",
  "done.rejectedValue": "{pos} posiciones · {alt} altitudes",
  "done.maxShift": "Mayor corrección de posición",
  "done.sigma": "Incertidumbre mediana",
  "done.altRange": "Rango de altitud",
  "done.altRangeValue": "{min}–{max} m (original {omin}–{omax} m)",
  "done.written": "Qué se escribió en el archivo",
  "done.blocks": "Bloques GPS reescritos",
  "done.fixRaised": "Bloques con fix ajustado a 3D",
  "done.fixMissing": "Bloques sin campo de fix (sin cambios)",
  "done.dopChanged": "Bloques con DOP ajustado",
  "done.timesRewritten": "Horas GPS reescritas",
  "done.bytes": "Bytes reescritos",
  "done.created": "Nueva fecha de creación",
  "done.how": "Cómo se restauró",
  "done.how1.t": "Lectura",
  "done.how1.d": "Se leyeron los bloques GPS (GPS5, GPSF, GPSP, GPSU) de la pista de telemetría del vídeo y se comprobó la hora de cada bloque con el reloj del vídeo.",
  "done.how2.t": "Filtrado",
  "done.how2.d": "Un filtro de Kalman de velocidad constante con suavizador RTS combinó la posición y la velocidad Doppler, ponderando cada medición según su DOP.",
  "done.how3.t": "Descarte de saltos",
  "done.how3.d": "Se descartaron las mediciones que se desplazaron más de lo que explica la velocidad ({n} posiciones).",
  "done.how4.t": "Huecos",
  "done.how4.d": "Los tramos sin medición útil se interpolaron; los extremos se extrapolaron como máximo 5 s ({n} posiciones estimadas).",
  "done.how5.t": "Altitud",
  "done.how5.d": "La altitud usó solo muestras con fix 3D, con un suavizado fuerte ({n} altitudes estimadas).",
  "done.how6.t": "Escritura en el sitio",
  "done.how6.d": "Se copió el vídeo y solo se sobrescribieron los campos GPS, con el mismo tamaño. DOP declarado: {dop} en los bloques corregidos, {dopEst} en los mayoritariamente estimados.",
  "done.how7.t": "Verificación",
  "done.how7.d": "El archivo nuevo se comparó byte a byte con el original: fuera de los campos GPS y de las fechas de creación, es idéntico. Los metadatos de GoPro están intactos.",
  "done.how8.t": "Nombre y fecha nuevos",
  "done.how8.d": "El archivo recibió un nombre y una fecha de creación nuevos para que la nube de GoPro no lo trate como un duplicado del original.",
  "done.honesty": "Los valores son reconstruidos, no medidos. La puntuación \"después\" es la que declara el archivo, no una nueva medición. Conserva el vídeo original y el informe JSON, que marca cada punto como corregido o estimado.",
  "done.quality": "Calidad a lo largo del vídeo",

  "unit.s": "{v} s",
  "unit.m": "{v} m",

  "err.title": "Algo salió mal",
  "err.dismiss": "Cerrar",
  "err.aborted": "Restauración abortada. No se guardó ningún archivo.",
  "err.noGps5": "Este vídeo no tiene pista GPS5. Puede que la cámara tuviera el GPS apagado o que grabe GPS9 (HERO11 o posterior), que aún no es compatible.",
  "err.noTelemetry": "Este vídeo no tiene pista de telemetría de GoPro (gpmd).",
  "err.notMp4": "El archivo no es un vídeo MP4.",
  "err.invalidMp4": "El archivo no es un MP4 válido.",
  "err.exists": "Ya existe un archivo de salida con este nombre.",
  "err.noGpsTime": "Ningún bloque GPS tiene una hora (GPSU) válida.",
  "err.busy": "Ya hay una restauración en curso.",
  "err.generic": "La operación falló.",
  "err.details": "Detalles",
};

const zh: Dict = {
  "app.tagline": "修复受损的 GPS 数据，让 GoPro Quik 能够添加遥测仪表",
  "app.language": "语言",
  "app.version": "版本",

  "input.title": "选择一个 GoPro 视频",
  "input.hint": "将 MP4 拖到此处，或浏览文件。原始文件不会被修改。",
  "input.browse": "浏览…",
  "input.change": "更换视频",
  "input.dialogTitle": "选择一个 GoPro 视频",
  "input.selected": "已选视频",
  "input.process": "修复 GPS 数据",
  "input.output": "修复后的视频保存在 {dir}",
  "input.supported": "支持 GPS5 遥测的相机（HERO5 至 HERO10）",

  "diag.running": "正在评估 GPS 数据…",
  "diag.title": "GPS 评估",
  "diag.subtitle": "这是视频中记录的 GPS 的真实状况。目前尚未做任何修改。",
  "diag.score": "评分",
  "diag.verdict.bom": "良好",
  "diag.verdict.degradado": "受损",
  "diag.verdict.ruim": "较差",
  "diag.advice.bom": "GPS 数据可靠。修复是可选的，但仍可使轨迹更平滑。",
  "diag.advice.degradado": "GPS 数据可用但有缺陷。建议在 Quik 中使用前先修复。",
  "diag.advice.ruim": "GPS 数据较差。请先修复，再在 Quik 中添加仪表。",
  "diag.timeline": "视频各时段的质量",
  "diag.timelineHint": "每一条约代表一秒的 GPS 数据",
  "diag.recorded": "录制时间",
  "diag.duration": "GPS 时长",
  "diag.samples": "采样数",
  "diag.fix": "定位（3D / 2D / 无）",
  "diag.dop": "DOP 中位数",
  "diag.highDop": "DOP > 5 的采样",
  "diag.jumps": "位置跳变",
  "diag.altJumps": "海拔跳变",
  "diag.maxAltJump": "最大海拔跳变",
  "diag.timeSuspect": "时间不一致的数据块",
  "diag.distance": "3D 定位下的距离",
  "diag.problems": "发现的问题（视频时间）",
  "diag.noProblems": "未发现问题。",
  "diag.confirmQuestion": "要修复此视频的 GPS 吗？",
  "diag.confirmNote": "将在 {dir} 中创建一个新视频。只有 GPS 字段会改变，原始文件保持不变。",
  "diag.confirm": "确认修复",
  "diag.abort": "中止",

  "issue.sem_fix": "无定位",
  "issue.fix_2d": "仅 2D 定位（海拔不可靠）",
  "issue.dop_alto": "精度较差（DOP > 5）",
  "issue.saltos_posicao": "位置跳变",
  "issue.saltos_altitude": "海拔跳变",
  "issue.horario_inconsistente": "GPS 时间（GPSU）不一致",

  "run.title": "正在修复 GPS 数据",
  "run.subtitle": "你可以随时中止。中止后不会保留任何文件。",
  "run.abort": "中止修复",
  "run.aborting": "正在中止…",
  "stage.leitura": "读取遥测数据",
  "stage.reconstrucao": "重建轨迹",
  "stage.copia": "复制视频",
  "stage.verificacao": "逐字节校验",
  "stage.releitura": "重新读取新文件",
  "stage.relatorio": "保存报告",

  "done.title": "GPS 已修复",
  "done.subtitle": "新视频已可导入 GoPro Quik。",
  "done.file": "修复后的视频",
  "done.report": "报告（JSON）",
  "done.showFolder": "在文件夹中显示",
  "done.another": "修复另一个视频",
  "done.scoreBefore": "修复前",
  "done.scoreAfter": "修复后（声明值）",
  "done.what": "修复内容",
  "done.positions": "位置",
  "done.altitudes": "海拔",
  "done.corrected": "已校正",
  "done.estimated": "已估算",
  "done.rejected": "被判为跳变而舍弃的测量",
  "done.rejectedValue": "{pos} 个位置 · {alt} 个海拔",
  "done.maxShift": "最大位置校正",
  "done.sigma": "不确定度中位数",
  "done.altRange": "海拔范围",
  "done.altRangeValue": "{min}–{max} 米（原始 {omin}–{omax} 米）",
  "done.written": "写入文件的内容",
  "done.blocks": "重写的 GPS 数据块",
  "done.fixRaised": "定位提升为 3D 的数据块",
  "done.fixMissing": "没有定位字段的数据块（保持原样）",
  "done.dopChanged": "调整了 DOP 的数据块",
  "done.timesRewritten": "重写的 GPS 时间",
  "done.bytes": "重写的字节数",
  "done.created": "新的创建日期",
  "done.how": "修复方法",
  "done.how1.t": "读取",
  "done.how1.d": "从视频的遥测轨道读取 GPS 数据块（GPS5、GPSF、GPSP、GPSU），并将每个数据块的时间与视频时钟进行核对。",
  "done.how2.t": "滤波",
  "done.how2.d": "使用带 RTS 平滑器的匀速卡尔曼滤波器，结合位置与多普勒速度，并按 DOP 为每次测量加权。",
  "done.how3.t": "剔除跳变",
  "done.how3.d": "位移超出速度可解释范围的测量被舍弃（{n} 个位置）。",
  "done.how4.t": "填补空缺",
  "done.how4.d": "没有可用测量的片段通过插值补全；首尾最多外推 5 秒（{n} 个估算位置）。",
  "done.how5.t": "海拔",
  "done.how5.d": "海拔仅使用 3D 定位的采样，并进行强平滑（{n} 个估算海拔）。",
  "done.how6.t": "原位写入",
  "done.how6.d": "复制视频后，仅以相同大小覆盖 GPS 字段。声明的 DOP：校正数据块为 {dop}，以估算为主的数据块为 {dopEst}。",
  "done.how7.t": "校验",
  "done.how7.d": "新文件已与原文件逐字节比较：除 GPS 字段和创建日期外完全相同。GoPro 元数据保持完整。",
  "done.how8.t": "新的名称和日期",
  "done.how8.d": "文件获得了新的名称和创建日期，以免 GoPro 云端将其视为原文件的重复项。",
  "done.honesty": "这些数值是重建的，而非实测。“修复后”的评分是文件所声明的，并非重新测量。请保留原始视频和 JSON 报告，报告会将每个点标记为已校正或已估算。",
  "done.quality": "视频各时段的质量",

  "unit.s": "{v} 秒",
  "unit.m": "{v} 米",

  "err.title": "出现问题",
  "err.dismiss": "关闭",
  "err.aborted": "修复已中止。未写入任何文件。",
  "err.noGps5": "此视频没有 GPS5 轨道。相机可能关闭了 GPS，或者记录的是 GPS9（HERO11 或更新机型），目前尚不支持。",
  "err.noTelemetry": "此视频没有 GoPro 遥测轨道（gpmd）。",
  "err.notMp4": "该文件不是 MP4 视频。",
  "err.invalidMp4": "该文件不是有效的 MP4。",
  "err.exists": "已存在同名的输出文件。",
  "err.noGpsTime": "没有任何 GPS 数据块具有有效时间（GPSU）。",
  "err.busy": "已有一个修复任务正在进行。",
  "err.generic": "操作失败。",
  "err.details": "详细信息",
};

const dicts: Record<Lang, Dict> = { en, pt, es, zh };

const STORAGE_KEY = "gopro-gps-repair.lang";

function initialLang(): Lang {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved && saved in dicts) return saved as Lang;
  } catch {
    // sem armazenamento: usa o padrão
  }
  return "en";
}

type Ctx = {
  lang: Lang;
  setLang: (l: Lang) => void;
  t: (key: Key, vars?: Record<string, string | number>) => string;
  num: (v: number, digits?: number) => string;
  locale: string;
};

const I18nContext = createContext<Ctx | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(initialLang);
  const locale = languages.find((l) => l.code === lang)!.locale;

  const setLang = useCallback((l: Lang) => {
    setLangState(l);
    try {
      localStorage.setItem(STORAGE_KEY, l);
    } catch {
      // ignora
    }
  }, []);

  const value = useMemo<Ctx>(() => {
    const fmts = new Map<number, Intl.NumberFormat>();
    const num = (v: number, digits = 0) => {
      if (!Number.isFinite(v)) return "—";
      let f = fmts.get(digits);
      if (!f) {
        f = new Intl.NumberFormat(locale, { minimumFractionDigits: digits, maximumFractionDigits: digits });
        fmts.set(digits, f);
      }
      return f.format(v);
    };
    const t = (key: Key, vars?: Record<string, string | number>) => {
      let s = dicts[lang][key] ?? en[key] ?? key;
      if (vars) {
        for (const [k, v] of Object.entries(vars)) s = s.split(`{${k}}`).join(String(v));
      }
      return s;
    };
    return { lang, setLang, t, num, locale };
  }, [lang, locale, setLang]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const c = useContext(I18nContext);
  if (!c) throw new Error("useI18n fora do I18nProvider");
  return c;
}

// errorKey traduz as mensagens de erro do Go (em português) para chaves.
export function errorKey(message: string): Key {
  const m = message.toLowerCase();
  if (m.includes("cancelado")) return "err.aborted";
  if (m.includes("sem blocos gps5")) return "err.noGps5";
  if (m.includes("sem trilha de telemetria")) return "err.noTelemetry";
  if (m.includes("não é um vídeo mp4")) return "err.notMp4";
  if (m.includes("sem box moov") || m.includes("tamanho inválido")) return "err.invalidMp4";
  if (m.includes("já existe")) return "err.exists";
  if (m.includes("nenhum bloco com horário")) return "err.noGpsTime";
  if (m.includes("em andamento")) return "err.busy";
  return "err.generic";
}
