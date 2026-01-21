import React, { useState, useMemo, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { api } from "../../lib/api";
import { formatBytes, cn } from "../../lib/utils";
import { FileIcon } from "./FileIcon";
import { FilePreview } from "./preview/FilePreview";
import { useSettingsStore } from "../../store/useSettingsStore";
import IconFolder from "~icons/gravity-ui/folder";
import IconChevronRight from "~icons/gravity-ui/chevron-right";
import IconPlus from "~icons/gravity-ui/plus";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import IconArrowUp from "~icons/gravity-ui/arrow-up";
import IconPencil from "~icons/gravity-ui/pencil";
import IconCopy from "~icons/gravity-ui/copy";
import IconArrowRightFromSquare from "~icons/gravity-ui/arrow-right-from-square";
import IconLayoutHeaderCellsLarge from "~icons/gravity-ui/layout-header-cells-large";
import IconCircleCheckFill from "~icons/gravity-ui/circle-check-fill";
import {
  Button,
  Dropdown,
  Modal,
  TextField,
  Input,
  ListBox,
  ScrollShadow,
  Label,
  Chip,
} from "@heroui/react";
import type { Selection } from "@heroui/react";
import { toast } from "sonner";

interface ClipboardItem {
  op: "copy" | "move";
  paths: string[];
}

function useDisclosure() {
  const [isOpen, setIsOpen] = useState(false);
  return {
    isOpen,
    onOpen: () => setIsOpen(true),
    onClose: () => setIsOpen(false),
    onOpenChange: (open: boolean) => setIsOpen(open),
  };
}

import IconCheck from "~icons/gravity-ui/check";

interface FileBrowserProps {
  path: string;
  query?: string;
}

type ModalType = "create" | "rename" | null;

import IconMagnifyingGlass from "~icons/gravity-ui/magnifier";
import IconGear from "~icons/gravity-ui/gear";

export function FileBrowser({ path, query }: FileBrowserProps) {
  const navigateRouter = useNavigate();
  const [clipboard, setClipboard] = useState<ClipboardItem | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<Selection>(new Set([]));
  const [isSelectionMode, setIsSelectionMode] = useState(false);
  const [previewFile, setPreviewFile] = useState<any | null>(null);
  const { searchQuery, setSearchQuery } = useSettingsStore();
  const queryClient = useQueryClient();

  // Sync prop query to store
  React.useEffect(() => {
    if (query !== undefined && query !== searchQuery) {
      setSearchQuery(query);
    }
  }, [query]);

  // Unified Modal state
  const modal = useDisclosure();
  const [modalType, setModalType] = useState<ModalType>(null);
  const [modalInputValue, setModalInputValue] = useState("");
  const [renameOldPath, setRenameOldPath] = useState("");

  // Context Menu State
  const [menuOpen, setMenuOpen] = useState(false);
  const [menuPosition, setMenuPosition] = useState<{
    x: number;
    y: number;
  } | null>(null);
  const [menuFile, setMenuFile] = useState<any | null>(null);
  const menuTriggerRef = useRef<HTMLDivElement>(null);

  const {
    data: filesResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["files", path],
    queryFn: () => api.listFiles(path),
    enabled: !searchQuery,
  });

  const { data: searchResponse } = useQuery({
    queryKey: ["files", "search", searchQuery],
    queryFn: () => api.search(searchQuery),
    enabled: !!searchQuery,
  });

  const files = useMemo(() => {
    const list = searchQuery
      ? searchResponse?.data || []
      : filesResponse?.data || [];
    return [...list].sort((a, b) => {
      if (a.isDir && !b.isDir) return -1;
      if (!a.isDir && b.isDir) return 1;
      return a.name.localeCompare(b.name);
    });
  }, [searchQuery, searchResponse, filesResponse]);

  const navigate = (newPath: string) => {
    setSelectedKeys(new Set([]));
    setSearchQuery(""); // Clear search when navigating to a specific path
    navigateRouter({
      to: "/files",
      search: { path: newPath, query: undefined },
    });
  };

  const navigateUp = () => {
    if (path === "/") return;
    const parent = path.split("/").slice(0, -1).join("/") || "/";
    navigate(parent);
  };

  // Breadcrumbs
  const parts = path.split("/").filter(Boolean);
  const breadcrumbs = parts.map((part, index) => {
    const fullPath = "/" + parts.slice(0, index + 1).join("/");
    return { name: part, path: fullPath };
  });

  // Mutations
  const deleteMutation = useMutation({
    mutationFn: (filePaths: string[]) =>
      Promise.all(filePaths.map((p) => api.deleteFile(p))),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files", path] });
      setSelectedKeys(new Set([]));
      toast.success("Items deleted");
    },
  });

  const mkdirMutation = useMutation({
    mutationFn: (folderPath: string) => api.mkdir(folderPath),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files", path] });
      modal.onClose();
      setModalInputValue("");
      toast.success("Folder created");
    },
  });

  const operateMutation = useMutation({
    mutationFn: (vars: {
      op: "rename" | "copy" | "move";
      items: { src: string; dst: string }[];
    }) =>
      Promise.all(
        vars.items.map((item) => api.operateFile(vars.op, item.src, item.dst)),
      ),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: ["files", path] });
      setSelectedKeys(new Set([]));
      if (vars.op === "rename") {
        modal.onClose();
        setModalInputValue("");
        toast.success("Renamed successfully");
      } else {
        toast.success(`${vars.op === "copy" ? "Copy" : "Move"} started`);
        setClipboard(null);
      }
    },
    onError: (err: any) => {
      toast.error("Operation failed: " + err.message);
    },
  });

  // Handlers
  const handleModalConfirm = () => {
    if (!modalInputValue) return;

    if (modalType === "create") {
      const fullPath =
        path === "/" ? `/${modalInputValue}` : `${path}/${modalInputValue}`;
      mkdirMutation.mutate(fullPath);
    } else if (modalType === "rename") {
      const parent = renameOldPath.split("/").slice(0, -1).join("/");
      const dst =
        parent === "" ? `/${modalInputValue}` : `${parent}/${modalInputValue}`;
      operateMutation.mutate({
        op: "rename",
        items: [{ src: renameOldPath, dst }],
      });
    }
  };

  const handlePaste = () => {
    if (!clipboard) return;

    const items = clipboard.paths.map((srcPath) => {
      const filename = srcPath.split("/").pop();
      const dst = path === "/" ? `/${filename}` : `${path}/${filename}`;
      return { src: srcPath, dst };
    });

    operateMutation.mutate({
      op: clipboard.op,
      items,
    });
  };
  const selectedPaths = useMemo(() => {
    if (selectedKeys === "all") return files.map((f) => f.path);
    return Array.from(selectedKeys).map((k) => String(k));
  }, [selectedKeys, files]);

  const handleBulkDelete = () => {
    if (selectedPaths.length === 0) return;
    deleteMutation.mutate(selectedPaths);
  };

  const handleBulkClipboard = (op: "copy" | "move") => {
    if (selectedPaths.length === 0) return;
    setClipboard({ op, paths: selectedPaths });
    toast.success(
      `${op === "copy" ? "Copied" : "Cut"} ${selectedPaths.length} items`,
    );
  };

  if (isLoading)
    return (
      <div className="p-12 flex justify-center h-full items-center">
        <div className="animate-spin w-8 h-8 border-2 border-accent border-t-transparent rounded-full" />
      </div>
    );
  if (error)
    return (
      <div className="p-12 text-center text-danger h-full flex flex-col justify-center">
        Error loading files
      </div>
    );

  return (
    <div className="flex flex-col h-full w-full bg-background select-none">
      {/* Toolbar */}
      <div className="flex items-center gap-2 p-4 border-b border-border bg-default/5 shrink-0 h-16">
        <Button
          isIconOnly
          size="sm"
          variant="ghost"
          className="rounded-xl"
          onPress={navigateUp}
          isDisabled={path === "/"}
        >
          <IconArrowUp />
        </Button>

        <div className="flex items-center gap-1 text-sm flex-1 overflow-hidden px-2 no-scrollbar">
          {searchQuery ? (
            <div className="flex items-center gap-2 font-bold text-accent">
              <IconMagnifyingGlass className="w-4 h-4" />
              <span>Search results for "{searchQuery}"</span>
              <Button
                size="sm"
                variant="ghost"
                className="h-7 min-w-0 px-2 rounded-lg text-muted hover:text-foreground"
                onPress={() => {
                  setSearchQuery("");
                  navigateRouter({
                    to: "/files",
                    search: { path, query: undefined },
                  });
                }}
              >
                Clear
              </Button>
            </div>
          ) : (
            breadcrumbs.map((b) => (
              <React.Fragment key={b.path}>
                <IconChevronRight className="w-4 h-4 text-muted/50" />
                <Button
                  size="sm"
                  variant="ghost"
                  className={cn(
                    "rounded-xl font-bold",
                    b.path === path ? "text-accent bg-accent/10" : "text-muted",
                  )}
                  onPress={() => navigate(b.path)}
                >
                  {b.name}
                </Button>
              </React.Fragment>
            ))
          )}
        </div>

        <div className="flex items-center gap-2">
          <Button
            size="sm"
            isIconOnly
            variant="ghost"
            onPress={() => navigateRouter({ to: "/settings/search" })}
            className="rounded-xl"
          >
            <IconGear />
          </Button>

          <Button
            size="sm"
            isIconOnly
            variant={isSelectionMode ? "primary" : "ghost"}
            onPress={() => {
              setIsSelectionMode(!isSelectionMode);
              if (isSelectionMode) setSelectedKeys(new Set([]));
            }}
          >
            <IconCheck />
          </Button>

          {selectedPaths.length > 0 && (
            <>
              <Button
                size="sm"
                variant="ghost"
                className="rounded-xl font-bold text-danger hover:bg-danger/10"
                onPress={handleBulkDelete}
              >
                <IconTrashBin className="w-4 h-4 mr-1" />
                Delete ({selectedPaths.length})
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="rounded-xl font-bold"
                onPress={() => handleBulkClipboard("copy")}
              >
                <IconCopy className="w-4 h-4 mr-1" />
                Copy
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="rounded-xl font-bold"
                onPress={() => handleBulkClipboard("move")}
              >
                <IconArrowRightFromSquare className="w-4 h-4 mr-1" />
                Cut
              </Button>
              <div className="w-px h-6 bg-border mx-1" />
            </>
          )}

          {clipboard && (
            <Button
              size="sm"
              variant="secondary"
              className="rounded-xl font-bold shadow-lg shadow-secondary/20"
              onPress={handlePaste}
            >
              <IconLayoutHeaderCellsLarge className="w-4 h-4 mr-1" />
              Paste
            </Button>
          )}

          <Button
            size="sm"
            isIconOnly
            variant="primary"
            onPress={() => {
              setModalType("create");
              setModalInputValue("");
              modal.onOpen();
            }}
          >
            <IconPlus />
          </Button>
        </div>
      </div>

      {/* File List using ListBox */}
      <div className="flex-1 overflow-hidden bg-background">
        <ScrollShadow className="h-full px-4 py-4 no-scrollbar">
          {files.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-32 text-muted">
              <div className="w-20 h-20 bg-default/10 rounded-full flex items-center justify-center mb-4">
                <IconFolder className="w-10 h-10 opacity-50" />
              </div>
              <p className="text-xl font-bold tracking-tight">Empty folder</p>
              <p className="text-sm">There are no files here yet.</p>
            </div>
          ) : (
            <ListBox
              aria-label="File List"
              items={files}
              selectionMode={isSelectionMode ? "multiple" : "none"}
              selectedKeys={selectedKeys}
              onSelectionChange={setSelectedKeys}
              onAction={(key) => {
                if (isSelectionMode) {
                  setSelectedKeys((prev) => {
                    const currentSet = new Set(
                      prev === "all" ? files.map((f) => f.path) : prev,
                    );
                    if (currentSet.has(key)) currentSet.delete(key);
                    else currentSet.add(key);
                    return currentSet;
                  });
                  return;
                }
                 const file = files.find((f) => f.path === key);
                 if (file && file.isDir) {
                   navigate(file.path);
                 } else if (file) {
                   setPreviewFile(file);
                 }
               }}
              className="p-0 gap-2 w-full"
            >
              {(file) => (
                <ListBox.Item
                  id={file.path}
                  textValue={file.name}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    setMenuPosition({ x: e.clientX, y: e.clientY });
                    setMenuFile(file);
                    setMenuOpen(true);
                  }}
                  className={cn(
                    "px-3 py-2 rounded-xl cursor-pointer transition-all duration-200 group mb-1",
                    "hover:bg-default/10 hover:border-border",
                    "data-[selected=true]:bg-accent/10 data-[selected=true]:border-accent/30 flex items-center gap-3 w-full",
                  )}
                >
                  <FileIcon
                    name={file.name}
                    isDir={file.isDir}
                    className="w-10 h-10 transition-transform duration-200 group-hover:scale-105 shadow-sm"
                  />

                  <div className="flex-1 min-w-0 flex items-center justify-between gap-4">
                    <div className="flex flex-col min-w-0 gap-1">
                      <div className="flex items-center gap-2">
                        <Label className="font-semibold truncate text-foreground text-sm tracking-tight leading-none cursor-pointer">
                          {file.name}
                        </Label>
                        {searchQuery && file.remote && (
                          <Chip
                            size="sm"
                            variant="soft"
                            color="accent"
                            className="h-4 text-[9px] px-1 font-black uppercase tracking-tighter shrink-0"
                          >
                            {file.remote}
                          </Chip>
                        )}
                      </div>
                      {searchQuery && (
                        <p className="text-[10px] text-muted truncate opacity-60">
                          {file.path}
                        </p>
                      )}
                    </div>

                    <div className="flex items-center gap-3 shrink-0 text-[10px] sm:text-xs text-muted font-medium">
                      {!file.isDir && (
                        <span className="bg-default/10 px-1.5 py-0.5 rounded text-foreground/70 whitespace-nowrap">
                          {formatBytes(file.size)}
                        </span>
                      )}

                      <span className="hidden sm:flex items-center gap-1 opacity-70 whitespace-nowrap">
                        {file.modTime &&
                          new Date(file.modTime).toLocaleDateString(undefined, {
                            year: "numeric",
                            month: "short",
                            day: "numeric",
                          })}
                      </span>
                    </div>
                  </div>

                  <ListBox.ItemIndicator>
                    {({ isSelected }) =>
                      isSelected ? (
                        <IconCircleCheckFill className="size-4 text-accent" />
                      ) : null
                    }
                  </ListBox.ItemIndicator>
                </ListBox.Item>
              )}
            </ListBox>
          )}
        </ScrollShadow>
      </div>

      {/* Shared Context Menu */}
      <div
        ref={menuTriggerRef}
        style={{
          position: "fixed",
          top: menuPosition?.y ?? 0,
          left: menuPosition?.x ?? 0,
          width: 0,
          height: 0,
        }}
      />
      <Dropdown
        isOpen={menuOpen}
        onOpenChange={(open) => !open && setMenuOpen(false)}
      >
        <Dropdown.Trigger className="hidden">
          <span />
        </Dropdown.Trigger>
        <Dropdown.Popover
          triggerRef={menuTriggerRef}
          offset={0}
          placement="bottom start"
          className="min-w-[200px] bg-surface border border-border shadow-2xl rounded-2xl p-1"
        >
          <Dropdown.Menu
            dependencies={[menuFile]}
            aria-label="File Actions"
            onAction={(key) => {
              console.log("jfojdfaljflk", menuFile, key);
              if (!menuFile) return;
              const file = menuFile;
              setMenuOpen(false);

              if (key === "open" && file.isDir) navigate(file.path);
              if (key === "delete") deleteMutation.mutateAsync([file.path]);
              if (key === "rename") {
                setModalType("rename");
                setModalInputValue(file.name);
                setRenameOldPath(file.path);
                modal.onOpen();
              }
              if (key === "copy") {
                setClipboard({
                  op: "copy",
                  paths: [file.path],
                });
                toast.success("Copied to clipboard");
              }
              if (key === "cut") {
                setClipboard({
                  op: "move",
                  paths: [file.path],
                });
                toast.success("Cut to clipboard");
              }
            }}
          >
            <Dropdown.Item
              id="open"
              textValue="Open"
              className="rounded-xl py-2.5 px-3"
            >
              <IconFolder className="size-4 shrink-0" />
              <Label className="font-bold">Open</Label>
            </Dropdown.Item>
            <Dropdown.Item
              id="copy"
              textValue="Copy"
              className="rounded-xl py-2.5 px-3"
            >
              <IconCopy className="size-4 shrink-0" />
              <Label className="font-bold">Copy</Label>
            </Dropdown.Item>
            <Dropdown.Item
              id="cut"
              textValue="Cut"
              className="rounded-xl py-2.5 px-3"
            >
              <IconArrowRightFromSquare className="size-4 shrink-0" />
              <Label className="font-bold">Cut</Label>
            </Dropdown.Item>
            <Dropdown.Item
              id="rename"
              textValue="Rename"
              className="rounded-xl py-2.5 px-3"
            >
              <IconPencil className="size-4 shrink-0" />
              <Label className="font-bold">Rename</Label>
            </Dropdown.Item>
            <Dropdown.Item
              id="delete"
              variant="danger"
              textValue="Delete"
              className="rounded-xl py-2.5 px-3 bg-danger/10 text-danger"
            >
              <IconTrashBin className="size-4 shrink-0" />
              <Label className="font-bold">Delete</Label>
            </Dropdown.Item>
          </Dropdown.Menu>
        </Dropdown.Popover>
      </Dropdown>

      {/* Unified Modal for all Operations */}
      <Modal.Backdrop isOpen={modal.isOpen} onOpenChange={modal.onOpenChange}>
        <Modal.Container>
          <Modal.Dialog className="bg-surface border border-border shadow-2xl rounded-2xl">
            <Modal.Header className="p-6 pb-2">
              <Modal.Heading className="text-xl font-bold">
                {modalType === "create" ? "Create New Folder" : "Rename Item"}
              </Modal.Heading>
            </Modal.Header>
            <Modal.Body className="px-6 py-4">
              <TextField
                value={modalInputValue}
                onChange={setModalInputValue}
                autoFocus
                className="w-full"
              >
                <Label className="text-sm font-semibold mb-1.5 block">
                  {modalType === "create" ? "Folder Name" : "New Name"}
                </Label>
                <Input
                  placeholder={
                    modalType === "create"
                      ? "Enter folder name"
                      : "Enter new name"
                  }
                  className="h-11 bg-default/10 border-none rounded-xl"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleModalConfirm();
                  }}
                />
              </TextField>
            </Modal.Body>
            <Modal.Footer className="p-6 pt-2 flex gap-2">
              <Button
                variant="ghost"
                className="flex-1 rounded-xl font-bold"
                onPress={() => modal.onClose()}
              >
                Cancel
              </Button>
              <Button
                variant="primary"
                className="flex-1 rounded-xl font-bold"
                onPress={handleModalConfirm}
              >
                {modalType === "create" ? "Create" : "Rename"}
              </Button>
             </Modal.Footer>
           </Modal.Dialog>
         </Modal.Container>
       </Modal.Backdrop>

       <FilePreview
         file={previewFile}
         onClose={() => setPreviewFile(null)}
       />
     </div>
   );
 }
