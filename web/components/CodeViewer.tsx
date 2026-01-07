"use client";

import { useMemo } from "react";
import hljs from "highlight.js/lib/core";
import javascript from "highlight.js/lib/languages/javascript";
import typescript from "highlight.js/lib/languages/typescript";
import go from "highlight.js/lib/languages/go";
import python from "highlight.js/lib/languages/python";
import java from "highlight.js/lib/languages/java";
import c from "highlight.js/lib/languages/c";
import cpp from "highlight.js/lib/languages/cpp";
import ruby from "highlight.js/lib/languages/ruby";
import php from "highlight.js/lib/languages/php";
import css from "highlight.js/lib/languages/css";
import xml from "highlight.js/lib/languages/xml";
import json from "highlight.js/lib/languages/json";
import yaml from "highlight.js/lib/languages/yaml";
import markdown from "highlight.js/lib/languages/markdown";
import bash from "highlight.js/lib/languages/bash";
import dockerfile from "highlight.js/lib/languages/dockerfile";
import rust from "highlight.js/lib/languages/rust";
import "highlight.js/styles/github-dark.css";

hljs.registerLanguage("javascript", javascript);
hljs.registerLanguage("typescript", typescript);
hljs.registerLanguage("go", go);
hljs.registerLanguage("python", python);
hljs.registerLanguage("java", java);
hljs.registerLanguage("c", c);
hljs.registerLanguage("cpp", cpp);
hljs.registerLanguage("ruby", ruby);
hljs.registerLanguage("php", php);
hljs.registerLanguage("css", css);
hljs.registerLanguage("xml", xml);
hljs.registerLanguage("json", json);
hljs.registerLanguage("yaml", yaml);
hljs.registerLanguage("markdown", markdown);
hljs.registerLanguage("bash", bash);
hljs.registerLanguage("dockerfile", dockerfile);
hljs.registerLanguage("rust", rust);

function extToLanguage(ext: string): string | undefined {
  const map: Record<string, string> = {
    js: "javascript",
    jsx: "javascript",
    ts: "typescript",
    tsx: "typescript",
    go: "go",
    py: "python",
    java: "java",
    c: "c",
    h: "c",
    cpp: "cpp",
    cc: "cpp",
    cxx: "cpp",
    rb: "ruby",
    php: "php",
    css: "css",
    html: "xml",
    xml: "xml",
    json: "json",
    yml: "yaml",
    yaml: "yaml",
    md: "markdown",
    sh: "bash",
    bash: "bash",
    dockerfile: "dockerfile",
    rs: "rust",
    makefile: "bash",
  };
  return map[ext.toLowerCase()];
}

export function CodeViewer({
  content,
  filePath,
}: {
  content: string;
  filePath: string;
}) {
  const ext = useMemo(() => {
    const name = filePath.split("/").pop() || "";
    if (name.toLowerCase() === "dockerfile") return "dockerfile";
    const parts = name.split(".");
    return parts.length > 1 ? parts.pop() || "" : "";
  }, [filePath]);

  const language = useMemo(() => extToLanguage(ext || ""), [ext]);
  const lines = useMemo(() => content.split("\n"), [content]);

  return (
    <div className="overflow-x-auto text-sm font-mono leading-6 bg-panel">
      <table className="w-full border-collapse">
        <tbody>
          {lines.map((line, i) => {
            let highlighted = "";
            try {
              if (language) {
                highlighted = hljs.highlight(line, { language }).value;
              } else {
                highlighted = hljs.highlightAuto(line).value;
              }
            } catch {
              highlighted = line;
            }
            return (
              <tr key={i}>
                <td className="w-12 text-right select-none text-muted bg-panel pr-4 border-r border-base py-0.5">
                  {i + 1}
                </td>
                <td
                  className="pl-4 whitespace-pre text-base py-0.5"
                  dangerouslySetInnerHTML={{ __html: highlighted || line }}
                />
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

