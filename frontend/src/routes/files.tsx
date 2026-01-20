import { createFileRoute } from "@tanstack/react-router";
import { FileBrowser } from "../components/files/FileBrowser";
import { z } from "zod";

const filesSearchSchema = z.object({
  path: z.string().optional().default("/"),
});

export const Route = createFileRoute("/files")({
  validateSearch: filesSearchSchema,
  component: FilesPage,
});

function FilesPage() {
  const { path } = Route.useSearch();
  
  return (
    <div className="h-full w-full flex flex-col overflow-hidden">
      <FileBrowser path={path} />
    </div>
  );
}
