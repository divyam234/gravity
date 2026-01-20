import React, { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { api } from "../../lib/api";
import { formatBytes, cn } from "../../lib/utils";
import IconFolder from "~icons/gravity-ui/folder";
import IconFile from "~icons/gravity-ui/file";
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
    Description
} from "@heroui/react";
import type { Selection } from "@heroui/react";
import { toast } from "sonner";

interface ClipboardItem {
    op: 'copy' | 'move';
    paths: string[];
}

function useDisclosure() {
    const [isOpen, setIsOpen] = useState(false);
    return {
        isOpen,
        onOpen: () => setIsOpen(true),
        onClose: () => setIsOpen(false),
        onOpenChange: (open: boolean) => setIsOpen(open)
    };
}

interface InputModalProps {
    isOpen: boolean;
    onOpenChange: (open: boolean) => void;
    title: string;
    label: string;
    value: string;
    onValueChange: (val: string) => void;
    onConfirm: () => void;
    confirmLabel: string;
}

function InputModal({ isOpen, onOpenChange, title, label, value, onValueChange, onConfirm, confirmLabel }: InputModalProps) {
    return (
        <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
            <Modal.Backdrop className="bg-background/80 backdrop-blur-sm" />
            <Modal.Container>
                <Modal.Dialog className="bg-content1 border border-border shadow-2xl rounded-2xl">
                    <Modal.Header className="p-6 pb-2">
                        <Modal.Heading className="text-xl font-bold">{title}</Modal.Heading>
                    </Modal.Header>
                    <Modal.Body className="px-6 py-4">
                        <TextField value={value} onChange={onValueChange} autoFocus className="w-full">
                            <Label className="text-sm font-semibold mb-1.5 block">{label}</Label>
                            <Input 
                                placeholder={`Enter ${label.toLowerCase()}`}
                                className="h-11 bg-default/10 border-none rounded-xl"
                                onKeyDown={(e) => {
                                    if (e.key === "Enter") onConfirm();
                                }}
                            />
                        </TextField>
                    </Modal.Body>
                    <Modal.Footer className="p-6 pt-2 flex gap-2">
                        <Button variant="ghost" className="flex-1 rounded-xl font-bold" onPress={() => onOpenChange(false)}>
                            Cancel
                        </Button>
                        <Button variant="primary" className="flex-1 rounded-xl font-bold" onPress={onConfirm}>
                            {confirmLabel}
                        </Button>
                    </Modal.Footer>
                </Modal.Dialog>
            </Modal.Container>
        </Modal>
    );
}

interface FileBrowserProps {
    path: string;
}

