#!/usr/bin/env python3
"""Gera go-sources.json: os módulos Go da interface como fontes do flatpak-builder.

O Flathub compila sem internet. Cada módulo listado em gui/go.sum vira um
arquivo baixado de proxy.golang.org (com sha256) e colocado num "proxy local"
(goproxy/), que o manifesto usa com GOPROXY=file://. O go.sum continua
conferindo cada módulo.

Uso (na raiz do repositório, com os módulos já baixados no cache do Go):
    (cd gui && go mod download)
    python3 packaging/flatpak/gen-go-sources.py > packaging/flatpak/go-sources.json
"""
import hashlib
import json
import os
import subprocess
import sys

root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
gosum = os.path.join(root, "gui", "go.sum")
cache = subprocess.check_output(["go", "env", "GOMODCACHE"], text=True).strip()


def escape(path):
    # Escapa maiúsculas como o proxy do Go: "A" -> "!a".
    return "".join("!" + c.lower() if c.isupper() else c for c in path)


need = {}  # (módulo, versão) -> {"mod", "zip"}
for line in open(gosum):
    mod, ver, _ = line.split()
    if ver.endswith("/go.mod"):
        need.setdefault((mod, ver[: -len("/go.mod")]), set()).add("mod")
    else:
        need.setdefault((mod, ver), set()).update({"mod", "zip"})

sources = []
for (mod, ver), kinds in sorted(need.items()):
    for kind in sorted(kinds):
        rel = f"{escape(mod)}/@v/{ver}.{kind}"
        local = os.path.join(cache, "cache", "download", rel)
        if not os.path.exists(local):
            sys.exit(f"falta no cache: {local} (rode: cd gui && go mod download)")
        sha = hashlib.sha256(open(local, "rb").read()).hexdigest()
        sources.append({
            "type": "file",
            "url": f"https://proxy.golang.org/{rel}",
            "sha256": sha,
            "dest": f"goproxy/{escape(mod)}/@v",
            "dest-filename": f"{ver}.{kind}",
        })

json.dump(sources, sys.stdout, indent=2)
print()
