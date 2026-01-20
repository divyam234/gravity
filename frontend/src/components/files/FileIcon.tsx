import React from "react";
import { cn } from "../../lib/utils";

// Gravity UI Icons
import IconFolder from "~icons/gravity-ui/folder";
import IconFile from "~icons/gravity-ui/file";
import IconFileCode from "~icons/gravity-ui/file-code";
import IconFileText from "~icons/gravity-ui/file-text";
import IconFileZipper from "~icons/gravity-ui/file-zipper";
import IconPicture from "~icons/gravity-ui/picture";
import IconVideo from "~icons/gravity-ui/video";
import IconMusic from "~icons/gravity-ui/music-note";
import IconFileP from "~icons/gravity-ui/file-letter-p";
import IconFileW from "~icons/gravity-ui/file-letter-w";
import IconFileX from "~icons/gravity-ui/file-letter-x";

interface IconConfig {
  icon: React.ComponentType<{ className?: string }>;
  className: string;
}

const defaultFile: IconConfig = { icon: IconFile, className: "text-muted-foreground bg-default/10" };

const iconMap: Record<string, IconConfig> = {
  // Documents
  pdf: { icon: IconFileP, className: "text-red-500 bg-red-500/10" },
  txt: { icon: IconFileText, className: "text-muted-foreground bg-default/10" },
  doc: { icon: IconFileW, className: "text-blue-600 bg-blue-600/10" },
  docx: { icon: IconFileW, className: "text-blue-600 bg-blue-600/10" },
  xls: { icon: IconFileX, className: "text-green-600 bg-green-600/10" },
  xlsx: { icon: IconFileX, className: "text-green-600 bg-green-600/10" },
  ppt: { icon: IconFileP, className: "text-orange-500 bg-orange-500/10" },
  pptx: { icon: IconFileP, className: "text-orange-500 bg-orange-500/10" },

  // Media
  png: { icon: IconPicture, className: "text-purple-500 bg-purple-500/10" },
  jpg: { icon: IconPicture, className: "text-purple-500 bg-purple-500/10" },
  jpeg: { icon: IconPicture, className: "text-purple-500 bg-purple-500/10" },
  gif: { icon: IconPicture, className: "text-purple-500 bg-purple-500/10" },
  svg: { icon: IconPicture, className: "text-orange-500 bg-orange-500/10" },
  webp: { icon: IconPicture, className: "text-purple-500 bg-purple-500/10" },
  mp4: { icon: IconVideo, className: "text-red-600 bg-red-600/10" },
  mkv: { icon: IconVideo, className: "text-red-600 bg-red-600/10" },
  mov: { icon: IconVideo, className: "text-red-600 bg-red-600/10" },
  avi: { icon: IconVideo, className: "text-red-600 bg-red-600/10" },
  mp3: { icon: IconMusic, className: "text-pink-500 bg-pink-500/10" },
  wav: { icon: IconMusic, className: "text-pink-500 bg-pink-500/10" },
  flac: { icon: IconMusic, className: "text-pink-500 bg-pink-500/10" },

  // Archive
  zip: { icon: IconFileZipper, className: "text-yellow-600 bg-yellow-600/10" },
  rar: { icon: IconFileZipper, className: "text-yellow-600 bg-yellow-600/10" },
  "7z": { icon: IconFileZipper, className: "text-yellow-600 bg-yellow-600/10" },
  tar: { icon: IconFileZipper, className: "text-yellow-600 bg-yellow-600/10" },
  gz: { icon: IconFileZipper, className: "text-yellow-600 bg-yellow-600/10" },

  // Code
  js: { icon: IconFileCode, className: "text-yellow-500 bg-yellow-500/10" },
  jsx: { icon: IconFileCode, className: "text-cyan-500 bg-cyan-500/10" },
  ts: { icon: IconFileCode, className: "text-blue-500 bg-blue-500/10" },
  tsx: { icon: IconFileCode, className: "text-blue-500 bg-blue-500/10" },
  py: { icon: IconFileCode, className: "text-blue-400 bg-blue-400/10" },
  html: { icon: IconFileCode, className: "text-orange-600 bg-orange-600/10" },
  css: { icon: IconFileCode, className: "text-blue-600 bg-blue-600/10" },
  json: { icon: IconFileCode, className: "text-yellow-200 bg-yellow-200/10" },
  md: { icon: IconFileText, className: "text-foreground bg-default/10" },
  go: { icon: IconFileCode, className: "text-cyan-500 bg-cyan-500/10" },
  rs: { icon: IconFileCode, className: "text-orange-700 bg-orange-700/10" },
  dockerfile: { icon: IconFileCode, className: "text-blue-500 bg-blue-500/10" },
  xml: { icon: IconFileCode, className: "text-orange-400 bg-orange-400/10" },
  yaml: { icon: IconFileCode, className: "text-purple-400 bg-purple-400/10" },
  yml: { icon: IconFileCode, className: "text-purple-400 bg-purple-400/10" },
  sh: { icon: IconFileCode, className: "text-green-500 bg-green-500/10" },
  bash: { icon: IconFileCode, className: "text-green-500 bg-green-500/10" },
};

export function FileIcon({ name, isDir, className }: { name: string; isDir: boolean; className?: string }) {
  if (isDir) {
    return (
      <div className={cn("flex items-center justify-center rounded-lg bg-warning/10 text-warning shrink-0", className)}>
        <IconFolder className="w-6 h-6" />
      </div>
    );
  }

  const parts = name.split('.');
  const ext = parts.length > 1 ? parts.pop()?.toLowerCase() : "";
  
  const config = (ext && iconMap[ext]) || defaultFile;
  
  return (
    <div className={cn("flex items-center justify-center rounded-lg shrink-0", config.className, className)}>
      <config.icon className="w-6 h-6" />
    </div>
  );
}