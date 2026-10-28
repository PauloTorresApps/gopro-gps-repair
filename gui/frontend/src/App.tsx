import { useCallback, useEffect, useRef, useState } from "react";
import { Events } from "@wailsio/runtime";
import { RepairService } from "../bindings/gopro-gps-repair/gui";
import type { Report } from "../bindings/gopro-gps-repair/internal/diagnose";
import type { Result } from "../bindings/gopro-gps-repair/internal/repair";
import { errorKey, languages, useI18n, type Lang } from "./i18n";
import { Button, Card, Icon, Spinner } from "./ui";
import { DiagnosisView } from "./DiagnosisView";
import { ProgressView, type ProgressState } from "./ProgressView";
import { ReportView } from "./ReportView";

// Fases da tela: escolher o vídeo → avaliar → confirmar → restaurar → relatório.
type Phase = "input" | "diagnosing" | "confirm" | "repairing" | "done";

type ErrorState = { key: ReturnType<typeof errorKey>; detail: string } | null;

export default function App() {
  const { t } = useI18n();
  const [phase, setPhase] = useState<Phase>("input");
  const [video, setVideo] = useState("");
  const [outputDir, setOutputDir] = useState("");
  const [report, setReport] = useState<Report | null>(null);
  const [result, setResult] = useState<Result | null>(null);
  const [progress, setProgress] = useState<ProgressState>({ stage: "", done: 0, total: 0, seen: [] });
  const [aborting, setAborting] = useState(false);
  const [error, setError] = useState<ErrorState>(null);
  const phaseRef = useRef(phase);
  phaseRef.current = phase;

  useEffect(() => {
    RepairService.OutputDir().then(setOutputDir).catch(() => {});
    const offDrop = Events.On("arquivo:solto", (ev) => {
      if (phaseRef.current === "input" || phaseRef.current === "confirm") {
        chooseVideo(ev.data);
      }
    });
    const offProgress = Events.On("reparo:progresso", (ev) => {
      const p = ev.data;
      setProgress((prev) => ({
        stage: p.etapa,
        done: p.feito,
        total: p.total,
        seen: prev.seen.includes(p.etapa) ? prev.seen : [...prev.seen, p.etapa],
      }));
    });
    return () => {
      offDrop();
      offProgress();
    };
  }, []);

  const fail = (e: unknown) => {
    const detail = e instanceof Error ? e.message : String(e);
    setError({ key: errorKey(detail), detail });
  };

  const chooseVideo = (path: string) => {
    if (!path) return;
    setVideo(path);
    setReport(null);
    setResult(null);
    setError(null);
    setPhase("input");
  };

  const browse = async () => {
    try {
      chooseVideo(await RepairService.SelectVideo(t("input.dialogTitle")));
    } catch (e) {
      fail(e);
    }
  };

  // 1º passo: avalia o GPS e mostra a situação real antes de restaurar.
  const evaluate = async () => {
    setError(null);
    setPhase("diagnosing");
    try {
      const r = await RepairService.Diagnose(video);
      setReport(r);
      setPhase("confirm");
    } catch (e) {
      fail(e);
      setPhase("input");
    }
  };

  // 2º passo, depois da confirmação: restaura.
  const repair = async () => {
    setError(null);
    setAborting(false);
    setProgress({ stage: "", done: 0, total: 0, seen: [] });
    setPhase("repairing");
    try {
      const r = await RepairService.Repair(video);
      setResult(r);
      setPhase("done");
    } catch (e) {
      fail(e);
      setPhase("confirm");
    } finally {
      setAborting(false);
    }
  };

  const abortRepair = () => {
    setAborting(true);
    RepairService.Abort().catch(fail);
  };

  const reset = useCallback(() => {
    setVideo("");
    setReport(null);
    setResult(null);
    setError(null);
    setPhase("input");
  }, []);

  return (
    <div className="flex h-full flex-col">
      <Header />
      <main className="mx-auto w-full max-w-5xl flex-1 overflow-y-auto px-6 pb-10 pt-6">
        {error && <ErrorBanner error={error} onClose={() => setError(null)} />}

        {(phase === "input" || phase === "diagnosing") && (
          <InputView
            video={video}
            outputDir={outputDir}
            busy={phase === "diagnosing"}
            onBrowse={browse}
            onProcess={evaluate}
          />
        )}

        {phase === "confirm" && report && (
          <DiagnosisView video={video} report={report} outputDir={outputDir} onConfirm={repair} onAbort={reset} />
        )}

        {phase === "repairing" && (
          <ProgressView video={video} progress={progress} aborting={aborting} onAbort={abortRepair} />
        )}

        {phase === "done" && result && <ReportView result={result} onAnother={reset} />}
      </main>
    </div>
  );
}

