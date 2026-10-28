import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { languages, useI18n, type Lang } from "./i18n";

// LanguageSelect substitui o <select> nativo: no Linux, o WebKitGTK desenha a
// lista do select com o tema do sistema e ignora as cores das opções, deixando
// o texto ilegível. Aqui a lista é desenhada pelo próprio app (fundo claro,
// texto preto fosco).
export function LanguageSelect() {
  const { t, lang, setLang } = useI18n();
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const root = useRef<HTMLDivElement>(null);
  const current = languages.find((l) => l.code === lang)!;

  // Fecha ao clicar fora.
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, [open]);

  const openList = () => {
    setActive(languages.findIndex((l) => l.code === lang));
    setOpen(true);
  };

  const choose = (code: Lang) => {
    setLang(code);
    setOpen(false);
  };

  const onKey = (e: KeyboardEvent) => {
    if (!open) {
      if (["ArrowDown", "ArrowUp", "Enter", " "].includes(e.key)) {
        e.preventDefault();
        openList();
      }
      return;
    }
    if (e.key === "Escape") {
      e.preventDefault();
      setOpen(false);
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((i) => (i + 1) % languages.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((i) => (i - 1 + languages.length) % languages.length);
    } else if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      choose(languages[active].code);
    } else if (e.key === "Tab") {
      setOpen(false);
    }
  };

  return (
    <div ref={root} className="relative flex items-center gap-2 text-xs text-ink-400">
      <span id="lang-label" className="sr-only sm:not-sr-only">
        {t("app.language")}
      </span>
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-labelledby="lang-label"
        onClick={() => (open ? setOpen(false) : openList())}
        onKeyDown={onKey}
        className="flex min-w-36 items-center justify-between gap-3 rounded-lg border border-ink-700 bg-ink-850 px-3 py-1.5 text-sm text-ink-100 hover:border-ink-600 focus:border-sky-brand focus:outline-none"
      >
        <span>{current.label}</span>
        <svg viewBox="0 0 20 20" className={`h-4 w-4 text-ink-400 transition ${open ? "rotate-180" : ""}`} fill="currentColor" aria-hidden="true">
          <path d="M5.5 7.5 10 12l4.5-4.5z" />
        </svg>
      </button>
      {open && (
        <ul
          role="listbox"
          aria-labelledby="lang-label"
          className="absolute right-0 top-full z-50 mt-1.5 min-w-full overflow-hidden rounded-lg border border-ink-300 bg-white py-1 shadow-xl shadow-black/40"
        >
          {languages.map((l, i) => {
            const selected = l.code === lang;
            return (
              <li
                key={l.code}
                role="option"
                aria-selected={selected}
                onMouseEnter={() => setActive(i)}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => choose(l.code)}
                className={`flex cursor-pointer items-center justify-between gap-4 px-3 py-1.5 text-sm text-ink-950 ${
                  i === active ? "bg-sky-brand/20" : ""
                } ${selected ? "font-semibold" : ""}`}
              >
                <span>{l.label}</span>
                {selected && (
                  <svg viewBox="0 0 20 20" className="h-4 w-4 text-sky-brand-lo" fill="none" stroke="currentColor" strokeWidth="2.4" aria-hidden="true">
                    <path d="m4.5 10.5 3.5 3.5 7.5-8" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
