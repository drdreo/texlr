---
name: texlr-handoff
description: Create polished PDF and self-contained LaTeX source handoff documents with Texlr. Use when the user asks for an agent handoff, implementation report, architecture brief, research report, delivery summary, polished plan, or conversion of Markdown/project findings into a designed document with images, Graphviz, or Mermaid.
license: MIT
compatibility: Requires Texlr and Nix. Mermaid rendering on macOS requires Google Chrome, Chromium, or Brave Browser.
---

# Texlr handoff

Use Texlr when the requested deliverable is a handoff or polished document. The
canonical source must be LaTeX; do not make Markdown the final artifact.

Before authoring, read `references/authoring.md` from this skill directory.

## Workflow

1. Resolve the requested input material and read it completely. For repository
   work, verify relevant facts against the source tree when practical. Do not
   invent results, decisions, owners, dates, or metrics.
2. Respect every user-supplied output path exactly. If paths are omitted, use:
   - `<repo>/artifacts/<slug>.pdf`
   - `<repo>/artifacts/<slug>-source/`
   - `<repo>/artifacts/<slug>.log`
3. Create the authoring tree in a temporary directory. Keep the main `.tex`,
   images, and all diagram sources beneath that one root. The published source
   bundle is the editable artifact; remove the temporary authoring tree after a
   successful build.
4. Author direct LaTeX with `\documentclass{texlr}`. Adapt the document structure
   to the material instead of forcing empty boilerplate sections.
5. Add figures only when they improve the handoff:
   - PNG, JPEG, or PDF: normal `\includegraphics`.
   - Graphviz: `\graphviz[<graphicx options>]{diagrams/name.dot}`.
   - Mermaid: `\mermaid[<graphicx options>]{diagrams/name.mmd}`.
   - Generate data plots separately and include their rendered image files.
6. Validate before publishing:

   ```bash
   texlr validate /absolute/path/to/document.tex --json
   ```

7. Build with explicit absolute paths:

   ```bash
   texlr build /absolute/path/to/document.tex \
     --pdf /absolute/path/to/handoff.pdf \
     --source /absolute/path/to/handoff-source \
     --log /absolute/path/to/handoff.log \
     --json
   ```

8. If an intentional rebuild targets existing Texlr artifacts, add `--force`.
   Never use `--force` to bypass an unrelated-directory safety error.
9. On failure, inspect the JSON `error`, `logPath`, and retained `workDir`; fix
   the source and retry until validation and build both succeed.
10. Verify the PDF, source directory, log, and `texlr-manifest.json` exist. Return
    those absolute paths with a one-paragraph content summary.

## Tool availability

Prefer the installed `texlr` command. If it is unavailable but Nix is present,
use `nix run github:drdreo/texlr --` in place of `texlr` and tell the user that
Texlr is not installed globally.

## Quality bar

- Preserve the source material's meaning and clearly distinguish facts,
  proposals, open questions, dependencies, and risks.
- Use a concise title, useful metadata, descriptive headings, captions, and
  cross-references.
- Escape LaTeX-special characters and use `\url{...}` for URLs.
- Prefer readable tables and diagrams over dense prose, but do not add
  decorative complexity.
- Do not claim completion until Texlr returns `"success": true` and the expected
  artifacts are present.
