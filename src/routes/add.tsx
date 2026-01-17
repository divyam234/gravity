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
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconChevronRight from "~icons/gravity-ui/chevron-right";
import IconFileArrowUp from "~icons/gravity-ui/file-arrow-up";
import IconLink from "~icons/gravity-ui/link";
import { useAria2Actions } from "../hooks/useAria2";
import { cn } from "../lib/utils";

export const Route = createFileRoute("/add")({
	component: AddDownloadPage,
});

function AddDownloadPage() {
	const navigate = useNavigate();
	const baseId = useId();
	const [selectedTab, setSelectedTab] = React.useState<React.Key>(
		`${baseId}-links`,
	);
	const [uris, setUris] = React.useState("");
	const [options, setOptions] = React.useState<{ dir?: string }>({});
	const fileInputRef = React.useRef<HTMLInputElement>(null);
	const [selectedFile, setSelectedFile] = React.useState<File | null>(null);

	React.useEffect(() => {
		const handleFileDrop = (e: any) => {
			if (e.detail) {
				setSelectedFile(e.detail);
				setSelectedTab(`${baseId}-torrent`);
			}
		};
		window.addEventListener("aria2-file-drop", handleFileDrop);
		return () => window.removeEventListener("aria2-file-drop", handleFileDrop);
	}, [baseId]);

	const { addUri, addTorrent, addMetalink } = useAria2Actions();

	const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0];
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
				<h2 className="text-2xl font-bold">Add Download</h2>
			</div>

			<div className="bg-default-50/50 p-6 rounded-2xl border border-default-100 space-y-6">
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
					<div className="flex flex-col gap-6">
						<button
							type="button"
							className="flex flex-col items-center justify-center p-12 border-2 border-dashed border-default-200 rounded-2xl text-default-400 gap-4 hover:border-primary/50 transition-colors cursor-pointer w-full text-left"
							onClick={() => fileInputRef.current?.click()}
						>
							<input
								type="file"
								ref={fileInputRef}
								className="hidden"
								accept=".torrent,.metalink"
								onChange={handleFileChange}
							/>
							<IconFileArrowUp
								className={cn(
									"w-12 h-12",
									selectedFile ? "text-primary opacity-100" : "opacity-20",
								)}
							/>
							<p className="text-center w-full">
								{selectedFile
									? selectedFile.name
									: "Click to browse or drag and drop .torrent or .metalink files here"}
							</p>
							{selectedFile && (
								<Button
									size="sm"
									variant="tertiary"
									onClick={(e) => {
										e.stopPropagation();
										setSelectedFile(null);
									}}
								>
									Clear File
								</Button>
							)}
						</button>
					</div>
				)}

				<Accordion>
					<Accordion.Item id={`${baseId}-advanced-item`}>
						<Accordion.Heading>
							<Accordion.Trigger className="flex items-center gap-2 text-primary hover:underline py-2">
								<IconChevronRight className="w-4 h-4 group-data-[expanded=true]:rotate-90 transition-transform" />
								<span>Advanced Options</span>
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
