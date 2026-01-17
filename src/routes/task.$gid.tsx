import {
	Avatar,
	Button,
	Card,
	Chip,
	ListBox,
	ScrollShadow,
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
	taskFilesOptions,
	taskStatusOptions,
	useTaskFiles,
	useTaskPeers,
	useTaskServers,
	useTaskStatus,
} from "../hooks/useAria2";
import type { Aria2File } from "../lib/aria2-rpc";
import { aria2 } from "../lib/aria2-rpc";
import { formatBytes } from "../lib/utils";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/task/$gid")({
	component: TaskDetailsPage,
	loader: async ({ context: { queryClient }, params: { gid } }) => {
		const { rpcUrl } = useSettingsStore.getState();
		if (!rpcUrl) return;
		await Promise.all([
			queryClient.ensureQueryData(taskStatusOptions(rpcUrl, gid)),
			queryClient.ensureQueryData(taskFilesOptions(rpcUrl, gid)),
		]);
	},
});

function TaskDetailsPage() {
	const { gid } = Route.useParams();
	const navigate = useNavigate();
	const baseId = useId();
	const { data: task } = useTaskStatus(gid);
	const { data: files } = useTaskFiles(gid);
	const [selectedTab, setSelectedTab] = React.useState<React.Key>(
		`${baseId}-overview`,
	);

	const { data: peers } = useTaskPeers(gid, selectedTab === `${baseId}-peers`);
	const { data: servers } = useTaskServers(
		gid,
		selectedTab === `${baseId}-servers`,
	);

	const [selectedKeys, setSelectedKeys] = React.useState<any>(new Set());

	React.useEffect(() => {
		if (files) {
			const selected = new Set(
				files
					.filter((f: Aria2File) => f.selected === "true")
					.map((f: Aria2File) => f.index),
			);
			setSelectedKeys(selected);
		}
	}, [files]);

	const handleSelectionChange = (keys: any) => {
		setSelectedKeys(keys);
		if (keys === "all") return;

		const selectFileStr = Array.from(keys).sort().join(",");
		aria2.changeOption(gid, { "select-file": selectFileStr });
	};

	return (
		<div className="max-w-6xl mx-auto space-y-6">
			<div className="flex items-center gap-4">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/tasks/all" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<h2 className="text-2xl font-bold tracking-tight">Task Details</h2>
				<code className="text-xs bg-default-100 px-2 py-1 rounded text-default-500">
					{gid}
				</code>
			</div>

			<div className="flex flex-col md:flex-row gap-6">
				{/* Sidebar / Tabs */}
				<div className="w-full md:w-64">
					<Card className="p-2 shadow-sm border-default-100">
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
				<div className="flex-1 min-h-500">
					<Card className="h-full overflow-hidden flex flex-col bg-default-50/30 shadow-sm border-default-100">
						{selectedTab === `${baseId}-overview` && task && (
							<ScrollShadow className="flex-1 p-6">
								<div className="space-y-8 text-foreground">
									<section>
										<h3 className="text-lg font-bold mb-4 flex items-center gap-2">
											<div className="w-1.5 h-1.5 rounded-full bg-accent" />
											Identity
										</h3>
										<div className="grid grid-cols-1 md:grid-cols-2 gap-6 bg-background p-6 rounded-2xl border border-default-100">
											<div className="space-y-1">
												<p className="text-xs text-default-500 uppercase font-bold">
													Filename
												</p>
												<p className="text-base font-semibold break-all">
													{task.bittorrent?.info?.name ||
														task.files[0]?.path?.split("/").pop() ||
														gid}
												</p>
											</div>
											<div className="space-y-1">
												<p className="text-xs text-default-500 uppercase font-bold">
													GID
												</p>
												<p className="font-mono text-sm bg-default-50 px-2 py-0.5 rounded border border-default-100 inline-block">
													{gid}
												</p>
											</div>
											{task.infoHash && (
												<div className="space-y-1 md:col-span-2 pt-2 border-t border-default-50">
													<p className="text-xs text-default-500 uppercase font-bold">
														Info Hash
													</p>
													<p className="font-mono text-sm break-all text-accent">
														{task.infoHash}
													</p>
												</div>
											)}
										</div>
									</section>

									<section>
										<h3 className="text-lg font-bold mb-4 flex items-center gap-2">
											<div className="w-1.5 h-1.5 rounded-full bg-accent" />
											Location
										</h3>
										<div className="bg-background p-6 rounded-2xl border border-default-100">
											<div className="space-y-1">
												<p className="text-xs text-default-500 uppercase font-bold">
													Download Directory
												</p>
												<p className="text-sm font-medium break-all">
													{task.dir}
												</p>
											</div>
										</div>
									</section>

									<section>
										<h3 className="text-lg font-bold mb-4 flex items-center gap-2">
											<div className="w-1.5 h-1.5 rounded-full bg-accent" />
											Status
										</h3>
										<div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-center">
											<div className="p-4 bg-background rounded-2xl border border-default-100 shadow-sm">
												<p className="text-xs text-default-500 uppercase font-bold mb-1">
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
											<div className="p-4 bg-background rounded-2xl border border-default-100 shadow-sm">
												<p className="text-xs text-default-500 uppercase font-bold mb-1">
													Connections
												</p>
												<p className="text-xl font-bold">{task.connections}</p>
											</div>
											<div className="p-4 bg-background rounded-2xl border border-default-100 shadow-sm">
												<p className="text-xs text-default-500 uppercase font-bold mb-1">
													Total Size
												</p>
												<p className="text-base font-bold">
													{formatBytes(task.totalLength)}
												</p>
											</div>
											<div className="p-4 bg-background rounded-2xl border border-default-100 shadow-sm">
												<p className="text-xs text-default-500 uppercase font-bold mb-1">
													Uploaded
												</p>
												<p className="text-base font-bold text-accent">
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
								<div className="p-6 border-b border-default-100 flex justify-between items-center bg-default-50/50">
									<div>
										<span className="text-xl font-bold block leading-tight">
											Files
										</span>
										<span className="text-sm text-default-500">
											{files?.length} items • Toggle to enable/disable
										</span>
									</div>
									<Chip
										variant="soft"
										color="accent"
										size="sm"
										className="font-bold"
									>
										SELECTIVE DOWNLOAD
									</Chip>
								</div>

								<ScrollShadow className="flex-1">
									<ListBox
										aria-label="Files list"
										selectionMode="multiple"
										selectedKeys={selectedKeys}
										onSelectionChange={handleSelectionChange}
										className="p-4"
										variant="default"
									>
										{(files || []).map((file: Aria2File) => (
											<ListBox.Item
												key={file.index}
												id={file.index}
												textValue={file.path}
												className="mb-2 p-4 rounded-2xl border border-default-100 bg-background hover:border-accent/50 transition-all group data-[selected=true]:border-accent/40 data-[selected=true]:bg-accent/5"
											>
												<div className="flex items-start gap-4 w-full">
													<ListBox.ItemIndicator className="mt-1" />

													<div className="flex-1 min-w-0">
														<div className="flex items-center gap-2">
															<IconFile className="w-4.5 h-4.5 text-default-400 group-hover:text-accent transition-colors shrink-0" />
															<span
																className="text-base font-semibold truncate"
																title={file.path}
															>
																{file.path.split("/").pop() || "Unknown File"}
															</span>
														</div>
														<div className="flex gap-4 mt-2 text-sm text-default-500">
															<span className="flex items-center gap-1.5">
																<div className="w-1 h-1 rounded-full bg-default-300" />
																{formatBytes(file.length)}
															</span>
															<span className="flex items-center gap-1.5 text-success font-medium">
																<div className="w-1 h-1 rounded-full bg-success" />
																{(
																	(Number(file.completedLength) /
																		Number(file.length)) *
																	100
																).toFixed(1)}
																% complete
															</span>
														</div>
													</div>
												</div>
											</ListBox.Item>
										))}
									</ListBox>
								</ScrollShadow>
							</div>
						)}

						{selectedTab === `${baseId}-peers` && (
							<ScrollShadow className="flex-1 p-4">
								{!peers || peers.length === 0 ? (
									<div className="flex flex-col items-center justify-center h-full text-default-400 gap-4">
										<div className="p-6 rounded-full bg-default-50">
											<IconPersons className="w-12 h-12 opacity-20" />
										</div>
										<p className="font-medium">No peers connected</p>
									</div>
								) : (
									<div className="space-y-3">
										{peers.map((peer: any) => (
											<div
												key={`${peer.ip}-${peer.port}`}
												className="flex items-center justify-between p-5 rounded-2xl border border-default-100 bg-background shadow-sm hover:border-accent/30 transition-colors"
											>
												<div className="flex items-center gap-4">
													<Avatar className="w-10 h-10 font-bold bg-accent/10 text-accent">
														<Avatar.Fallback>
															{peer.peerId.slice(1, 3).toUpperCase()}
														</Avatar.Fallback>
													</Avatar>
													<div className="flex flex-col gap-1">
														<div className="flex items-center gap-2">
															<span className="text-sm font-bold">
																{peer.ip}
															</span>
															<span className="text-xs bg-default-100 px-1.5 py-0.5 rounded text-default-500">
																:{peer.port}
															</span>
														</div>
														<span className="text-xs font-mono text-default-400 truncate max-w-200">
															{peer.peerId}
														</span>
													</div>
												</div>
												<div className="flex gap-8">
													<div className="flex flex-col items-end">
														<span className="text-[10px] text-default-400 uppercase font-black tracking-wider">
															Down
														</span>
														<span className="text-sm text-success font-bold">
															{formatBytes(peer.downloadSpeed)}/s
														</span>
													</div>
													<div className="flex flex-col items-end border-l border-default-100 pl-8">
														<span className="text-[10px] text-default-400 uppercase font-black tracking-wider">
															Up
														</span>
														<span className="text-sm text-accent font-bold">
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
										<div className="p-6 rounded-full bg-default-50">
											<IconNodesDown className="w-12 h-12 opacity-20" />
										</div>
										<p className="font-medium">
											No server information available
										</p>
									</div>
								) : (
									<div className="space-y-6">
										{servers.map((srv: any) => (
											<div key={srv.index} className="space-y-3">
												<div className="flex items-center gap-2 px-2">
													<div className="w-1 h-3 rounded-full bg-accent" />
													<div className="text-xs font-black text-default-400 uppercase tracking-widest">
														File Index: {srv.index}
													</div>
												</div>
												<div className="space-y-2">
													{srv.servers.map((s: any) => (
														<div
															key={s.uri}
															className="p-4 rounded-2xl border border-default-100 bg-background flex justify-between items-center shadow-sm hover:border-accent-soft-hover transition-colors"
														>
															<div className="flex flex-col min-w-0 gap-1">
																<span className="text-sm font-bold truncate text-accent/80">
																	{s.uri}
																</span>
																<div className="flex items-center gap-2">
																	{s.uri === srv.currentUri && (
																		<Chip
																			size="sm"
																			variant="soft"
																			color="success"
																			className="h-4 text-[10px] font-bold"
																		>
																			CURRENT
																		</Chip>
																	)}
																	<span className="text-xs text-default-500 font-medium">
																		Priority: {s.currentPriority}
																	</span>
																</div>
															</div>
															<div className="flex flex-col items-end shrink-0 ml-4">
																<span className="text-[10px] text-default-400 uppercase font-black">
																	Speed
																</span>
																<span className="text-sm text-success font-bold">
																	{formatBytes(s.downloadSpeed)}/s
																</span>
															</div>
														</div>
													))}
												</div>
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
