import { useQuery } from "@tanstack/react-query";
import Editor from "@monaco-editor/react";
import { getFileUrl } from "../../../lib/openapi";

interface PreviewProps {
  file: {
    name: string;
    path: string;
  };
}

export function CodePreview({ file }: PreviewProps) {
  const { data: content, isLoading, isError, error } = useQuery({
    queryKey: ["file-content", file.path],
    queryFn: async () => {
      const res = await fetch(getFileUrl(file.path));
      if (!res.ok) throw new Error(`Failed to load file: ${res.statusText}`);
      return res.text();
    },
  });

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
      {isLoading ? (
        <div className="flex items-center justify-center h-full">
          <div className="animate-spin w-8 h-8 border-2 border-accent border-t-transparent rounded-full" />
        </div>
      ) : isError ? (
        <div className="flex items-center justify-center h-full text-danger p-8">
          Error loading file: {(error as Error).message}
        </div>
      ) : (
        <Editor
          height="100%"
          defaultLanguage={langMap[ext] || "plaintext"}
          value={content || ""}
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