export function FileBrowser({ path }: FileBrowserProps) {
    const navigateRouter = useNavigate();
    const [clipboard, setClipboard] = useState<ClipboardItem | null>(null);
    const [selectedKeys, setSelectedKeys] = useState<Selection>(new Set([]));
    const queryClient = useQueryClient();
    
    // Modals
    const createFolderModal = useDisclosure();
    const renameModal = useDisclosure();
    
    const [newFolderName, setNewFolderName] = useState("");
    const [renameState, setRenameState] = useState({ oldPath: "", newName: "" });

    const { data: filesResponse, isLoading, error } = useQuery({
        queryKey: ["files", path],
        queryFn: () => api.listFiles(path),
    });

    const files = useMemo(() => filesResponse?.data || [], [filesResponse]);

    const navigate = (newPath: string) => {
        setSelectedKeys(new Set([]));
        navigateRouter({
            to: "/files",
            search: { path: newPath }
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
        mutationFn: (filePaths: string[]) => Promise.all(filePaths.map(p => api.deleteFile(p))),
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
            createFolderModal.onClose();
            setNewFolderName("");
            toast.success("Folder created");
        },
    });

    const operateMutation = useMutation({
        mutationFn: (vars: { op: 'rename' | 'copy' | 'move', items: { src: string, dst: string }[] }) => 
            Promise.all(vars.items.map(item => api.operateFile(vars.op, item.src, item.dst))),
        onSuccess: (_data, vars) => {
            queryClient.invalidateQueries({ queryKey: ["files", path] });
            setSelectedKeys(new Set([]));
            if (vars.op === 'rename') {
                renameModal.onClose();
                toast.success("Renamed successfully");
            } else {
                toast.success(`${vars.op === 'copy' ? 'Copy' : 'Move'} started`);
                setClipboard(null);
            }
        },
        onError: (err: any) => {
            toast.error("Operation failed: " + err.message);
        }
    });

    // Handlers
    const handleCreateFolder = () => {
        if (!newFolderName) return;
        const fullPath = path === "/" ? `/${newFolderName}` : `${path}/${newFolderName}`;
        mkdirMutation.mutate(fullPath);
    };

    const handleRename = () => {
        if (!renameState.newName) return;
        const parent = renameState.oldPath.split("/").slice(0, -1).join("/");
        const dst = parent === "" ? `/${renameState.newName}` : `${parent}/${renameState.newName}`;
        
        operateMutation.mutate({
            op: 'rename',
            items: [{ src: renameState.oldPath, dst }]
        });
    };

    const handlePaste = () => {
        if (!clipboard) return;
        
        const items = clipboard.paths.map(srcPath => {
            const filename = srcPath.split("/").pop();
            const dst = path === "/" ? `/${filename}` : `${path}/${filename}`;
            return { src: srcPath, dst };
        });
        
        operateMutation.mutate({
            op: clipboard.op,
            items
        });
    };

    const selectedPaths = useMemo(() => {
        if (selectedKeys === "all") return files.map(f => f.path);
        return Array.from(selectedKeys).map(k => String(k));
    }, [selectedKeys, files]);

    const handleBulkDelete = () => {
        if (selectedPaths.length === 0) return;
        deleteMutation.mutate(selectedPaths);
    };

    const handleBulkClipboard = (op: 'copy' | 'move') => {
        if (selectedPaths.length === 0) return;
        setClipboard({ op, paths: selectedPaths });
        toast.success(`${op === 'copy' ? 'Copied' : 'Cut'} ${selectedPaths.length} items`);
    };

    if (isLoading) return <div className="p-12 flex justify-center h-full items-center"><div className="animate-spin w-8 h-8 border-2 border-primary border-t-transparent rounded-full"/></div>;
    if (error) return <div className="p-12 text-center text-danger h-full flex flex-col justify-center">Error loading files</div>;

    return (
        <div className="flex flex-col h-full w-full bg-background select-none">
            {/* Toolbar */}
            <div className="flex items-center gap-2 p-4 border-b border-border bg-default/5 shrink-0 h-16">
                <Button isIconOnly size="sm" variant="ghost" className="rounded-xl" onPress={navigateUp} isDisabled={path === "/"}>
                     <IconArrowUp />
                </Button>
                
                <div className="flex items-center gap-1 text-sm flex-1 overflow-hidden px-2 no-scrollbar">
                    <Button 
                        size="sm" 
                        variant="ghost" 
                        className={cn("rounded-xl font-bold", path === "/" ? "text-primary bg-primary/10" : "text-muted")}
                        onPress={() => navigate("/")}
                    >
                        Home
                    </Button>
                    {breadcrumbs.map((b) => (
                        <React.Fragment key={b.path}>
                            <IconChevronRight className="w-4 h-4 text-muted/50" />
                            <Button 
                                size="sm" 
                                variant="ghost"
                                className={cn("rounded-xl font-bold", b.path === path ? "text-primary bg-primary/10" : "text-muted")}
                                onPress={() => navigate(b.path)}
                            >
                                {b.name}
                            </Button>
                        </React.Fragment>
                    ))}
                </div>

                <div className="flex items-center gap-2">
                    {selectedPaths.length > 0 && (
                        <>
                            <Button size="sm" variant="ghost" className="rounded-xl font-bold text-danger hover:bg-danger/10" onPress={handleBulkDelete}>
                                <IconTrashBin className="w-4 h-4 mr-1" />
                                Delete ({selectedPaths.length})
                            </Button>
                            <Button size="sm" variant="ghost" className="rounded-xl font-bold" onPress={() => handleBulkClipboard('copy')}>
                                <IconCopy className="w-4 h-4 mr-1" />
                                Copy
                            </Button>
                            <Button size="sm" variant="ghost" className="rounded-xl font-bold" onPress={() => handleBulkClipboard('move')}>
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

                    <Button size="sm" variant="primary" className="rounded-xl font-bold shadow-lg shadow-primary/20 px-4" onPress={createFolderModal.onOpen}>
                        <IconPlus className="w-4 h-4 mr-1" />
                        New Folder
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
                            selectionMode="multiple"
                            selectedKeys={selectedKeys}
                            onSelectionChange={setSelectedKeys}
                            onAction={(key) => {
                                const file = files.find(f => f.path === key);
                                if (file && file.isDir) {
                                    navigate(file.path);
                                }
                            }}
                            className="p-0 gap-2 w-full"
                        >
                            {(file) => (
                                <ListBox.Item 
                                    id={file.path} 
                                    textValue={file.name}
                                    className={cn(
                                        "px-4 py-4 rounded-2xl cursor-pointer transition-all duration-200 border-2 border-transparent outline-none group mb-2",
                                        "hover:bg-default/10 hover:border-border",
                                        "data-[selected=true]:bg-primary/10 data-[selected=true]:border-primary/30"
                                    )}
                                >
                                    <div className="flex items-center gap-4 w-full">
                                        <div className="relative">
                                            <div className={cn(
                                                "w-12 h-12 flex items-center justify-center rounded-xl transition-transform duration-200 group-hover:scale-110 shadow-sm",
                                                file.isDir ? "bg-warning/10 text-warning" : "bg-primary/10 text-primary"
                                            )}>
                                                {file.isDir ? (
                                                    <IconFolder className="w-7 h-7" />
                                                ) : (
                                                    <IconFile className="w-7 h-7" />
                                                )}
                                            </div>
                                            <div className={cn(
                                                "absolute -top-1 -right-1 w-5 h-5 bg-primary text-white rounded-full flex items-center justify-center scale-0 transition-transform duration-200 shadow-md",
                                                "group-data-[selected=true]:scale-100"
                                            )}>
                                                <IconCircleCheckFill className="w-4 h-4" />
                                            </div>
                                        </div>
                                        
                                        <div className="flex-1 min-w-0 flex flex-col justify-center">
                                            <Label className="font-bold truncate text-foreground text-lg tracking-tight mb-0.5">
                                                {file.name}
                                            </Label>
                                            <Description className="text-xs text-muted flex items-center gap-3 font-medium">
                                                {!file.isDir && <span className="bg-default/10 px-2 py-0.5 rounded-md font-bold">{formatBytes(file.size)}</span>}
                                                <span className="flex items-center gap-1">
                                                    {file.modTime && new Date(file.modTime).toLocaleDateString(undefined, {
                                                        year: 'numeric',
                                                        month: 'short',
                                                        day: 'numeric'
                                                    })}
                                                </span>
                                                {file.isDir && <span className="uppercase text-[10px] font-black tracking-widest bg-warning/10 text-warning px-2 py-0.5 rounded-md">Folder</span>}
                                            </Description>
                                        </div>

                                        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                            <Dropdown>
                                                <Dropdown.Trigger>
                                                    <Button isIconOnly size="sm" variant="ghost" className="rounded-xl h-10 w-10 min-w-0" onPress={e => e.continuePropagation()}>
                                                        ⋮
                                                    </Button>
                                                </Dropdown.Trigger>
                                                <Dropdown.Popover className="min-w-[200px] bg-content1 border border-border shadow-2xl rounded-2xl p-1">
                                                    <Dropdown.Menu aria-label="File Actions" onAction={(key) => {
                                                        if (key === "open" && file.isDir) navigate(file.path);
                                                        if (key === "delete") deleteMutation.mutate([file.path]);
                                                        if (key === "rename") {
                                                            setRenameState({ oldPath: file.path, newName: file.name });
                                                            renameModal.onOpen();
                                                        }
                                                        if (key === "copy") {
                                                            setClipboard({ op: 'copy', paths: [file.path] });
                                                            toast.success("Copied to clipboard");
                                                        }
                                                        if (key === "cut") {
                                                            setClipboard({ op: 'move', paths: [file.path] });
                                                            toast.success("Cut to clipboard");
                                                        }
                                                    }}>
                                                        <Dropdown.Item key="open" textValue="Open" className="rounded-xl py-2.5 px-3">
                                                            <div className="flex items-center gap-3">
                                                                <IconFolder className="w-4 h-4 text-muted" />
                                                                <Label className="font-bold">Open</Label>
                                                            </div>
                                                        </Dropdown.Item>
                                                        <Dropdown.Item key="copy" textValue="Copy" className="rounded-xl py-2.5 px-3">
                                                            <div className="flex items-center gap-3">
                                                                <IconCopy className="w-4 h-4 text-muted"/>
                                                                <Label className="font-bold">Copy</Label>
                                                            </div>
                                                        </Dropdown.Item>
                                                        <Dropdown.Item key="cut" textValue="Cut" className="rounded-xl py-2.5 px-3">
                                                            <div className="flex items-center gap-3">
                                                                <IconArrowRightFromSquare className="w-4 h-4 text-muted"/>
                                                                <Label className="font-bold">Cut</Label>
                                                            </div>
                                                        </Dropdown.Item>
                                                        <Dropdown.Item key="rename" textValue="Rename" className="rounded-xl py-2.5 px-3">
                                                            <div className="flex items-center gap-3">
                                                                <IconPencil className="w-4 h-4 text-muted"/>
                                                                <Label className="font-bold">Rename</Label>
                                                            </div>
                                                        </Dropdown.Item>
                                                        <Dropdown.Item 
                                                            key="delete" 
                                                            variant="danger"
                                                            textValue="Delete"
                                                            className="rounded-xl py-2.5 px-3 bg-danger/10 text-danger"
                                                        >
                                                            <div className="flex items-center gap-3">
                                                                <IconTrashBin className="w-4 h-4" />
                                                                <Label className="font-bold">Delete</Label>
                                                            </div>
                                                        </Dropdown.Item>
                                                    </Dropdown.Menu>
                                                </Dropdown.Popover>
                                            </Dropdown>
                                        </div>
                                    </div>
                                </ListBox.Item>
                            )}
                        </ListBox>
                    )}
                </ScrollShadow>
            </div>

            {/* Create Folder Modal */}
            <InputModal 
                isOpen={createFolderModal.isOpen}
                onOpenChange={createFolderModal.onOpenChange}
                title="Create New Folder"
                label="Folder Name"
                value={newFolderName}
                onValueChange={setNewFolderName}
                onConfirm={handleCreateFolder}
                confirmLabel="Create"
            />

            {/* Rename Modal */}
            <InputModal 
                isOpen={renameModal.isOpen}
                onOpenChange={renameModal.onOpenChange}
                title="Rename Item"
                label="New Name"
                value={renameState.newName}
                onValueChange={(val) => setRenameState(prev => ({ ...prev, newName: val }))}
                onConfirm={handleRename}
                confirmLabel="Rename"
            />
        </div>
    );
}
