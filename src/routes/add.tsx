import {
	Accordion,
	Button,
	FieldError,
	Input,
	Label,
	Tabs,
	TextArea,
	TextField,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import React, { useId } from "react";
import { FileTrigger } from "react-aria-components";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconFileArrowUp from "~icons/gravity-ui/file-arrow-up";
import IconLink from "~icons/gravity-ui/link";
import { useAria2Actions } from "../hooks/useAria2";
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

	const { addUri, addTorrent, addMetalink } = useAria2Actions();

	const handleFileSelect = React.useCallback(
		async (files: FileList | null) => {
			if (!files) return;
			const fileList = Array.from(files);

			for (const file of fileList) {
				const reader = new FileReader();
				reader.onload = () => {
					const base64 = (reader.result as string).split(",")[1];
					const onSuccess = () => navigate({ to: "/tasks/active" });

					if (file.name.endsWith(".torrent")) {
						addTorrent.mutate(
							{
								torrent: base64,
								options: options as Record<string, string>,
							},
							{ onSuccess },
						);
					} else if (file.name.endsWith(".metalink")) {
						addMetalink.mutate(
							{
								metalink: base64,
								options: options as Record<string, string>,
							},
							{ onSuccess },
						);
					}
				};
				reader.readAsDataURL(file);
			}
		},
		[addTorrent, addMetalink, options, navigate],
	);

	// Handle file from global drop
	React.useEffect(() => {
		if (pendingFile) {
			handleFileSelect([pendingFile] as any);
			clearPendingFile();
		}
	}, [pendingFile, clearPendingFile, handleFileSelect]);

	const validateUris = (val: string) => {
		if (!val.trim()) return "Enter at least one link";
		const lines = val.split("\n").filter((l) => l.trim());
		const invalid = lines.find(
			(l) =>
				!/^(http|https|ftp|sftp|magnet):/i.test(l.trim()) &&
				!/^[a-f0-9]{40}$/i.test(l.trim()),
		);
		if (invalid) return "Invalid protocol in one of the links";
		return true;
	};

	const handleSubmit = async () => {
		const onSuccess = () => navigate({ to: "/tasks/active" });

		if (selectedTab === `${baseId}-links` && validateUris(uris) === true) {
			const uriList = uris.split("\n").filter((u) => u.trim());
			addUri.mutate(
				{
					uris: uriList,
					options: options as Record<string, string>,
				},
				{ onSuccess },
			);
		}
	};

	return (
		<div className="max-w-2xl mx-auto space-y-8 pb-12">
			<div className="flex items-center gap-4">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/tasks/all" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<h2 className="text-2xl font-bold tracking-tight">Add Download</h2>
			</div>

			<div className="bg-muted-background/40 p-8 rounded-[40px] border border-border space-y-8 shadow-sm">
				<Tabs
					aria-label="Download Type"
					selectedKey={selectedTab as string}
					onSelectionChange={setSelectedTab}
					className="w-full"
				>
					<Tabs.ListContainer className="bg-default/10 p-1.5 rounded-2xl">
						<Tabs.List className="w-full">
							<Tabs.Tab id={`${baseId}-links`} className="w-full py-2.5">
								<div className="flex items-center justify-center gap-2">
									<IconLink className="w-4 h-4" />
									<span className="font-bold">Links</span>
								</div>
								<Tabs.Indicator className="bg-background rounded-xl shadow-sm" />
							</Tabs.Tab>
							<Tabs.Tab id={`${baseId}-torrent`} className="w-full py-2.5">
								<div className="flex items-center justify-center gap-2">
									<IconFileArrowUp className="w-4 h-4" />
									<span className="font-bold">File</span>
								</div>
								<Tabs.Indicator className="bg-background rounded-xl shadow-sm" />
							</Tabs.Tab>
						</Tabs.List>
					</Tabs.ListContainer>
				</Tabs>

				<div className="min-h-[200px]">
					{selectedTab === `${baseId}-links` && (
						<TextField
							className="w-full"
							value={uris}
							onChange={setUris}
							validate={validateUris}
							validationBehavior="aria"
						>
							<div className="flex flex-col gap-3">
								<Label className="text-sm font-bold tracking-tight px-1">
									Download Links
								</Label>
								<div className="relative group">
									<TextArea
										placeholder="https://example.com/file.zip&#10;magnet:?xt=urn:btih:..."
										className="w-full p-4 bg-default/10 rounded-2xl text-sm border border-transparent focus:bg-default/20 focus:border-accent/30 transition-all outline-none min-h-[160px] leading-relaxed data-[invalid=true]:border-danger/50"
									/>
									<FieldError className="absolute -bottom-6 right-1 text-[10px] text-danger font-black uppercase tracking-widest animate-in fade-in slide-in-from-top-1" />
								</div>
								<p className="text-[10px] text-muted uppercase font-black tracking-widest px-1">
									Enter one URL per line. Supports HTTP, FTP, SFTP and Magnet.
								</p>
							</div>
						</TextField>
					)}

					{selectedTab === `${baseId}-torrent` && (
						<div className="flex flex-col gap-3 relative group">
							<Label className="text-sm font-bold tracking-tight px-1">
								Torrent or Metalink File
							</Label>
							<FileTrigger
								acceptedFileTypes={[".torrent", ".metalink"]}
								allowsMultiple
								onSelect={handleFileSelect}
							>
								<Button
									variant="secondary"
									className="flex flex-col items-center justify-center p-12 border-2 border-dashed border-border rounded-[32px] text-muted gap-4 hover:border-accent hover:text-accent transition-all cursor-pointer w-full h-auto bg-default/5 hover:bg-accent/5 overflow-hidden"
								>
									<div className="w-16 h-16 bg-background rounded-3xl flex items-center justify-center shadow-sm border border-border group-hover:scale-110 transition-transform duration-500">
										<IconFileArrowUp className="w-8 h-8 opacity-30 text-accent group-hover:opacity-100" />
									</div>
									<div className="flex flex-col gap-1 items-center">
										<p className="text-center w-full font-bold">
											Click to browse files
										</p>
										<p className="text-[10px] uppercase font-black tracking-widest opacity-60">
											Supports .torrent and .metalink
										</p>
									</div>
								</Button>
							</FileTrigger>
						</div>
					)}
				</div>

				<Accordion className="px-0">
					<Accordion.Item
						id={`${baseId}-advanced-item`}
						className="border-none bg-default/5 rounded-3xl overflow-hidden px-2 py-1"
					>
						<Accordion.Heading>
							<Accordion.Trigger className="px-4 py-3 hover:bg-default/10 rounded-2xl transition-all outline-none group">
								<div className="flex items-center gap-3">
									<div className="w-8 h-8 bg-background rounded-xl flex items-center justify-center shadow-sm border border-border group-data-[expanded=true]:text-accent group-data-[expanded=true]:border-accent/30">
										<IconChevronDown className="w-4 h-4 group-data-[expanded=true]:rotate-180 transition-transform duration-300" />
									</div>
									<span className="font-bold text-sm tracking-tight">
										Advanced Download Options
									</span>
								</div>
							</Accordion.Trigger>
						</Accordion.Heading>
						<Accordion.Panel>
							<Accordion.Body className="px-6 pb-6 pt-4">
								<TextField
									value={options.dir || ""}
									onChange={(val) => setOptions({ ...options, dir: val })}
								>
									<div className="flex flex-col gap-2">
										<Label className="text-xs font-bold tracking-tight uppercase text-muted">
											Override Download Directory
										</Label>
										<Input
											placeholder="/path/to/downloads"
											className="w-full h-11 px-4 bg-background rounded-2xl text-sm border border-border focus:border-accent/30 transition-all outline-none"
										/>
										<p className="text-[10px] text-muted leading-relaxed">
											Leave empty to use the default directory configured in
											aria2.
										</p>
									</div>
								</TextField>
							</Accordion.Body>
						</Accordion.Panel>
					</Accordion.Item>
				</Accordion>

				<div className="flex justify-end gap-3 pt-4">
					<Button
						variant="ghost"
						className="px-6 h-12 rounded-2xl font-bold"
						onPress={() => navigate({ to: "/tasks/all" })}
					>
						Cancel
					</Button>
					<Button
						className="px-8 h-12 rounded-2xl font-bold shadow-lg shadow-accent/20 bg-accent text-accent-foreground"
						onPress={handleSubmit}
						isDisabled={
							selectedTab === `${baseId}-links`
								? validateUris(uris) !== true
								: false
						}
					>
						Start Download
					</Button>
				</div>
			</div>
		</div>
	);
}
