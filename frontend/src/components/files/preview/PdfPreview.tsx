import React from "react";
import { api } from "../../../lib/api";

interface PreviewProps {
  file: {
    name: string;
    path: string;
  };
}

export function PdfPreview({ file }: PreviewProps) {
  const url = api.getFileUrl(file.path);

  return (
    <div className="h-full w-full bg-white">
      <iframe 
        src={url} 
        title={file.name}
        className="w-full h-full border-none"
      />
    </div>
  );
}
