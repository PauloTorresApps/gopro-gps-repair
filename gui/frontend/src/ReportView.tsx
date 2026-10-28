import { RepairService } from "../bindings/gopro-gps-repair/gui";
import type { Result } from "../bindings/gopro-gps-repair/internal/repair";
import { useI18n, type Key } from "./i18n";
import { Button, Card, Icon, QualityStrip, ScoreRing, SectionTitle, Stat } from "./ui";

// ReportView mostra o que foi restaurado e como.
export function ReportView({ result: r, onAnother }: { result: Result; onAnother: () => void }) {
  const { t, num, locale } = useI18n();
  const before = r.antes!;
  const after = r.depois!;
  const rec = r.reconstrucao!;
  const w = r.gravacao!;
  const total = r.amostras;

  const how: { n: number; vars?: Record<string, string | number> }[] = [
    { n: 1 },
    { n: 2 },
    { n: 3, vars: { n: num(rec.posicoes_rejeitadas) } },
    { n: 4, vars: { n: num(rec.posicoes_estimadas) } },
    { n: 5, vars: { n: num(rec.altitudes_estimadas) } },
    { n: 6, vars: { dop: num(r.dop_corrigido, 1), dopEst: num(r.dop_estimado, 1) } },
    { n: 7 },
  ];
  if (!r.manter_data) how.push({ n: 8 });

  const created = w.data_de_criacao ? new Date(w.data_de_criacao) : null;

  return (
    <div className="space-y-6">
      <Card className="border-sky-brand/30">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-sky-brand text-ink-950">
            <span className="scale-150">{Icon.check}</span>
          </div>
          <div className="min-w-0 flex-1">
            <h2 className="text-xl font-semibold">{t("done.title")}</h2>
            <p className="text-sm text-ink-300">{t("done.subtitle")}</p>
          </div>
          <div className="flex gap-3">
            <Button variant="ghost" onClick={() => RepairService.ShowInFolder(r.saida).catch(() => {})}>
              {Icon.folder} {t("done.showFolder")}
            </Button>
            <Button onClick={onAnother}>{t("done.another")}</Button>
          </div>
        </div>
        <dl className="mt-5 grid gap-3 border-t border-ink-800 pt-4 text-sm">
          <div>
            <dt className="text-xs text-ink-400">{t("done.file")}</dt>
            <dd className="selectable break-all font-medium">{r.saida}</dd>
          </div>
          <div>
            <dt className="text-xs text-ink-400">{t("done.report")}</dt>
            <dd className="selectable break-all text-ink-300">{r.relatorio}</dd>
          </div>
        </dl>
      </Card>

      <Card>
        <div className="flex flex-wrap items-center justify-center gap-10">
          <ScoreRing score={before.nota} verdict={before.veredito} label={t("done.scoreBefore")} size={116} />
          <svg viewBox="0 0 24 24" className="h-8 w-8 text-ink-400" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
            <path d="M5 12h14M13 6l6 6-6 6" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          <ScoreRing score={after.nota} verdict={after.veredito} label={t("done.scoreAfter")} size={116} />
        </div>
        <div className="mt-6 grid gap-4 md:grid-cols-2">
          <div>
            <div className="mb-2 text-xs text-ink-400">{t("done.scoreBefore")}</div>
            <QualityStrip blocks={before.lista_blocos ?? []} duration={before.duracao_s} />
          </div>
          <div>
            <div className="mb-2 text-xs text-ink-400">{t("done.scoreAfter")}</div>
            <QualityStrip blocks={after.lista_blocos ?? []} duration={after.duracao_s} />
          </div>
        </div>
      </Card>

      <Card>
        <SectionTitle>{t("done.what")}</SectionTitle>
        <div className="space-y-4">
          <Split
            label={t("done.positions")}
            corrected={total - rec.posicoes_estimadas}
            estimated={rec.posicoes_estimadas}
            total={total}
          />
          <Split
            label={t("done.altitudes")}
            corrected={total - rec.altitudes_estimadas}
            estimated={rec.altitudes_estimadas}
            total={total}
          />
        </div>
        <div className="mt-5 grid grid-cols-2 gap-3 md:grid-cols-4">
          <Stat
            label={t("done.rejected")}
            value={
              <span className="text-sm">
                {t("done.rejectedValue", { pos: num(rec.posicoes_rejeitadas), alt: num(rec.altitudes_rejeitadas) })}
              </span>
            }
          />
          <Stat label={t("done.maxShift")} value={t("unit.m", { v: num(rec.maior_correcao_m, 1) })} />
          <Stat label={t("done.sigma")} value={t("unit.m", { v: num(rec.incerteza_mediana_m, 1) })} />
          {r.faixa_altitude && r.faixa_altitude_original && (
            <Stat
              label={t("done.altRange")}
              value={
                <span className="text-sm">
                  {t("done.altRangeValue", {
                    min: num(r.faixa_altitude.min),
                    max: num(r.faixa_altitude.max),
                    omin: num(r.faixa_altitude_original.min),
                    omax: num(r.faixa_altitude_original.max),
                  })}
                </span>
              }
            />
          )}
        </div>
      </Card>

      <Card>
        <SectionTitle>{t("done.written")}</SectionTitle>
        <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
          <Stat label={t("done.blocks")} value={num(w.blocos_regravados)} />
          <Stat label={t("done.fixRaised")} value={num(w.blocos_com_fix_ajustado)} />
          <Stat label={t("done.dopChanged")} value={num(w.blocos_com_dop_ajustado)} />
          <Stat label={t("done.timesRewritten")} value={num(w.horarios_regravados)} />
          <Stat label={t("done.bytes")} value={num(w.bytes_regravados)} />
          {created && !Number.isNaN(created.getTime()) && (
            <Stat label={t("done.created")} value={created.toLocaleString(locale, { dateStyle: "short", timeStyle: "medium" })} />
          )}
          {w.blocos_sem_campo_fix > 0 && <Stat label={t("done.fixMissing")} value={num(w.blocos_sem_campo_fix)} tone="warn" />}
        </div>
      </Card>

      <Card>
        <SectionTitle>{t("done.how")}</SectionTitle>
        <ol className="space-y-4">
          {how.map(({ n, vars }) => (
            <li key={n} className="flex gap-4">
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-ink-800 text-xs font-bold text-sky-brand">
                {n}
              </span>
              <div>
                <div className="text-sm font-semibold">{t(`done.how${n}.t` as Key)}</div>
                <p className="mt-0.5 text-sm leading-relaxed text-ink-300">{t(`done.how${n}.d` as Key, vars)}</p>
              </div>
            </li>
          ))}
        </ol>
        <p className="mt-6 rounded-lg border border-warn/30 bg-warn/5 px-4 py-3 text-xs leading-relaxed text-ink-300">
          {t("done.honesty")}
        </p>
      </Card>
    </div>
  );
}

function Split({ label, corrected, estimated, total }: { label: string; corrected: number; estimated: number; total: number }) {
  const { t, num } = useI18n();
  const c = total > 0 ? (100 * corrected) / total : 0;
  return (
    <div>
      <div className="mb-1.5 flex flex-wrap items-baseline justify-between gap-2 text-sm">
        <span className="font-medium">{label}</span>
        <span className="tabular-nums text-ink-300">
          <span className="text-sky-brand">{num(corrected)}</span> {t("done.corrected")} ·{" "}
          <span className="text-warn">{num(estimated)}</span> {t("done.estimated")} / {num(total)}
        </span>
      </div>
      <div className="flex h-2.5 overflow-hidden rounded-full bg-ink-800">
        <div className="bg-sky-brand" style={{ width: `${c}%` }} />
        <div className="bg-warn" style={{ width: `${100 - c}%` }} />
      </div>
    </div>
  );
}
