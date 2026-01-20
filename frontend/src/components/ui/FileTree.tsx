import {
  Tree,
  TreeItem,
  TreeItemContent,
  Collection,
  Button,
  Checkbox,
} from "react-aria-components";
import { useState } from "react";
import IconChevronRight from "~icons/gravity-ui/chevron-right";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconFolder from "~icons/gravity-ui/folder";
import IconFile from "~icons/gravity-ui/file";
import { formatBytes } from "../../lib/utils";
import type { MagnetFile } from "../../lib/types";

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
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => {
    const keys = new Set<string>();
    files.forEach((f) => {
      if (f.isFolder) keys.add(f.id);
    });
    return keys;
  });

  const renderItem = (file: MagnetFile, level: number = 0) => {
    const isExpanded = expandedKeys.has(file.id);

    return (
      <TreeItem
        key={file.id}
        id={file.id}
        textValue={file.name}
        className="outline-none"
      >
        <TreeItemContent>
          <div
            className="flex items-center gap-2 py-2 px-3 rounded-xl hover:bg-default/10 cursor-pointer transition-colors group"
            style={{ paddingLeft: `${level * 20 + 12}px` }}
          >
            {/* Expand/Collapse button for folders */}
            {file.isFolder ? (
              <Button
                className="p-1 rounded hover:bg-default/20 outline-none border-none bg-transparent cursor-pointer"
                onPress={() => {
                  const newExpanded = new Set(expandedKeys);
                  if (isExpanded) {
                    newExpanded.delete(file.id);
                  } else {
                    newExpanded.add(file.id);
                  }
                  setExpandedKeys(newExpanded);
                }}
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
                    <svg viewBox="0 0 24 24" className="w-3 h-3 fill-none stroke-current stroke-[4]" aria-hidden="true">
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
              {formatBytes(file.size)}
            </span>
          </div>
        </TreeItemContent>

        {/* Children */}
        {file.isFolder && file.children && isExpanded && (
          <Collection items={file.children}>
            {(child) => renderItem(child, level + 1)}
          </Collection>
        )}
      </TreeItem>
    );
  };

  return (
    <Tree 
      aria-label="Torrent files" 
      className="w-full outline-none"
      selectionMode="multiple"
      selectedKeys={selectedKeys}
      onSelectionChange={(keys) => onSelectionChange(keys as Set<string>)}
    >
      <Collection items={files}>{(file) => renderItem(file, 0)}</Collection>
    </Tree>
  );
}

// Helper to get all file IDs (non-folder) recursively
export function getAllFileIds(files: MagnetFile[]): string[] {
  const ids: string[] = [];
  for (const file of files) {
    if (!file.isFolder) {
      ids.push(file.id);
    }
    if (file.children) {
      ids.push(...getAllFileIds(file.children));
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
    if (!file.isFolder && selectedKeys.has(file.id)) {
      total += file.size;
    }
    if (file.children) {
      total += getSelectedSize(file.children, selectedKeys);
    }
  }
  return total;
}
