import React from "react";
import { PhotoProvider, PhotoView } from "react-photo-view";
import "react-photo-view/dist/react-photo-view.css";
import { api } from "../../../lib/api";

interface PreviewProps {
  file: {
    name: string;
    path: string;
  };
}

export function ImagePreview({ file }: PreviewProps) {
  const url = api.getFileUrl(file.path);

  return (
    <div className="h-full w-full flex items-center justify-center p-8 overflow-auto">
      <PhotoProvider>
        <PhotoView src={url}>
          <img 
            src={url} 
            alt={file.name} 
            className="max-w-full max-h-full object-contain cursor-zoom-in shadow-2xl rounded"
          />
        </PhotoView>
      </PhotoProvider>
    </div>
  );
}
