% texlr.cls
% A polished class for agent handoff documents.
%
% Derived in part from HagenbergThesis by Wilhelm Burger and Wolfgang
% Hochleitner: https://github.com/Digital-Media/HagenbergThesis
% Used with permission and under CC BY 4.0. Modified and renamed for Texlr.
% Texlr is not affiliated with or endorsed by the upstream authors or
% FH Upper Austria. No upstream logos or institutional branding are included.
%
% SPDX-License-Identifier: CC-BY-4.0
\NeedsTeXFormat{LaTeX2e}
\ProvidesClass{texlr}[2026/08/30 Agent handoff documents]

\LoadClass[a4paper,11pt,oneside]{article}

\RequirePackage[T1]{fontenc}
\RequirePackage{lmodern}
\RequirePackage{microtype}
\RequirePackage{etoolbox}
% Match the single-column HgbArticle page geometry.
\RequirePackage[top=36mm,bottom=40mm,left=32mm,right=32mm]{geometry}
\RequirePackage{xcolor}
\RequirePackage{graphicx}
\RequirePackage{booktabs}
\RequirePackage{tabularx}
\RequirePackage{array}
\RequirePackage{enumitem}
\RequirePackage{amsmath}
\RequirePackage{amssymb}
\RequirePackage{listings}
\RequirePackage{caption}
\RequirePackage{fancyhdr}
\RequirePackage{titlesec}
\RequirePackage{hyperref}
\RequirePackage{url}

% HgbArticle is intentionally monochrome. Keep Texlr's additions neutral and
% high-contrast rather than introducing a separate brand palette.
\colorlet{texlrInk}{black}
\definecolor{texlrMuted}{gray}{0.25}
\colorlet{texlrAccent}{black}
\definecolor{texlrRule}{gray}{0.70}
\definecolor{texlrPanel}{gray}{0.95}
\definecolor{texlrCode}{gray}{0.95}

\hypersetup{
  colorlinks=false,
  pdfborder={0 0 0}
}
\AtBeginDocument{\urlstyle{sf}}

\frenchspacing
\raggedbottom
\setlength{\parindent}{1.5em}
\setlength{\parskip}{0pt plus 0.1pt}
\setlist{
  partopsep=0ex plus 0.1ex,
  topsep=0.6ex plus 0.3ex,
  itemsep=0ex plus 0.1ex,
  parsep=0.4ex plus 0.2ex
}
\captionsetup{font=small,labelfont=bf,labelsep=period,margin=5mm}
\setlength{\abovecaptionskip}{10pt}
\setlength{\belowcaptionskip}{10pt}

\titleformat{\section}
  {\Large\sffamily\bfseries}
  {\thesection}{0.75em}{}
\titleformat{\subsection}
  {\large\sffamily\bfseries}
  {\thesubsection}{0.65em}{}
\titleformat{\subsubsection}
  {\normalsize\sffamily\bfseries}
  {\thesubsubsection}{0.55em}{}
\titlespacing*{\section}{0pt}{2.1em}{0.7em}
\titlespacing*{\subsection}{0pt}{1.5em}{0.45em}

% Follow hgbheadings: sans-serif running title, page number at right, no rule.
\pagestyle{fancy}
\setlength{\headheight}{22pt}
\fancyhf{}
\renewcommand{\headrulewidth}{0pt}
\renewcommand{\footrulewidth}{0pt}
\fancyhead[L]{\sffamily\texlrRunningTitle}
\fancyhead[R]{\sffamily\thepage}
\fancypagestyle{plain}{%
  \fancyhf{}%
  \fancyfoot[C]{\sffamily\thepage}%
  \renewcommand{\headrulewidth}{0pt}%
  \renewcommand{\footrulewidth}{0pt}%
}

\newcommand{\texlrProject}{}
\newcommand{\texlrStatus}{Handoff}
\newcommand{\texlrRepository}{}
\newcommand{\texlrRevision}{}
\newcommand{\texlrRunningTitle}{}
\newcommand{\project}[1]{\renewcommand{\texlrProject}{#1}}
\newcommand{\status}[1]{\renewcommand{\texlrStatus}{#1}}
\newcommand{\repository}[1]{\renewcommand{\texlrRepository}{#1}}
\newcommand{\revision}[1]{\renewcommand{\texlrRevision}{#1}}
\newcommand{\runningtitle}[1]{\renewcommand{\texlrRunningTitle}{#1}}

\makeatletter
\renewcommand{\maketitle}{%
  \thispagestyle{plain}%
  \begingroup
  \sffamily
  \null\vspace{2em}%
  \begin{center}
    {\LARGE\bfseries\@title\par}%
    \ifdefempty{\texlrProject}{}{%
      \vspace{0.9em}%
      {\large\texlrProject\par}%
    }%
    \vspace{1.2em}%
    {\large\@author\par}%
    \vspace{0.35em}%
    {\normalsize\@date\par}%
    \ifdefempty{\texlrStatus}{}{%
      \vspace{0.75em}%
      {\small\texlrStatus\par}%
    }%
    \ifdefempty{\texlrRepository}{}{%
      \vspace{0.25em}%
      {\small\url{\texlrRepository}\par}%
    }%
    \ifdefempty{\texlrRevision}{}{%
      \vspace{0.25em}%
      {\small\texlrRevision\par}%
    }%
  \end{center}
  \endgroup
  \vspace{2em}%
  \ifdefempty{\texlrRunningTitle}{\global\let\texlrRunningTitle\@title}{}%
}
\makeatother

\lstdefinestyle{texlr}{
  backgroundcolor=\color{texlrCode},
  basicstyle=\ttfamily\footnotesize,
  breaklines=true,
  frame=none,
  numbers=left,
  numbersep=5pt,
  numberstyle=\normalfont\scriptsize,
  xleftmargin=6mm,
  framexleftmargin=6mm,
  showstringspaces=false,
  columns=flexible,
  keepspaces=true,
  tabsize=2
}
\lstset{style=texlr}

\newsavebox{\texlrCalloutBox}
\newenvironment{callout}[1][Note]
  {\par\medskip
   \begingroup
   \setlength{\fboxsep}{10pt}%
   \begin{lrbox}{\texlrCalloutBox}%
   \begin{minipage}{\dimexpr\linewidth-2\fboxsep-2\fboxrule\relax}
   {\sffamily\bfseries #1}\par\smallskip}
  {\end{minipage}%
   \end{lrbox}%
   \noindent\fcolorbox{texlrRule}{texlrPanel}{\usebox{\texlrCalloutBox}}%
   \endgroup
   \par\medskip}

% Texlr renders <source>.dot and <source>.mmd diagram inputs to <source>.pdf
% before compilation. The optional argument accepts normal graphicx options.
\newcommand{\graphviz}[2][]{\includegraphics[#1]{#2.pdf}}
\newcommand{\mermaid}[2][]{\includegraphics[#1]{#2.pdf}}

\newcommand{\decision}[2]{%
  \begin{callout}[Decision: #1]#2\end{callout}%
}

\endinput
