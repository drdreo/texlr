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
\RequirePackage[top=30mm,bottom=32mm,left=30mm,right=30mm]{geometry}
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

\definecolor{texlrInk}{HTML}{172033}
\definecolor{texlrMuted}{HTML}{667085}
\definecolor{texlrAccent}{HTML}{3157D5}
\definecolor{texlrRule}{HTML}{D8DEE9}
\definecolor{texlrPanel}{HTML}{F5F7FB}
\definecolor{texlrCode}{HTML}{F3F5F8}

\hypersetup{
  colorlinks=true,
  linkcolor=texlrAccent,
  urlcolor=texlrAccent,
  citecolor=texlrAccent,
  pdfborder={0 0 0}
}

\setlength{\parindent}{0pt}
\setlength{\parskip}{0.65em}
\setlist{nosep,leftmargin=1.5em,topsep=0.35em}
\captionsetup{font=small,labelfont=bf,labelsep=period}

\titleformat{\section}
  {\Large\sffamily\bfseries\color{texlrInk}}
  {\thesection}{0.75em}{}
\titleformat{\subsection}
  {\large\sffamily\bfseries\color{texlrInk}}
  {\thesubsection}{0.65em}{}
\titleformat{\subsubsection}
  {\normalsize\sffamily\bfseries\color{texlrInk}}
  {\thesubsubsection}{0.55em}{}
\titlespacing*{\section}{0pt}{2.1em}{0.7em}
\titlespacing*{\subsection}{0pt}{1.5em}{0.45em}

\pagestyle{fancy}
\setlength{\headheight}{22pt}
\fancyhf{}
\renewcommand{\headrulewidth}{0.4pt}
\renewcommand{\headrule}{\hbox to\headwidth{\color{texlrRule}\leaders\hrule height \headrulewidth\hfill}}
\fancyhead[L]{\small\sffamily\color{texlrMuted}\texlrRunningTitle}
\fancyhead[R]{\small\sffamily\color{texlrMuted}\texlrStatus}
\fancyfoot[C]{\small\sffamily\color{texlrMuted}\thepage}

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
  \thispagestyle{empty}%
  \begingroup
  \sffamily
  {\small\bfseries\color{texlrAccent}\MakeUppercase{\texlrStatus}\par}
  \vspace{0.8em}
  {\huge\bfseries\color{texlrInk}\@title\par}
  \ifdefempty{\texlrProject}{}{%
    \vspace{0.7em}%
    {\large\color{texlrMuted}\texlrProject\par}%
  }
  \vspace{1.8em}
  \color{texlrRule}\hrule height 1pt
  \vspace{1em}
  \begin{tabularx}{\textwidth}{@{}>{\bfseries\color{texlrMuted}}l X@{}}
    Author & \@author \\
    Date & \@date \\
    \ifdefempty{\texlrRepository}{}{Repository & \url{\texlrRepository} \\}
    \ifdefempty{\texlrRevision}{}{Revision & \texttt{\texlrRevision} \\}
  \end{tabularx}
  \vspace{1em}
  \color{texlrRule}\hrule height 1pt
  \endgroup
  \vspace{2em}
  \ifdefempty{\texlrRunningTitle}{\global\let\texlrRunningTitle\@title}{}
}
\makeatother

\lstdefinestyle{texlr}{
  backgroundcolor=\color{texlrCode},
  basicstyle=\ttfamily\small,
  breaklines=true,
  frame=single,
  rulecolor=\color{texlrRule},
  framerule=0.5pt,
  numbers=left,
  numberstyle=\tiny\color{texlrMuted},
  xleftmargin=1.6em,
  framexleftmargin=1.2em,
  showstringspaces=false,
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
   {\sffamily\bfseries\color{texlrAccent}#1}\par\smallskip
   \color{texlrInk}}
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
