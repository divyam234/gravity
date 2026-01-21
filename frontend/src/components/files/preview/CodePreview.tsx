import React, { useEffect, useState } from "react";
import Editor from "@monaco-editor/react";
import { api } from "../../../lib/api";

interface PreviewProps {
  file: {
    name: string;
    path: string;
  };
}

export function CodePreview({ file }: PreviewProps) {
  const [content, setContent] = useState<string>("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    fetch(api.getFileUrl(file.path))
      .then((res) => res.text())
      .then((text) => {
        setContent(text);
        setLoading(false);
      })
      .catch((err) => {
        setContent(`Error loading file: ${err.message}`);
        setLoading(false);
      });
  }, [file.path]);

  const ext = file.name.split(".").pop()?.toLowerCase() || "";
  const langMap: Record<string, string> = {
    ts: "typescript",
    tsx: "typescript",
    js: "javascript",
    jsx: "javascript",
    py: "python",
    go: "go",
    rs: "rust",
    cpp: "cpp",
    c: "c",
    h: "cpp",
    json: "json",
    yml: "yaml",
    yaml: "yaml",
    md: "markdown",
    sql: "sql",
    sh: "shell",
    html: "html",
    css: "css",
  };

  return (
    <div className="h-full w-full">
      {loading ? (
        <div className="flex items-center justify-center h-full">
          <div className="animate-spin w-8 h-8 border-2 border-accent border-t-transparent rounded-full" />
        </div>
      ) : (
        <Editor
          height="100%"
          defaultLanguage={langMap[ext] || "plaintext"}
          value={content}
          theme="vs-dark"
          options={{
            readOnly: true,
            minimap: { enabled: true },
            fontSize: 14,
            scrollBeyondLastLine: false,
            automaticLayout: true,
          }}
        />
      )}
    </div>
  );
}
