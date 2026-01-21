import React from "react";
import { Modal, Button } from "@heroui/react";
import IconX from "~icons/gravity-ui/xmark";
import { CodePreview } from "./CodePreview";
import { VideoPreview } from "./VideoPreview";
import { ImagePreview } from "./ImagePreview";
import { PdfPreview } from "./PdfPreview";
import { EpubPreview } from "./EpubPreview";

interface FilePreviewProps {
  file: {
    name: string;
    path: string;
    mimeType?: string;
  } | null;
  onClose: () => void;
}

export function FilePreview({ file, onClose }: FilePreviewProps) {
  if (!file) return null;

  const ext = file.name.split(".").pop()?.toLowerCase() || "";

  const isVideo = ["mp4", "mkv", "webm", "avi", "mov"].includes(ext);
  const isImage = ["jpg", "jpeg", "png", "gif", "webp", "svg"].includes(ext);
  const isPdf = ext === "pdf";
  const isEpub = ext === "epub";
  const isCode = [
    "ts",
    "tsx",
    "js",
    "jsx",
    "go",
    "py",
    "rs",
    "cpp",
    "c",
    "h",
    "java",
    "html",
    "css",
    "json",
    "yaml",
    "yml",
    "md",
    "txt",
    "sh",
    "sql",
  ].includes(ext);

  return (
    <Modal.Backdrop
      isOpen={!!file}
      variant="opaque"
      onOpenChange={(open) => !open && onClose()}
    >
      <Modal.Container className="will-change-auto">
        <Modal.Dialog className="max-w-5xl  p-0 bg-surface border border-border shadow-2xl rounded-2xl flex flex-col overflow-hidden h-full">
          <Modal.Header className="p-4 border-b border-border flex items-center justify-between shrink-0">
            <Modal.Heading className="text-lg font-bold truncate pr-8">
              {file.name}
            </Modal.Heading>
            <Button
              isIconOnly
              size="sm"
              variant="ghost"
              onPress={onClose}
              className="rounded-xl absolute right-4 top-4"
            >
              <IconX />
            </Button>
          </Modal.Header>
          <Modal.Body className="p-0 flex-1 overflow-hidden bg-black/5">
            {isVideo && <VideoPreview file={file} />}
            {isImage && <ImagePreview file={file} />}
            {isPdf && <PdfPreview file={file} />}
            {isEpub && <EpubPreview file={file} />}
            {isCode && <CodePreview file={file} />}
            {!isVideo && !isImage && !isPdf && !isEpub && !isCode && (
              <div className="flex flex-col items-center justify-center h-full text-muted">
                <p>No preview available for this file type.</p>
                <p className="text-sm mt-2">
                  {file.mimeType || "Unknown type"}
                </p>
              </div>
            )}
          </Modal.Body>
        </Modal.Dialog>
      </Modal.Container>
    </Modal.Backdrop>
  );
}
