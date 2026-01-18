import { useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { useAria2Actions } from "./useAria2";

export function useShortcuts() {
	const navigate = useNavigate();
	const { purgeDownloadResult } = useAria2Actions();

	useEffect(() => {
		const handleKeyDown = (e: KeyboardEvent) => {
			// Don't trigger if user is typing in an input/textarea
			if (
				document.activeElement?.tagName === "INPUT" ||
				document.activeElement?.tagName === "TEXTAREA"
			) {
				return;
			}

			if (e.shiftKey) {
				switch (e.key.toLowerCase()) {
					case "a":
						e.preventDefault();
						navigate({ to: "/add" });
						break;
					case "s":
						e.preventDefault();
						navigate({ to: "/settings" });
						break;
					case "c":
						e.preventDefault();
						purgeDownloadResult.mutate();
						break;
				}
			}

			if (e.key === "Escape") {
				navigate({ to: "/" });
			}
		};

		window.addEventListener("keydown", handleKeyDown);
		return () => window.removeEventListener("keydown", handleKeyDown);
	}, [navigate, purgeDownloadResult]);
}
