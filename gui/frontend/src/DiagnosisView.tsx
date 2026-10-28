import type { Report } from "../bindings/gopro-gps-repair/internal/diagnose";
import { useI18n, type Key } from "./i18n";
import { Button, Card, QualityStrip, ScoreRing, SectionTitle, Stat, verdictTone } from "./ui";

// DiagnosisView mostra a avaliação do GPS e pede confirmação para restaurar.
export function DiagnosisView({
  video,
  report: r,
  outputDir,
  onConfirm,
  onAbort,
}: {
  video: string;
  report: Report;
  outputDir: string;
  onConfirm: () => void;
  onAbort: () => void;
}) {
  const { t, num, locale } = useI18n();
  const tone = verdictTone(r.veredito);
  const issues = r.trechos_problema ?? [];
  const recorded = new Date(r.inicio);

  return (
    <div className="space-y-6">
      <Card>
        <div className="flex flex-col gap-6 sm:flex-row sm:items-center">
          <ScoreRing score={r.nota} verdict={r.veredito} />
          <div className="min-w-0 flex-1">
            <h2 className="text-xl font-semibold">{t("diag.title")}</h2>
            <p className="selectable mt-1 truncate text-sm text-ink-400" title={video}>
              {video.split(/[\\/]/).pop()}
            </p>
            <p className="mt-3 text-sm text-ink-300">{t("diag.subtitle")}</p>
            <p
              className={`mt-3 rounded-lg px-3 py-2 text-sm ${
                tone === "ok" ? "bg-sky-brand/10 text-sky-brand-hi" : tone === "warn" ? "bg-warn/10 text-warn" : "bg-bad/10 text-bad"
              }`}
            >
              {t(`diag.advice.${r.veredito}` as Key)}
            </p>
          </div>
        </div>
      </Card>

      <Card>
        <SectionTitle hint={t("diag.timelineHint")}>{t("diag.timeline")}</SectionTitle>
        <QualityStrip blocks={r.lista_blocos ?? []} duration={r.duracao_s} />
      </Card>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat
          label={t("diag.recorded")}
          value={
            Number.isNaN(recorded.getTime())
              ? "—"
              : recorded.toLocaleString(locale, { dateStyle: "short", timeStyle: "short" })
          }
        />
        <Stat label={t("diag.duration")} value={t("unit.s", { v: num(r.duracao_s, 1) })} />
        <Stat label={t("diag.samples")} value={num(r.amostras)} />
        <Stat
          label={t("diag.fix")}
          value={`${num(r.amostras_fix_3d)} / ${num(r.amostras_fix_2d)} / ${num(r.amostras_sem_fix)}`}
          tone={r.amostras_sem_fix + r.amostras_fix_2d > 0 ? "warn" : undefined}
        />
        <Stat label={t("diag.dop")} value={num(r.dop_mediano, 2)} tone={r.dop_mediano > 5 ? "bad" : undefined} />
        <Stat label={t("diag.highDop")} value={num(r.amostras_dop_alto)} tone={r.amostras_dop_alto > 0 ? "warn" : undefined} />
        <Stat label={t("diag.jumps")} value={num(r.saltos_posicao)} tone={r.saltos_posicao > 0 ? "warn" : undefined} />
        <Stat
          label={t("diag.altJumps")}
          value={
            r.saltos_altitude > 0
              ? `${num(r.saltos_altitude)} · ${t("unit.m", { v: num(r.maior_salto_altitude_m, 1) })}`
              : num(0)
          }
          tone={r.saltos_altitude > 0 ? "warn" : undefined}
        />
        {r.blocos_horario_suspeito > 0 && (
          <Stat label={t("diag.timeSuspect")} value={num(r.blocos_horario_suspeito)} tone="warn" />
        )}
        <Stat label={t("diag.distance")} value={t("unit.m", { v: num(r.distancia_m) })} />
      </div>

      <Card>
        <SectionTitle>{t("diag.problems")}</SectionTitle>
        {issues.length === 0 ? (
          <p className="text-sm text-ink-300">{t("diag.noProblems")}</p>
        ) : (
          <ul className="divide-y divide-ink-800">
            {issues.map((is) => (
              <li key={is.tipo} className="flex flex-col gap-1 py-2.5 sm:flex-row sm:items-baseline sm:gap-6">
                <span className="w-64 shrink-0 text-sm font-medium">{t(`issue.${is.tipo}` as Key)}</span>
                <span className="text-sm tabular-nums text-ink-300">
                  {(is.trechos ?? [])
                    .map((s) => t("unit.s", { v: `${num(s.inicio_s, 1)}–${num(s.fim_s, 1)}` }))
                    .join(", ")}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Card>

      <Card className="border-sky-brand/30">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
          <div className="flex-1">
            <h3 className="text-base font-semibold">{t("diag.confirmQuestion")}</h3>
            <p className="selectable mt-1 text-sm text-ink-300">{t("diag.confirmNote", { dir: outputDir || "~/gopro-gps-repair" })}</p>
          </div>
          <div className="flex gap-3">
            <Button variant="ghost" onClick={onAbort}>
              {t("diag.abort")}
            </Button>
            <Button onClick={onConfirm} className="px-6">
              {t("diag.confirm")}
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}
