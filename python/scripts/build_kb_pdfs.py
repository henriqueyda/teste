"""Compile the markdown policy sources in kb/sources/ into PDFs in kb/policies/.

The KB documents are PDFs (closer to a real bank's policy library) but are authored as
markdown for easy editing/versioning ("documents as code"). Run via `make kb`.

Supports a small markdown subset: `# title`, `## heading`, `### subheading`,
`- bullet`, blank lines, and paragraphs. Uses fpdf2 core fonts (latin-1 / Portuguese).
"""
from __future__ import annotations

from pathlib import Path

from fpdf import FPDF

ROOT = Path(__file__).resolve().parents[2]
SRC_DIR = ROOT / "kb" / "sources"
OUT_DIR = ROOT / "kb" / "policies"


def render(md_path: Path, pdf_path: Path) -> None:
    pdf = FPDF(format="A4")
    pdf.set_auto_page_break(auto=True, margin=20)
    pdf.add_page()
    pdf.set_margins(20, 20, 20)

    def write(font_style: str, size: float, height: float, text: str) -> None:
        pdf.set_font("Helvetica", font_style, size)
        # new_x/new_y reset the cursor to the left margin on the next line
        pdf.multi_cell(0, height, text, new_x="LMARGIN", new_y="NEXT")

    for raw in md_path.read_text(encoding="utf-8").splitlines():
        line = raw.rstrip()
        if not line:
            pdf.ln(3)
        elif line.startswith("### "):
            write("B", 11, 6, line[4:])
        elif line.startswith("## "):
            write("B", 13, 7, line[3:])
        elif line.startswith("# "):
            write("B", 16, 9, line[2:])
        elif line.startswith("- "):
            write("", 11, 6, f"  -  {line[2:]}")
        else:
            write("", 11, 6, line)

    pdf.output(str(pdf_path))


def main() -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    sources = sorted(SRC_DIR.glob("*.md"))
    if not sources:
        raise SystemExit(f"no markdown sources found in {SRC_DIR}")
    for md in sources:
        out = OUT_DIR / (md.stem + ".pdf")
        render(md, out)
        print(f"built {out.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
