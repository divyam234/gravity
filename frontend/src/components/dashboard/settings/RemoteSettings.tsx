import { Button, Card, Input, Label, Spinner } from "@heroui/react";
import React, { useState } from "react";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import IconPlus from "~icons/gravity-ui/plus";
import { useRemoteActions, useRemotes } from "../../../hooks/useRemotes";
import { useSettingsStore } from "../../../store/useSettingsStore";

export const RemoteSettings: React.FC = () => {
	const { defaultRemote, setDefaultRemote } = useSettingsStore();
	const { data: remotes, isLoading } = useRemotes();
	const { deleteRemote, createRemote } = useRemoteActions();

	const [isAdding, setIsAdding] = useState(false);
	const [newName, setNewName] = useState("");
	const [newType, setNewType] = useState("");
	const [newParams, setNewParams] = useState("{}");

	const handleCreate = async () => {
		try {
			const parameters = JSON.parse(newParams);
			createRemote.mutate(
				{ name: newName, type: newType, parameters },
				{
					onSuccess: () => {
						setIsAdding(false);
						setNewName("");
						setNewType("");
						setNewParams("{}");
					},
				},
			);
		} catch (e) {
			console.error("Invalid JSON in parameters");
		}
	};

	return (
		<div className="space-y-10">
			<section className="space-y-6">
				<div className="border-b border-border pb-2">
					<h3 className="text-lg font-bold">Default Destination</h3>
					<p className="text-sm text-muted">
						Files will be automatically copied here after download completes.
					</p>
				</div>

				<div className="flex flex-col gap-2">
					<Label className="text-sm font-bold tracking-tight">
						Target Remote & Path
					</Label>
					<Input
						value={defaultRemote}
						onChange={(e) => setDefaultRemote(e.target.value)}
						placeholder="e.g. gdrive:downloads or s3:bucket/folder"
						className="w-full bg-default/10 rounded-xl"
					/>
					<p className="text-[10px] text-muted uppercase font-black tracking-widest">
						Use "remote:" or "remote:/path" syntax.
					</p>
				</div>
			</section>

			<section className="space-y-6">
				<div className="flex items-center justify-between border-b border-border pb-2">
					<div>
						<h3 className="text-lg font-bold">Cloud Remotes</h3>
						<p className="text-sm text-muted">
							Manage your cloud storage connections.
						</p>
					</div>
					<Button
						size="sm"
						variant="ghost"
						onPress={() => setIsAdding(!isAdding)}
						className="rounded-xl"
					>
						<IconPlus className="w-4 h-4 mr-2" />
						New Remote
					</Button>
				</div>

				{isAdding && (
					<Card className="p-4 bg-default/5 border-border shadow-none rounded-2xl space-y-4">
						<div className="grid grid-cols-2 gap-4">
							<div className="space-y-1.5">
								<Label className="text-xs font-bold uppercase tracking-wider text-muted">
									Name
								</Label>
								<Input
									value={newName}
									onChange={(e) => setNewName(e.target.value)}
									placeholder="my-gdrive"
									className="bg-background/50 rounded-lg"
								/>
							</div>
							<div className="space-y-1.5">
								<Label className="text-xs font-bold uppercase tracking-wider text-muted">
									Type
								</Label>
								<Input
									value={newType}
									onChange={(e) => setNewType(e.target.value)}
									placeholder="drive, s3, dropbox..."
									className="bg-background/50 rounded-lg"
								/>
							</div>
						</div>
						<div className="space-y-1.5">
							<Label className="text-xs font-bold uppercase tracking-wider text-muted">
								Parameters (JSON)
							</Label>
							<Input
								value={newParams}
								onChange={(e) => setNewParams(e.target.value)}
								placeholder='{"token": "..."}'
								className="bg-background/50 rounded-lg font-mono"
							/>
						</div>
						<div className="flex justify-end gap-2">
							<Button variant="ghost" onPress={() => setIsAdding(false)}>
								Cancel
							</Button>
							<Button
								className="bg-accent text-accent-foreground"
								onPress={handleCreate}
								isPending={createRemote.isPending}
							>
								Create
							</Button>
						</div>
					</Card>
				)}

				<div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
					{isLoading ? (
						<div className="col-span-full flex justify-center py-8">
							<Spinner size="sm" />
						</div>
					) : (
						remotes?.map((remote) => (
							<Card
								key={remote.name}
								className="flex flex-row items-center justify-between p-3 px-4 bg-default/5 border-border shadow-none rounded-xl group"
							>
								<div className="flex items-center gap-3">
									<div className="w-2 h-2 rounded-full bg-success" />
									<span className="font-bold tracking-tight">{remote.name}</span>
								</div>
								<div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
									<Button
										isIconOnly
										size="sm"
										variant="ghost"
										onPress={() => setDefaultRemote(`${remote.name}:`)}
										className="h-8 w-8 min-w-0"
									>
										<span className="text-[10px] font-black">DEF</span>
									</Button>
									<Button
										isIconOnly
										size="sm"
										variant="ghost"
										className="text-danger h-8 w-8 min-w-0"
										onPress={() => deleteRemote.mutate(remote.name)}
										isPending={deleteRemote.isPending}
									>
										<IconTrashBin className="w-4 h-4" />
									</Button>
								</div>
							</Card>
						))
					)}
					{!isLoading && remotes?.length === 0 && (
						<div className="col-span-full text-center py-8 text-muted">
							No remotes configured. Add one to start uploading.
						</div>
					)}
				</div>
			</section>
		</div>
	);
};
