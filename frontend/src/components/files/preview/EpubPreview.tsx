import { useState } from "react";
import { ReactReader } from "react-reader";
import { api } from "../../../lib/api";

interface PreviewProps {
  file: {
    name: string;
    path: string;
  };
}

export function EpubPreview({ file }: PreviewProps) {
  const [location, setLocation] = useState<string | number>(0);
  const url = api.getFileUrl(file.path);

  return (
    <div className="h-full w-full bg-white relative">
      <ReactReader
        url={url}
        location={location}
        locationChanged={(epubcfi: string) => setLocation(epubcfi)}
        title={file.name}
        epubOptions={{
          flow: "scrolled",
          manager: "continuous",
        }}
      />
    </div>
  );
}
