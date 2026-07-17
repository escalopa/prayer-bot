#!/usr/bin/env python3
"""Validate the static calculation site and its downloadable artifacts."""

from html.parser import HTMLParser
import hashlib
import json
from pathlib import Path
from urllib.parse import urlparse


ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"
PDF = ROOT / "output" / "pdf" / "prayer-time-calculations.pdf"


class PageParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.ids: list[str] = []
        self.links: list[str] = []

    def handle_starttag(self, _tag: str, attrs: list[tuple[str, str | None]]) -> None:
        values = dict(attrs)
        if values.get("id"):
            self.ids.append(values["id"] or "")
        if values.get("href"):
            self.links.append(values["href"] or "")


def main() -> None:
    html = (DOCS / "index.html").read_text(encoding="utf-8")
    parser = PageParser()
    parser.feed(html)
    parser.close()

    duplicates = sorted({value for value in parser.ids if parser.ids.count(value) > 1})
    assert not duplicates, f"duplicate HTML ids: {duplicates}"

    ids = set(parser.ids)
    for link in parser.links:
        if link.startswith("#"):
            assert link[1:] in ids, f"missing anchor target: {link}"
            continue
        parsed = urlparse(link)
        if parsed.scheme or link.startswith("//"):
            continue
        clean = parsed.path.removeprefix("./")
        if clean == "downloads/prayer-time-calculations.pdf":
            target = PDF
        else:
            target = DOCS / clean
        assert target.exists(), f"missing local link target: {link}"

    for method in (
        "Muslim World League",
        "Egyptian General Authority",
        "Umm al-Qura",
        "Karachi",
        "ISNA",
        "Diyanet",
        "Kemenag",
        "MUIS",
        "JAKIM",
    ):
        assert method in html, f"missing method from public page: {method}"

    tex = (DOCS / "calculation-methods.tex").read_text(encoding="utf-8")
    for marker in ("\\section{Solar geometry}", "\\section{High-latitude rules}", "\\section{Gregorian and Hijri dates}"):
        assert marker in tex, f"missing LaTeX section: {marker}"

    assert PDF.stat().st_size > 10_000, "compiled PDF is unexpectedly small"
    assert PDF.read_bytes().startswith(b"%PDF-"), "download is not a PDF"

    manifest = json.loads((DOCS / "artifact-manifest.json").read_text(encoding="utf-8"))
    for artifact in (manifest["source"], manifest["pdf"]):
        path = ROOT / artifact["path"]
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        assert digest == artifact["sha256"], f"stale compiled artifact: {path}"
    print("Calculation documentation is internally consistent.")


if __name__ == "__main__":
    main()
