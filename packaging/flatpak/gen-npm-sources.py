#!/usr/bin/env python3
"""Gera npm-sources.json: os pacotes npm do frontend como fontes do flatpak-builder.

Cada pacote de gui/frontend/package-lock.json vira um arquivo baixado do
registro do npm, conferido pelo sha512 que o próprio lockfile registra. No
build, os arquivos entram no cache do npm (`npm cache add`) e a instalação
roda offline (`npm ci --offline`).

Pacotes nativos de outros sistemas (Windows, macOS...) ficam de fora; os de
Linux recebem "only-arches" para baixar só o da arquitetura em uso.

Uso, na raiz do repositório:
    python3 packaging/flatpak/gen-npm-sources.py > packaging/flatpak/npm-sources.json
"""
import base64
import json
import os
import sys

root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
lock = json.load(open(os.path.join(root, "gui", "frontend", "package-lock.json")))
arches = {"x64": "x86_64", "arm64": "aarch64"}

sources, seen = [], set()
for path, pkg in sorted(lock.get("packages", {}).items()):
    url, integrity = pkg.get("resolved"), pkg.get("integrity")
    if not path or not url or not integrity or url in seen:
        continue
    oses, cpus = pkg.get("os"), pkg.get("cpu")
    if oses and "linux" not in oses:
        continue
    only = None
    if cpus:
        only = [arches[c] for c in cpus if c in arches]
        if not only:
            continue
    libc = pkg.get("libc")
    if libc and "glibc" not in libc:
        continue  # variantes musl
    algo, b64 = integrity.split()[0].split("-", 1)
    if algo != "sha512":
        sys.exit(f"{path}: integridade {algo} não suportada")
    seen.add(url)
    src = {
        "type": "file",
        "url": url,
        "sha512": base64.b64decode(b64).hex(),
        "dest": "npm-tarballs",
        "dest-filename": url.rsplit("/", 1)[1].replace("@", "") if url.count("/") else url,
    }
    # Nomes únicos (pacotes com escopo podem repetir o nome do arquivo).
    scope = path.split("node_modules/")[-1].split("/")[0]
    if scope.startswith("@"):
        src["dest-filename"] = scope[1:] + "-" + src["dest-filename"]
    if only:
        src["only-arches"] = only
    sources.append(src)

names = [s["dest-filename"] for s in sources]
dups = {n for n in names if names.count(n) > 1}
if dups:
    sys.exit(f"nomes repetidos: {dups}")
json.dump(sources, sys.stdout, indent=2)
print()
