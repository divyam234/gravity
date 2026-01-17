import {
	Accordion,
	Button,
	Description,
	Input,
	Label,
	Tabs,
	TextArea,
	TextField,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import React, { useId } from "react";
import { FileTrigger } from "react-aria-components";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconChevronRight from "~icons/gravity-ui/chevron-right";
import IconFileArrowUp from "~icons/gravity-ui/file-arrow-up";
import IconLink from "~icons/gravity-ui/link";
import IconXmark from "~icons/gravity-ui/xmark";
import { useAria2Actions } from "../hooks/useAria2";
import { cn } from "../lib/utils";
import { useFileStore } from "../store/useFileStore";

export const Route = createFileRoute("/add")({
	component: AddDownloadPage,
});

function AddDownloadPage() {
	const navigate = useNavigate();
	const baseId = useId();
	const { pendingFile, clearPendingFile } = useFileStore();
	const [selectedTab, setSelectedTab] = React.useState<React.Key>(
		`${baseId}-links`,
	);
	const [uris, setUris] = React.useState("");
	const [options, setOptions] = React.useState<{ dir?: string }>({});
	const [selectedFile, setSelectedFile] = React.useState<File | null>(null);

	// Handle file from global drop
	React.useEffect(() => {
		if (pendingFile) {
			setSelectedFile(pendingFile);
			setSelectedTab(`${baseId}-torrent`);
			clearPendingFile();
		}
	}, [pendingFile, clearPendingFile, baseId]);

	const { addUri, addTorrent, addMetalink } = useAria2Actions();

	const handleSelect = (e: FileList | null) => {
		const file = e?.[0];
		if (file) {
			setSelectedFile(file);
		}
	};

	const handleSubmit = async () => {
		if (selectedTab === `${baseId}-links` && uris.trim()) {
			const uriList = uris.split("\n").filter((u) => u.trim());
			addUri.mutate({
				uris: uriList,
				options: options as Record<string, string>,
			});
			setUris("");
			navigate({ to: "/" });
		} else if (selectedTab === `${baseId}-torrent` && selectedFile) {
			const reader = new FileReader();
			reader.onload = () => {
				const base64 = (reader.result as string).split(",")[1];
				if (selectedFile.name.endsWith(".torrent")) {
					addTorrent.mutate({
						torrent: base64,
						options: options as Record<string, string>,
					});
				} else if (selectedFile.name.endsWith(".metalink")) {
					addMetalink.mutate({
						metalink: base64,
						options: options as Record<string, string>,
					});
				}
				setSelectedFile(null);
				navigate({ to: "/" });
			};
			reader.readAsDataURL(selectedFile);
		}
	};

	return (
		<div className="max-w-2xl mx-auto space-y-6">
			<div className="flex items-center gap-4">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<h2 className="text-2xl font-bold tracking-tight">Add Download</h2>
			</div>

			<div className="bg-default-50/50 p-6 rounded-3xl border border-default-100 space-y-6">
				<Tabs
					aria-label="Download Type"
					selectedKey={selectedTab as string}
					onSelectionChange={setSelectedTab}
				>
					<Tabs.ListContainer>
						<Tabs.List>
							<Tabs.Tab id={`${baseId}-links`}>
								<div className="flex items-center gap-2">
									<IconLink className="w-4 h-4" />
									<span>Links</span>
								</div>
								<Tabs.Indicator />
							</Tabs.Tab>
							<Tabs.Tab id={`${baseId}-torrent`}>
								<div className="flex items-center gap-2">
									<IconFileArrowUp className="w-4 h-4" />
									<span>Torrent File</span>
								</div>
								<Tabs.Indicator />
							</Tabs.Tab>
						</Tabs.List>
					</Tabs.ListContainer>
				</Tabs>

				{selectedTab === `${baseId}-links` && (
					<div className="flex flex-col gap-6">
						<TextField className="w-full">
							<Label className="text-small font-medium mb-2 block">
								Download Links
							</Label>
							<TextArea
								placeholder="https://example.com/file.zip"
								rows={5}
								value={uris}
								onChange={(e) => setUris(e.target.value)}
								fullWidth
							/>
							<Description className="text-xs text-default-500 mt-1">
								Enter one URL per line.
							</Description>
						</TextField>
					</div>
				)}

				{selectedTab === `${baseId}-torrent` && (
					<div className="flex flex-col gap-6 relative group">
						<FileTrigger
							acceptedFileTypes={[".torrent", ".metalink"]}
							onSelect={handleSelect}
						>
							<Button
								variant="tertiary"
								className="flex flex-col items-center justify-center p-12 border-2 border-dashed border-default-200 rounded-3xl text-default-400 gap-4 hover:border-primary transition-all cursor-pointer w-full h-auto bg-transparent hover:bg-primary/5"
							>
								<IconFileArrowUp
									className={cn(
										"w-12 h-12 transition-transform duration-300 group-hover:-translate-y-1",
										selectedFile ? "text-primary opacity-100" : "opacity-20",
									)}
								/>
								<p className="text-center w-full font-medium">
									{selectedFile
										? selectedFile.name
										: "Click to browse or drag and drop files here"}
								</p>
							</Button>
						</FileTrigger>

						{selectedFile && (
							<Button
								isIconOnly
								size="sm"
								variant="secondary"
								className="absolute top-2 right-2 rounded-full shadow-sm"
								onPress={() => setSelectedFile(null)}
							>
								<IconXmark className="w-4 h-4" />
							</Button>
						)}
					</div>
				)}

				<Accordion>
					<Accordion.Item id={`${baseId}-advanced-item`}>
						<Accordion.Heading>
							<Accordion.Trigger className="flex items-center gap-2 text-primary hover:underline py-2 outline-none">
								<IconChevronRight className="w-4 h-4 group-data-[expanded=true]:rotate-90 transition-transform" />
								<span className="font-semibold text-small">
									Advanced Options
								</span>
							</Accordion.Trigger>
						</Accordion.Heading>
						<Accordion.Panel>
							<Accordion.Body className="pt-4 pb-2 space-y-4">
								<TextField>
									<Label className="text-small font-medium mb-1 block">
										Download Directory
									</Label>
									<Input
										placeholder="/home/user/downloads"
										value={options.dir || ""}
										onChange={(e) =>
											setOptions({ ...options, dir: e.target.value })
										}
									/>
								</TextField>
							</Accordion.Body>
						</Accordion.Panel>
					</Accordion.Item>
				</Accordion>

				<div className="flex justify-end gap-3 pt-6 border-t border-default-100">
					<Button variant="ghost" onPress={() => navigate({ to: "/" })}>
						Cancel
					</Button>
					<Button
						onPress={handleSubmit}
						isDisabled={
							selectedTab === `${baseId}-links` ? !uris.trim() : !selectedFile
						}
					>
						Download Now
					</Button>
				</div>
			</div>
		</div>
	);
}
