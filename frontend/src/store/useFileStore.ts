import { create } from "zustand";

interface FileState {
	pendingFile: File | null;
	setPendingFile: (file: File | null) => void;
	clearPendingFile: () => void;
}

export const useFileStore = create<FileState>((set) => ({
	pendingFile: null,
	setPendingFile: (pendingFile) => set({ pendingFile }),
	clearPendingFile: () => set({ pendingFile: null }),
}));
