import { useI18n, type Key } from "./i18n";
import { Button, Card, Icon, Spinner } from "./ui";

export type ProgressState = { stage: string; done: number; total: number; seen: string[] };

// Etapas na ordem em que o Go as informa (pacote internal/repair).
const stages = ["leitura", "reconstrucao", "copia", "verificacao", "releitura", "relatorio"] as const;

// ProgressView acompanha a restauração e oferece o botão de abortar.
export function ProgressView({
  video,
  progress,
  aborting,
  onAbort,
}: {
  video: string;
  progress: ProgressState;
  aborting: boolean;
  onAbort: () => void;
}) {
  const { t, num } = useI18n();
  const current = stages.indexOf(progress.stage as (typeof stages)[number]);
  const pct = progress.total > 0 ? Math.min(100, (100 * progress.done) / progress.total) : null;
  // Cópia e verificação dominam o tempo; o resto é rápido.
  const weights = [2, 3, 40, 50, 3, 2];
  const before = weights.slice(0, Math.max(0, current)).reduce((a, b) => a + b, 0);
  const overall = current < 0 ? 0 : before + (weights[current] * (pct ?? 50)) / 100;

  return (
    <Card className="mx-auto max-w-2xl">
      <div className="flex items-center gap-4">
        <Spinner className="h-7 w-7" />
        <div className="min-w-0">
          <h2 className="text-xl font-semibold">{t("run.title")}</h2>
          <p className="selectable truncate text-sm text-ink-400" title={video}>
            {video.split(/[\\/]/).pop()}
          </p>
        </div>
      </div>

      <div className="mt-6">
        <div className="h-2.5 overflow-hidden rounded-full bg-ink-800">
          <div
            className="h-full rounded-full bg-sky-brand transition-[width] duration-300"
            style={{ width: `${Math.max(2, overall)}%` }}
          />
        </div>
        <div className="mt-1.5 text-right text-xs tabular-nums text-ink-400">{num(overall, 0)}%</div>
      </div>

      <ol className="mt-4 space-y-2">
        {stages.map((s, i) => {
          const done = current > i || (current === -1 && progress.seen.includes(s));
          const active = current === i;
          return (
            <li key={s} className="flex items-center gap-3 text-sm">
              <span
                className={`flex h-6 w-6 items-center justify-center rounded-full ${
                  done ? "bg-sky-brand text-ink-950" : active ? "border-2 border-sky-brand" : "border border-ink-600"
                }`}
              >
                {done && Icon.check}
              </span>
              <span className={done ? "text-ink-300" : active ? "font-semibold text-ink-100" : "text-ink-400"}>
                {t(`stage.${s}` as Key)}
              </span>
              {active && pct !== null && (
                <span className="ml-auto tabular-nums text-xs text-sky-brand">{num(pct, 0)}%</span>
              )}
            </li>
          );
        })}
      </ol>

      <div className="mt-8 flex flex-col items-center gap-3 border-t border-ink-800 pt-6">
        <p className="text-center text-xs text-ink-400">{t("run.subtitle")}</p>
        <Button variant="danger" onClick={onAbort} disabled={aborting}>
          {aborting ? t("run.aborting") : t("run.abort")}
        </Button>
      </div>
    </Card>
  );
}
