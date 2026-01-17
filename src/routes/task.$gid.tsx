import {
	Button,
	Card,
	Checkbox,
	Chip,
	ScrollShadow,
	Spinner,
	Tabs,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import React, { useId } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconCircleInfo from "~icons/gravity-ui/circle-info";
import IconFile from "~icons/gravity-ui/file";
import IconNodesDown from "~icons/gravity-ui/nodes-down";
import IconPersons from "~icons/gravity-ui/persons";
import {
	useTaskFiles,
	useTaskPeers,
	useTaskServers,
	useTaskStatus,
} from "../hooks/useAria2";
import type { Aria2File } from "../lib/aria2-rpc";
import { aria2 } from "../lib/aria2-rpc";
import { formatBytes } from "../lib/utils";

export const Route = createFileRoute("/task/$gid")({
	component: TaskDetailsPage,
});

function TaskDetailsPage() {
	const { gid } = Route.useParams();
	const navigate = useNavigate();
	const baseId = useId();
	const { data: task, isLoading: isStatusLoading } = useTaskStatus(gid);
	const { data: files } = useTaskFiles(gid);
	const [selectedTab, setSelectedTab] = React.useState<React.Key>(
		`${baseId}-overview`,
	);

	const { data: peers } = useTaskPeers(gid, selectedTab === `${baseId}-peers`);
	const { data: servers } = useTaskServers(
		gid,
		selectedTab === `${baseId}-servers`,
	);

	const [selectedIndices, setSelectedIndices] = React.useState<Set<string>>(
		new Set(),
	);

	React.useEffect(() => {
		if (files) {
			const selected = new Set(
				files
					.filter((f: Aria2File) => f.selected === "true")
					.map((f: Aria2File) => f.index),
			);
			setSelectedIndices(selected);
		}
	}, [files]);

	const handleToggleFile = (index: string, isSelected: boolean) => {
		const newSelected = new Set(selectedIndices);
		if (isSelected) newSelected.add(index);
		else newSelected.delete(index);

		setSelectedIndices(newSelected);

		// Apply to aria2
		const selectFileStr = Array.from(newSelected).sort().join(",");
		aria2.changeOption(gid, { "select-file": selectFileStr });
	};

	if (isStatusLoading) {
		return (
			<div className="flex items-center justify-center min-h-[400px]">
				<Spinner size="lg" />
			</div>
		);
	}

	return (
		<div className="max-w-6xl mx-auto space-y-6">
			<div className="flex items-center gap-4">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<h2 className="text-2xl font-bold">Task Details</h2>
				<code className="text-xs bg-default-100 px-2 py-1 rounded text-default-500">
					{gid}
				</code>
			</div>

			<div className="flex flex-col md:flex-row gap-6">
				{/* Sidebar / Tabs */}
				<div className="w-full md:w-64">
					<Card className="p-2">
						<Tabs
							aria-label="Task Details Sections"
							orientation="vertical"
							selectedKey={selectedTab as string}
							onSelectionChange={setSelectedTab}
							className="w-full"
						>
							<Tabs.ListContainer>
								<Tabs.List className="w-full">
									<Tabs.Tab id={`${baseId}-overview`}>
										<div className="flex items-center gap-2">
											<IconCircleInfo className="w-4 h-4" />
											<span>Overview</span>
										</div>
										<Tabs.Indicator />
									</Tabs.Tab>
									<Tabs.Tab id={`${baseId}-files`}>
										<div className="flex items-center gap-2">
											<IconFile className="w-4 h-4" />
											<span>Files</span>
										</div>
										<Tabs.Indicator />
									</Tabs.Tab>
									<Tabs.Tab id={`${baseId}-peers`}>
										<div className="flex items-center gap-2">
											<IconPersons className="w-4 h-4" />
											<span>Peers ({peers?.length || 0})</span>
										</div>
										<Tabs.Indicator />
									</Tabs.Tab>
									<Tabs.Tab id={`${baseId}-servers`}>
										<div className="flex items-center gap-2">
											<IconNodesDown className="w-4 h-4" />
											<span>Servers</span>
										</div>
										<Tabs.Indicator />
									</Tabs.Tab>
								</Tabs.List>
							</Tabs.ListContainer>
						</Tabs>
					</Card>
				</div>

				{/* Content Area */}
				<div className="flex-1 min-h-[500px]">
					<Card className="h-full overflow-hidden flex flex-col bg-default-50/30">
						{selectedTab === `${baseId}-overview` && task && (
							<ScrollShadow className="flex-1 p-6">
								<div className="space-y-8">
									<section>
										<h3 className="text-lg font-bold mb-4">Identity</h3>
										<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
											<div className="space-y-1">
												<p className="text-tiny text-default-500 uppercase font-bold">
													Filename
												</p>
												<p className="text-small break-all">
													{task.bittorrent?.info?.name ||
														task.files[0]?.path?.split("/").pop() ||
														gid}
												</p>
											</div>
											<div className="space-y-1">
												<p className="text-tiny text-default-500 uppercase font-bold">
													GID
												</p>
												<p className="font-mono text-small">{gid}</p>
											</div>
											{task.infoHash && (
												<div className="space-y-1 md:col-span-2">
													<p className="text-tiny text-default-500 uppercase font-bold">
														Info Hash
													</p>
													<p className="font-mono text-small break-all">
														{task.infoHash}
													</p>
												</div>
											)}
										</div>
									</section>

									<section>
										<h3 className="text-lg font-bold mb-4">Location</h3>
										<div className="grid grid-cols-1 gap-4">
											<div className="space-y-1">
												<p className="text-tiny text-default-500 uppercase font-bold">
													Download Directory
												</p>
												<p className="text-small break-all">{task.dir}</p>
											</div>
										</div>
									</section>

									<section>
										<h3 className="text-lg font-bold mb-4">Status</h3>
										<div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-center">
											<div className="p-4 bg-background rounded-2xl border border-default-100">
												<p className="text-tiny text-default-500 uppercase font-bold mb-1">
													State
												</p>
												<Chip
													size="sm"
													variant="soft"
													color="success"
													className="uppercase font-bold"
												>
													{task.status}
												</Chip>
											</div>
											<div className="p-4 bg-background rounded-2xl border border-default-100">
												<p className="text-tiny text-default-500 uppercase font-bold mb-1">
													Connections
												</p>
												<p className="text-xl font-bold">{task.connections}</p>
											</div>
											<div className="p-4 bg-background rounded-2xl border border-default-100">
												<p className="text-tiny text-default-500 uppercase font-bold mb-1">
													Total Size
												</p>
												<p className="text-medium font-bold">
													{formatBytes(task.totalLength)}
												</p>
											</div>
											<div className="p-4 bg-background rounded-2xl border border-default-100">
												<p className="text-tiny text-default-500 uppercase font-bold mb-1">
													Uploaded
												</p>
												<p className="text-medium font-bold text-primary">
													{formatBytes(task.uploadLength)}
												</p>
											</div>
										</div>
									</section>
								</div>
							</ScrollShadow>
						)}

						{selectedTab === `${baseId}-files` && (
							<div className="flex flex-col h-full">
								<div className="p-4 border-b border-default-100 flex justify-between items-center bg-default-50">
									<span className="text-small font-medium text-default-500">
										{files?.length} Files found
									</span>
									<Chip size="sm" variant="soft" color="accent">
										Selective Download Enabled
									</Chip>
								</div>
								<ScrollShadow className="flex-1 p-4">
									<div className="space-y-3">
										{files?.map((file: Aria2File) => (
											<div
												key={file.index}
												className="flex items-start gap-4 p-4 rounded-2xl border border-default-100 bg-background hover:border-primary/50 hover:shadow-sm transition-all group"
											>
												<Checkbox
													isSelected={selectedIndices.has(file.index)}
													onChange={(isSelected) =>
														handleToggleFile(file.index, isSelected)
													}
												>
													<Checkbox.Control>
														<Checkbox.Indicator />
													</Checkbox.Control>
												</Checkbox>

												<div className="flex-1 min-w-0">
													<div className="flex items-center gap-2">
														<IconFile className="w-4.5 h-4.5 text-default-400 group-hover:text-primary transition-colors shrink-0" />
														<span
															className="text-medium font-semibold truncate"
															title={file.path}
														>
															{file.path.split("/").pop() || "Unknown File"}
														</span>
													</div>
													<div className="flex gap-4 mt-2 text-small text-default-500">
														<span>{formatBytes(file.length)}</span>
														<span className="text-success font-medium">
															{(
																(Number(file.completedLength) /
																	Number(file.length)) *
																100
															).toFixed(1)}
															%
														</span>
													</div>
												</div>
											</div>
										))}
									</div>
								</ScrollShadow>
							</div>
						)}

						{selectedTab === `${baseId}-peers` && (
							<ScrollShadow className="flex-1 p-4">
								{!peers || peers.length === 0 ? (
									<div className="flex flex-col items-center justify-center h-full text-default-400 gap-4">
										<IconPersons className="w-12 h-12 opacity-20" />
										<p>No peers connected</p>
									</div>
								) : (
									<div className="space-y-3">
										{peers.map((peer: any) => (
											<div
												key={`${peer.ip}-${peer.port}`}
												className="flex items-center justify-between p-4 rounded-2xl border border-default-100 bg-background"
											>
												<div className="flex flex-col">
													<span className="text-small font-bold">
														{peer.ip}:{peer.port}
													</span>
													<span className="text-tiny text-default-500">
														{peer.peerId}
													</span>
												</div>
												<div className="flex gap-6 text-small">
													<div className="flex flex-col items-end">
														<span className="text-tiny text-default-400 uppercase font-bold">
															Down
														</span>
														<span className="text-success font-bold">
															{formatBytes(peer.downloadSpeed)}/s
														</span>
													</div>
													<div className="flex flex-col items-end">
														<span className="text-tiny text-default-400 uppercase font-bold">
															Up
														</span>
														<span className="text-primary font-bold">
															{formatBytes(peer.uploadSpeed)}/s
														</span>
													</div>
												</div>
											</div>
										))}
									</div>
								)}
							</ScrollShadow>
						)}

						{selectedTab === `${baseId}-servers` && (
							<ScrollShadow className="flex-1 p-4">
								{!servers || servers.length === 0 ? (
									<div className="flex flex-col items-center justify-center h-full text-default-400 gap-4">
										<IconNodesDown className="w-12 h-12 opacity-20" />
										<p>No server information available</p>
									</div>
								) : (
									<div className="space-y-4">
										{servers.map((srv: any) => (
											<div key={srv.index} className="space-y-3">
												<div className="text-tiny font-bold text-default-400 uppercase px-2">
													File Index: {srv.index}
												</div>
												{srv.servers.map((s: any) => (
													<div
														key={s.uri}
														className="p-4 rounded-2xl border border-default-100 bg-background flex justify-between items-center"
													>
														<div className="flex flex-col min-w-0">
															<span className="text-small font-bold truncate">
																{s.uri}
															</span>
															<span className="text-tiny text-default-500">
																Status: {s.currentPriority} | Current:{" "}
																{s.uri === srv.currentUri ? "Yes" : "No"}
															</span>
														</div>
														<div className="flex flex-col items-end shrink-0">
															<span className="text-tiny text-default-400 uppercase font-bold">
																Speed
															</span>
															<span className="text-success font-bold">
																{formatBytes(s.downloadSpeed)}/s
															</span>
														</div>
													</div>
												))}
											</div>
										))}
									</div>
								)}
							</ScrollShadow>
						)}
					</Card>
				</div>
			</div>
		</div>
	);
}
