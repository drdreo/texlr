# Texlr authoring reference

## Minimal document

```tex
\documentclass{texlr}

\title{Growth cofounder workstreams}
\author{Planning agent}
\date{\today}
\project{Growth}
\status{Handoff}
\repository{https://github.com/example/project}
\revision{main}

\begin{document}
\maketitle

\section{Executive summary}
State the purpose, current position, and most important next action.

\section{Workstreams}
Organize the source material into a structure that supports execution.

\begin{callout}[Open question]
State an unresolved issue without presenting it as a decision.
\end{callout}

\section{Dependencies and risks}
Separate dependencies, constraints, assumptions, and mitigations.

\section{Next actions}
Name concrete actions. Include owners or dates only when supported by the input.

\end{document}
```

Omit empty metadata commands. `\title`, `\author`, and `\date` should always be
present. Use `\runningtitle{Short title}` when the full title is too long for a
page header.

## Useful components

### Decision

```tex
\decision{Use a single acquisition funnel}{This reduces measurement ambiguity
while the first channel is being validated.}
```

### Code or commands

```tex
\begin{lstlisting}[language=bash,caption={Verification command}]
./bin/verify-growth-metrics
\end{lstlisting}
```

### Table

```tex
\begin{table}[ht]
  \centering
  \caption{Workstream summary.}
  \begin{tabularx}{\linewidth}{@{}l X l@{}}
    \toprule
    Workstream & Outcome & State \\
    \midrule
    Positioning & Validated message & Active \\
    Distribution & Repeatable channel & Proposed \\
    \bottomrule
  \end{tabularx}
\end{table}
```

### Ordinary image

```tex
\begin{figure}[ht]
  \centering
  \includegraphics[width=0.88\linewidth]{images/funnel.pdf}
  \caption{Current acquisition funnel.}
  \label{fig:funnel}
\end{figure}
```

Supported image formats are PNG, JPEG, and PDF. Convert SVG before inclusion.

### Graphviz

`diagrams/dependencies.dot`:

```dot
digraph Dependencies {
  graph [rankdir=LR, bgcolor="transparent"];
  node [shape=box, style="rounded,filled", fillcolor="#F5F7FB",
        color="#3157D5", fontname="Helvetica"];
  "Positioning" -> "Outbound tests";
  "Instrumentation" -> "Channel decision";
  "Outbound tests" -> "Channel decision";
}
```

LaTeX:

```tex
\begin{figure}[ht]
  \centering
  \graphviz[width=0.86\linewidth]{diagrams/dependencies.dot}
  \caption{Dependencies between workstreams.}
  \label{fig:dependencies}
\end{figure}
```

### Mermaid

`diagrams/sequence.mmd`:

```text
flowchart LR
    Research --> Positioning
    Positioning --> Experiments
    Experiments --> Measurement
    Measurement --> Decision
```

LaTeX:

```tex
\begin{figure}[ht]
  \centering
  \mermaid[width=0.9\linewidth]{diagrams/sequence.mmd}
  \caption{Validation sequence.}
\end{figure}
```

## LaTeX safety

Escape ordinary text containing these characters:

| Input | LaTeX |
| --- | --- |
| `&` | `\&` |
| `%` | `\%` |
| `$` | `\$` |
| `#` | `\#` |
| `_` | `\_` |
| `{` | `\{` |
| `}` | `\}` |

Use `\textasciitilde{}` and `\textasciicircum{}` for literal tilde and caret.
Use `\textbackslash{}` for a literal backslash. Put URLs in `\url{...}` rather
than escaping them manually.

Do not paste Markdown fences, headings, tables, or emphasis syntax into the
LaTeX source. Convert them to LaTeX environments and commands.

## Conversion guidance

When converting a plan or Markdown source:

1. Preserve its hierarchy and all substantive content.
2. Consolidate repetition only when meaning is unchanged.
3. Label unsupported ownership, timing, or status as open rather than guessing.
4. Turn relationship-heavy content into one useful diagram when appropriate.
5. Keep detailed checklists as lists or tables; do not bury them in prose.
6. Include a final section for decisions needed or immediate next actions when
   the source supports it.

## Failure handling

- `output_exists`: confirm replacement is intended, then retry with `--force`.
- `invalid_output`: correct overlapping or unsafe paths; do not bypass it.
- `diagram_failed`: inspect the DOT/Mermaid source and the reported log.
- `latex_failed`: inspect the retained build and compiler log, then fix the
  reported source line.
- Missing `texlr`: use the Nix fallback from `SKILL.md`.

A failed build intentionally retains its staging directory. Do not report the
failed staging path as the final source bundle.
