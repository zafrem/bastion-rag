#!/usr/bin/env python3
"""Generate OpenAPI specs + Redoc HTML for the FastAPI modules.

Navigator and Anchor are FastAPI apps, so their OpenAPI schema is generated
directly from the route definitions and pydantic models — no hand-authoring.
The apps are built with mock dependencies (MagicMock); only the route/model
metadata is needed, the handlers are never executed.

Run from the repo root:  python3 scripts/gen-openapi.py
Outputs: docs/api/<module>/openapi.json and docs/api/<module>/index.html
"""
from __future__ import annotations

import json
import pathlib
import sys
from unittest.mock import MagicMock

ROOT = pathlib.Path(__file__).resolve().parent.parent
DOCS = ROOT / "docs" / "api"

REDOC_HTML = """<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{title}</title>
  <style>body {{ margin: 0; padding: 0; }}</style>
</head>
<body>
  <!-- Spec is in the same directory; Redoc is loaded from a CDN (needs internet). -->
  <redoc spec-url="openapi.json"></redoc>
  <script src="https://cdn.redocly.com/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>
"""


def gen_navigator():
    sys.path.insert(0, str(ROOT / "navigator"))
    from navigator.config import Config  # noqa: E402
    from navigator.rest import build_app  # noqa: E402

    # Real config (build_app may read config values at route-registration time);
    # service dependencies are mocked since handlers are never executed.
    cfg = Config.load(str(ROOT / "navigator" / "config" / "config.yaml"))
    app = build_app(cfg, MagicMock(), MagicMock(), MagicMock())
    return app.openapi()


def gen_anchor():
    sys.path.insert(0, str(ROOT / "anchor"))
    from anchor.config import Config  # noqa: E402
    from anchor.rest import build_app  # noqa: E402

    cfg = Config.load(str(ROOT / "anchor" / "config" / "config.yaml"))
    app = build_app(cfg, MagicMock(), MagicMock())
    return app.openapi()


def write(module: str, spec: dict) -> None:
    out = DOCS / module
    out.mkdir(parents=True, exist_ok=True)
    (out / "openapi.json").write_text(json.dumps(spec, indent=2) + "\n")
    title = spec.get("info", {}).get("title", module) + " API Reference"
    (out / "index.html").write_text(REDOC_HTML.format(title=title))
    print(f"{module:10s} {len(spec.get('paths', {}))} paths, "
          f"{len(spec.get('components', {}).get('schemas', {}))} schemas "
          f"-> {out.relative_to(ROOT)}")


if __name__ == "__main__":
    write("navigator", gen_navigator())
    write("anchor", gen_anchor())
