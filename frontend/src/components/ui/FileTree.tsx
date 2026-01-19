import { Checkbox } from "@heroui/react";
import {
  Tree,
  TreeItem,
  TreeItemContent,
  Collection,
  Button,
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
    // Auto-expand first level
    const keys = new Set<string>();
    files.forEach((f) => {
      if (f.isFolder) keys.add(f.id);
    });
    return keys;
  });

  const toggleSelection = (id: string, file: MagnetFile) => {
    const newSelection = new Set(selectedKeys);

    if (newSelection.has(id)) {
      // Deselect this and all children
      newSelection.delete(id);
      if (file.children) {
        getAllFileIds(file.children).forEach((childId) =>
          newSelection.delete(childId),
        );
      }
    } else {
      // Select this and all children
      newSelection.add(id);
      if (file.children) {
        getAllFileIds(file.children).forEach((childId) =>
          newSelection.add(childId),
        );
      }
    }

    onSelectionChange(newSelection);
  };

  const renderItem = (file: MagnetFile, level: number = 0) => {
    const isExpanded = expandedKeys.has(file.id);
    const isSelected = selectedKeys.has(file.id);

    // Check if all children are selected (for folder partial state)
    const childIds = file.children ? getAllFileIds(file.children) : [];
    const allChildrenSelected =
      childIds.length > 0 && childIds.every((id) => selectedKeys.has(id));
    const someChildrenSelected = childIds.some((id) => selectedKeys.has(id));

    return (
      <TreeItem
        key={file.id}
        id={file.id}
        textValue={file.name}
        className="outline-none"
      >
        <TreeItemContent>
          <div
            className="flex items-center gap-2 py-2 px-3 rounded-xl hover:bg-default/10 cursor-pointer transition-colors"
            style={{ paddingLeft: `${level * 20 + 12}px` }}
          >
            {/* Expand/Collapse button for folders */}
            {file.isFolder ? (
              <Button
                className="p-1 rounded hover:bg-default/20 outline-none"
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
              <span className="w-6" /> // Spacer for alignment
            )}

            {/* Checkbox */}
            <Checkbox
              isSelected={file.isFolder ? allChildrenSelected : isSelected}
              isIndeterminate={
                file.isFolder && someChildrenSelected && !allChildrenSelected
              }
              onChange={() => toggleSelection(file.id, file)}
              className="mr-2"
            />

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
    <Tree aria-label="Torrent files" className="w-full">
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