function Header() {
  const { t, lang, setLang } = useI18n();
  return (
    <header className="border-b border-ink-800 bg-ink-950/90">
      <div className="mx-auto flex max-w-5xl items-center gap-4 px-6 py-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-ink-800 text-sky-brand">{Icon.pin}</div>
        <div className="min-w-0 flex-1">
          <h1 className="text-lg font-bold tracking-tight">
            <span className="text-sky-brand">GPS</span> Repair <span className="font-medium text-ink-300">for GoPro</span>
          </h1>
          <p className="truncate text-xs text-ink-400">{t("app.tagline")}</p>
        </div>
        <label className="flex items-center gap-2 text-xs text-ink-400">
          <span className="sr-only sm:not-sr-only">{t("app.language")}</span>
          <select
            value={lang}
            onChange={(e) => setLang(e.target.value as Lang)}
            className="rounded-lg border border-ink-700 bg-ink-850 px-2.5 py-1.5 text-sm text-ink-100 focus:border-sky-brand focus:outline-none"
          >
            {languages.map((l) => (
              <option key={l.code} value={l.code} className="bg-white text-ink-950">
                {l.label}
              </option>
            ))}
          </select>
        </label>
      </div>
    </header>
  );
}

function ErrorBanner({ error, onClose }: { error: NonNullable<ErrorState>; onClose: () => void }) {
  const { t } = useI18n();
  const aborted = error.key === "err.aborted";
  return (
    <div
      role="alert"
      className={`mb-6 flex items-start gap-3 rounded-xl border px-4 py-3 ${
        aborted ? "border-ink-600 bg-ink-850 text-ink-100" : "border-bad/50 bg-bad/10 text-ink-100"
      }`}
    >
      <span className={aborted ? "text-ink-300" : "text-bad"}>{Icon.alert}</span>
      <div className="min-w-0 flex-1 text-sm">
        {!aborted && <div className="font-semibold">{t("err.title")}</div>}
        <div>{t(error.key)}</div>
        {error.key === "err.generic" && (
          <div className="selectable mt-1 break-words text-xs text-ink-300">
            {t("err.details")}: {error.detail}
          </div>
        )}
      </div>
      <button onClick={onClose} className="text-xs text-ink-300 hover:text-ink-100">
        {t("err.dismiss")}
      </button>
    </div>
  );
}

function basename(p: string) {
  return p.split(/[\\/]/).pop() || p;
}

function InputView({
  video,
  outputDir,
  busy,
  onBrowse,
  onProcess,
}: {
  video: string;
  outputDir: string;
  busy: boolean;
  onBrowse: () => void;
  onProcess: () => void;
}) {
  const { t } = useI18n();
  const [over, setOver] = useState(false);
  return (
    <div className="space-y-6">
      <Card className="p-0">
        <div
          data-file-drop-target
          onDragEnter={() => setOver(true)}
          onDragLeave={() => setOver(false)}
          onDrop={() => setOver(false)}
          className={`m-2 flex flex-col items-center justify-center rounded-xl border-2 border-dashed px-6 py-14 text-center transition ${
            over ? "border-sky-brand bg-sky-brand/5" : "border-ink-700"
          }`}
        >
          <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-ink-800 text-sky-brand">{Icon.upload}</div>
          <h2 className="text-xl font-semibold">{t("input.title")}</h2>
          <p className="mt-2 max-w-md text-sm text-ink-300">{t("input.hint")}</p>
          <Button variant={video ? "ghost" : "primary"} className="mt-6" onClick={onBrowse} disabled={busy}>
            {video ? t("input.change") : t("input.browse")}
          </Button>
          <p className="mt-4 text-xs text-ink-400">{t("input.supported")}</p>
        </div>
      </Card>

      {video && (
        <Card>
          <div className="flex flex-wrap items-center gap-4">
            <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-ink-800 text-sky-brand">{Icon.video}</div>
            <div className="min-w-0 flex-1">
              <div className="text-xs text-ink-400">{t("input.selected")}</div>
              <div className="selectable truncate font-semibold" title={video}>
                {basename(video)}
              </div>
              <div className="selectable truncate text-xs text-ink-400" title={video}>
                {video}
              </div>
            </div>
            <Button onClick={onProcess} disabled={busy} className="px-6 py-3">
              {busy ? (
                <>
                  <Spinner className="h-4 w-4 text-ink-950" /> {t("diag.running")}
                </>
              ) : (
                t("input.process")
              )}
            </Button>
          </div>
          {outputDir && (
            <p className="mt-4 flex items-center gap-2 border-t border-ink-800 pt-4 text-xs text-ink-400">
              {Icon.folder}
              <span className="selectable">{t("input.output", { dir: outputDir })}</span>
            </p>
          )}
        </Card>
      )}
    </div>
  );
}
