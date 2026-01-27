import {
  Tree,
  TreeItem,
  TreeItemContent,
  Collection,
  Button,
  Checkbox,
  type Key,
  type Selection,
} from "react-aria-components";
import { useState, useMemo } from "react";
import IconChevronRight from "~icons/gravity-ui/chevron-right";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconFolder from "~icons/gravity-ui/folder";
import IconFile from "~icons/gravity-ui/file";
import { formatBytes } from "../../lib/utils";
import type { components } from "../../gen/api";

type MagnetFile = components["schemas"]["model.MagnetFile"];

interface FileTreeProps {
  files: MagnetFile[];
  selectedKeys: Set<string>;
  onSelectionChange: (keys: Set<string>) => void;
}

export function FileTree({
  files,
  selectedKeys,
  onSelectionChange,
}: FileTreeProps) {
  const [expandedKeys, setExpandedKeys] = useState<Set<Key>>(() => {
    const keys = new Set<Key>();
    files.forEach((f) => {
      if (f.isFolder && f.id) keys.add(f.id);
    });
    return keys;
  });

  const { nodeMap, parentMap } = useMemo(() => {
    const nMap = new Map<string, MagnetFile>();
    const pMap = new Map<string, string>();

    const traverse = (nodes: MagnetFile[], parentId?: string) => {
      nodes.forEach((node) => {
        if (node.id) {
            nMap.set(node.id, node);
            if (parentId) pMap.set(node.id, parentId);
        }
        if (node.children) traverse(node.children as MagnetFile[], node.id);
      });
    };
    traverse(files);
    return { nodeMap: nMap, parentMap: pMap };
  }, [files]);

  const handleSelectionChange = (keys: Selection) => {
    if (keys === "all") return;
    const next = new Set(Array.from(keys).map(String));
    const current = selectedKeys;

    const added = [...next].filter((k) => !current.has(k));
    const removed = [...current].filter((k) => !next.has(k));

    const finalSelection = new Set(next);

    const getDescendants = (id: string): string[] => {
      const ids: string[] = [];
      const node = nodeMap.get(id);
      if (node && node.children) {
        node.children.forEach((child) => {
          if (child.id) {
            ids.push(child.id);
            ids.push(...getDescendants(child.id));
          }
        });
      }
      return ids;
    };

    // Handle added items
    added.forEach((id) => {
      // Select descendants
      getDescendants(id).forEach((childId) => finalSelection.add(childId));

      // Check parents
      let curr = id;
      while (true) {
        const parentId = parentMap.get(curr);
        if (!parentId) break;

        const parentNode = nodeMap.get(parentId);
        if (!parentNode || !parentNode.children) break;

        const allSiblingsSelected = (parentNode.children as MagnetFile[]).every((child) =>
          child.id && finalSelection.has(child.id),
        );
        if (allSiblingsSelected) {
          finalSelection.add(parentId);
          curr = parentId;
        } else {
          break;
        }
      }
    });

    // Handle removed items
    removed.forEach((id) => {
      // Deselect descendants
      getDescendants(id).forEach((childId) => finalSelection.delete(childId));

      // Deselect parents
      let curr = id;
      while (true) {
        const parentId = parentMap.get(curr);
        if (!parentId) break;
        finalSelection.delete(parentId);
        curr = parentId;
      }
    });

    onSelectionChange(finalSelection);
  };

  const renderItem = (file: MagnetFile, level: number = 0) => {
    return (
      <TreeItem
        key={file.id}
        id={file.id}
        textValue={file.name || "unknown"}
        className="outline-none scroll-m-2"
      >
        <TreeItemContent>
          {({ isExpanded }) => (
            <div
              className="flex items-center gap-2 py-2 px-3 rounded-xl hover:bg-default/10 cursor-pointer transition-colors group"
              style={{ paddingLeft: `${level * 20 + 12}px` }}
            >
              {/* Expand/Collapse button for folders */}
              {file.isFolder ? (
                <Button
                  slot="chevron"
                  className="p-1 rounded hover:bg-default/20 outline-none border-none bg-transparent cursor-pointer"
                >
                  {isExpanded ? (
                    <IconChevronDown className="w-4 h-4 text-muted" />
                  ) : (
                    <IconChevronRight className="w-4 h-4 text-muted" />
                  )}
                </Button>
              ) : (
                <span className="w-8" />
              )}

              {/* Selection Checkbox */}
              <Checkbox slot="selection">
                {({ isSelected, isIndeterminate }) => (
                  <div
                    className={`w-4 h-4 rounded border flex items-center justify-center transition-colors ${
                      isSelected || isIndeterminate
                        ? "bg-accent border-accent text-accent-foreground"
                        : "border-default-400 group-hover:border-default-500"
                    }`}
                  >
                    {isSelected && (
                      <svg
                        viewBox="0 0 24 24"
                        className="w-3 h-3 fill-none stroke-current stroke-[4]"
                        aria-hidden="true"
                      >
                        <title>Selected</title>
                        <polyline points="20 6 9 17 4 12" />
                      </svg>
                    )}
                    {isIndeterminate && (
                      <div className="w-2 h-0.5 bg-current rounded-full" />
                    )}
                  </div>
                )}
              </Checkbox>

              {/* Icon */}
              {file.isFolder ? (
                <IconFolder className="w-5 h-5 text-warning" />
              ) : (
                <IconFile className="w-5 h-5 text-muted" />
              )}

              {/* Name */}
              <span className="flex-1 text-sm font-medium truncate">
                {file.name}
              </span>

              {/* Size */}
              <span className="text-xs text-muted font-mono">
                {formatBytes(file.size || 0)}
              </span>
            </div>
          )}
        </TreeItemContent>

        {/* Children */}
        {file.isFolder && file.children && (
          <Collection items={file.children as MagnetFile[]}>
            {(child) => renderItem(child, level + 1)}
          </Collection>
        )}
      </TreeItem>
    );
  };

  return (
    <Tree
      aria-label="Torrent files"
      className="w-full outline-none p-4 pb-12"
      selectionMode="multiple"
      selectedKeys={selectedKeys}
      onSelectionChange={handleSelectionChange}
      expandedKeys={expandedKeys}
      onExpandedChange={setExpandedKeys}
    >
      <Collection items={files}>{(file) => renderItem(file, 0)}</Collection>
    </Tree>
  );
}

// Helper to get all file IDs (non-folder) recursively
export function getAllFileIds(files: MagnetFile[]): string[] {
  const ids: string[] = [];
  for (const file of files) {
    if (!file.isFolder && file.id) {
      ids.push(file.id);
    }
    if (file.children) {
      ids.push(...getAllFileIds(file.children as MagnetFile[]));
    }
  }
  return ids;
}

// Helper to calculate total size of selected files
export function getSelectedSize(
  files: MagnetFile[],
  selectedKeys: Set<string>,
): number {
  let total = 0;
  for (const file of files) {
    if (!file.isFolder && file.id && selectedKeys.has(file.id)) {
      total += (file.size || 0);
    }
    if (file.children) {
      total += getSelectedSize(file.children as MagnetFile[], selectedKeys);
    }
  }
  return total;
}