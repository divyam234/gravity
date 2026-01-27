
import { MediaPlayer, MediaProvider } from "@vidstack/react";
import { defaultLayoutIcons, DefaultVideoLayout } from "@vidstack/react/player/layouts/default";
import "@vidstack/react/player/styles/default/theme.css";
import "@vidstack/react/player/styles/default/layouts/video.css";
import { getFileUrl } from "../../../lib/openapi";

interface PreviewProps {
  file: {
    name: string;
    path: string;
  };
}

export function VideoPreview({ file }: PreviewProps) {
  const url = getFileUrl(file.path);

  return (
    <div className="h-full w-full bg-black flex items-center justify-center">
      <MediaPlayer
        title={file.name}
        src={url}
        className="w-full h-full"
        aspectRatio="16/9"
        load="visible"
      >
        <MediaProvider />
        <DefaultVideoLayout icons={defaultLayoutIcons} />
      </MediaPlayer>
    </div>
  );
}
