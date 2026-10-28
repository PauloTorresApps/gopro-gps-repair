import type { ButtonHTMLAttributes, ReactNode } from "react";
import type { BlockReport } from "../bindings/gopro-gps-repair/internal/diagnose";
import { useI18n, type Key } from "./i18n";

type Variant = "primary" | "ghost" | "danger";

export function Button({
  variant = "primary",
  className = "",
  children,
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  const base =
    "inline-flex items-center justify-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold transition " +
    "focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-brand/70 disabled:cursor-not-allowed disabled:opacity-50";
  const styles: Record<Variant, string> = {
    primary: "bg-sky-brand text-ink-950 hover:bg-sky-brand-hi active:bg-sky-brand-lo",
    ghost: "border border-ink-600 text-ink-100 hover:border-ink-400 hover:bg-ink-800",
    danger: "border border-bad/60 text-bad hover:bg-bad/10",
  };
  return (
    <button className={`${base} ${styles[variant]} ${className}`} {...rest}>
      {children}
    </button>
  );
}

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <section className={`matte rounded-2xl border border-ink-700/80 p-6 ${className}`}>{children}</section>;
}

export function SectionTitle({ children, hint }: { children: ReactNode; hint?: ReactNode }) {
  return (
    <div className="mb-4 flex items-baseline justify-between gap-4">
      <h3 className="text-xs font-semibold uppercase tracking-[0.14em] text-ink-300">{children}</h3>
      {hint && <span className="text-xs text-ink-400">{hint}</span>}
    </div>
  );
}

export function Stat({ label, value, tone }: { label: ReactNode; value: ReactNode; tone?: "warn" | "bad" | "ok" }) {
  const color = tone === "bad" ? "text-bad" : tone === "warn" ? "text-warn" : tone === "ok" ? "text-sky-brand" : "text-ink-100";
  return (
    <div className="rounded-xl bg-ink-850 px-4 py-3">
      <div className="text-xs text-ink-400">{label}</div>
      <div className={`mt-1 text-lg font-semibold tabular-nums ${color}`}>{value}</div>
    </div>
  );
}

export function verdictTone(verdict: string): "ok" | "warn" | "bad" {
  return verdict === "bom" ? "ok" : verdict === "degradado" ? "warn" : "bad";
}

const toneColor = { ok: "var(--color-ok)", warn: "var(--color-warn)", bad: "var(--color-bad)" };

// ScoreRing mostra a nota de 0 a 100 num anel.
export function ScoreRing({ score, verdict, size = 132, label }: { score: number; verdict: string; size?: number; label?: string }) {
  const { t } = useI18n();
  const r = 52;
  const c = 2 * Math.PI * r;
  const tone = verdictTone(verdict);
  return (
    <div className="flex flex-col items-center gap-2">
      <svg width={size} height={size} viewBox="0 0 120 120" role="img" aria-label={`${score}/100`}>
        <circle cx="60" cy="60" r={r} fill="none" stroke="var(--color-ink-700)" strokeWidth="9" />
        <circle
          cx="60"
          cy="60"
          r={r}
          fill="none"
          stroke={toneColor[tone]}
          strokeWidth="9"
          strokeLinecap="round"
          strokeDasharray={`${(c * Math.max(0, Math.min(100, score))) / 100} ${c}`}
          transform="rotate(-90 60 60)"
          style={{ transition: "stroke-dasharray 600ms ease" }}
        />
        <text x="60" y="58" textAnchor="middle" fill="var(--color-ink-100)" fontSize="34" fontWeight="700">
          {score}
        </text>
        <text x="60" y="80" textAnchor="middle" fill="var(--color-ink-400)" fontSize="12">
          /100
        </text>
      </svg>
      {label && <span className="text-xs uppercase tracking-wider text-ink-400">{label}</span>}
      <span
        className="rounded-full px-3 py-0.5 text-xs font-semibold"
        style={{ color: toneColor[tone], background: `color-mix(in srgb, ${toneColor[tone]} 14%, transparent)` }}
      >
        {t(`diag.verdict.${verdict}` as Key)}
      </span>
    </div>
  );
}

function qualityColor(q: number) {
  if (q >= 0.85) return "var(--color-ok)";
  if (q >= 0.5) return "var(--color-warn)";
  return "var(--color-bad)";
}

// QualityStrip desenha a qualidade de cada bloco (~1 s) ao longo do vídeo.
export function QualityStrip({ blocks, duration }: { blocks: BlockReport[]; duration: number }) {
  const { t, num } = useI18n();
  if (!blocks.length) return null;
  const end = Math.max(duration, blocks[blocks.length - 1].tempo_video_s + 1);
  const ticks = [0, 0.25, 0.5, 0.75, 1].map((f) => f * end);
  return (
    <div>
      <div className="flex h-10 items-end gap-px overflow-hidden rounded-lg bg-ink-850 p-1">
        {blocks.map((b) => (
          <div
            key={b.indice}
            className="min-w-px flex-1 rounded-[2px]"
            title={`${t("unit.s", { v: num(b.tempo_video_s, 1) })} · ${Math.round(b.qualidade * 100)}%`}
            style={{ height: `${Math.max(12, b.qualidade * 100)}%`, background: qualityColor(b.qualidade) }}
          />
        ))}
      </div>
      <div className="mt-1.5 flex justify-between text-[11px] tabular-nums text-ink-400">
        {ticks.map((v, i) => (
          <span key={i}>{t("unit.s", { v: num(v, 0) })}</span>
        ))}
      </div>
    </div>
  );
}

export function Spinner({ className = "h-5 w-5" }: { className?: string }) {
  return (
    <svg className={`animate-spin text-sky-brand ${className}`} viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <circle cx="12" cy="12" r="10" stroke="currentColor" strokeOpacity="0.2" strokeWidth="3" />
      <path d="M22 12a10 10 0 0 0-10-10" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
    </svg>
  );
}

export const Icon = {
  video: (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className="h-6 w-6" aria-hidden="true">
      <rect x="2.5" y="6" width="13" height="12" rx="2.5" />
      <path d="m15.5 10.5 6-3.5v10l-6-3.5" />
    </svg>
  ),
  upload: (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className="h-8 w-8" aria-hidden="true">
      <path d="M12 16V4" />
      <path d="m7 9 5-5 5 5" />
      <path d="M4 16v2.5A1.5 1.5 0 0 0 5.5 20h13a1.5 1.5 0 0 0 1.5-1.5V16" />
    </svg>
  ),
  pin: (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-5 w-5" aria-hidden="true">
      <path d="M12 21s-7-6.3-7-11.6A7 7 0 0 1 19 9.4C19 14.7 12 21 12 21z" />
      <circle cx="12" cy="9.5" r="2.5" />
    </svg>
  ),
  check: (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" className="h-4 w-4" aria-hidden="true">
      <path d="m5 12.5 4.5 4.5L19 7.5" />
    </svg>
  ),
  folder: (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className="h-4 w-4" aria-hidden="true">
      <path d="M3 7.5A1.5 1.5 0 0 1 4.5 6h4l2 2h9A1.5 1.5 0 0 1 21 9.5v8a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 17.5z" />
    </svg>
  ),
  alert: (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-5 w-5" aria-hidden="true">
      <path d="M12 3 2 20h20z" />
      <path d="M12 10v4" />
      <path d="M12 17h.01" />
    </svg>
  ),
};
