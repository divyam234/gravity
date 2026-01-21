import { Button, Card, Label, ScrollShadow, Input, TextField } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { api } from "../lib/api";
import { toast } from "sonner";
import { cn } from "../lib/utils";
import { useGlobalOption, useEngineActions, globalOptionOptions } from "../hooks/useEngine";

export const Route = createFileRoute("/settings/browser")({
	component: BrowserSettingsPage,
	loader: async ({ context: { queryClient } }) => {
		queryClient.prefetchQuery(globalOptionOptions());
	},
});

function BrowserSettingsPage() {
	const navigate = useNavigate();
  const queryClient = useQueryClient();
	const { data: options } = useGlobalOption();
	const { changeGlobalOption } = useEngineActions();

	const [cacheTTL, setCacheTTL] = useState("5m");
  const [isValid, setIsValid] = useState(true);

	useEffect(() => {
		if (options?.fileBrowserCacheTTL) {
			setCacheTTL(options.fileBrowserCacheTTL);
		}
	}, [options]);

  const validateDuration = (val: string) => {
    if (val.toLowerCase() === "unlimited") return true;
    // Matches patterns like 1m, 5h, 2d, 10s
    return /^\d+[smhd]$/i.test(val);
  };

	const handleTTLChange = (val: string) => {
		setCacheTTL(val);
    const valid = validateDuration(val);
    setIsValid(valid);
    
    if (valid) {
		  changeGlobalOption.mutate({ fileBrowserCacheTTL: val });
    }
	};

  const purgeMutation = useMutation({
    mutationFn: () => api.purgeFileCache(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files"] });
      toast.success("File browser cache purged");
    },
    onError: (err: any) => {
      toast.error("Failed to purge cache: " + err.message);
    }
  });

	return (
		<div className="flex flex-col h-full space-y-6">
			<div className="flex items-center gap-4 px-2 shrink-0">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/settings" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<div>
					<h2 className="text-2xl font-bold tracking-tight">Browser</h2>
					<p className="text-xs text-muted">File browsing performance & display</p>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-10">
						{/* Performance */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Performance</h3>
							</div>
              
              <div className="grid gap-4">
                <Card className="p-6 bg-background/50 border-border">
                  <div className="flex items-center justify-between">
                    <div className="flex-1 mr-8">
                      <Label className="text-sm font-bold">Cache Duration</Label>
                      <p className="text-xs text-muted mt-0.5">
                        How long to cache file listings (e.g., 1m, 1h, 1d, or "unlimited")
                      </p>
                    </div>
                    
                    <TextField className="w-48">
                      <Input
                        value={cacheTTL}
                        onChange={(e) => handleTTLChange(e.target.value)}
                        placeholder="e.g. 5m"
                        className={cn(
                          "h-11 bg-default/10 rounded-2xl border-none text-center font-bold",
                          !isValid && "ring-2 ring-danger/50"
                        )}
                      />
                    </TextField>
                  </div>
                  {!isValid && (
                    <p className="text-[10px] text-danger mt-2 font-bold uppercase tracking-wider">
                      Invalid duration format (use s, m, h, d or "unlimited")
                    </p>
                  )}
                </Card>

                <Card className="p-6 bg-background/50 border-border">
                  <div className="flex items-center justify-between">
                    <div>
                      <Label className="text-sm font-bold">Purge Cache</Label>
                      <p className="text-xs text-muted mt-0.5">
                        Immediately clear all cached file listings across all remotes
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      className="rounded-xl font-bold text-danger hover:bg-danger/10"
                      onPress={() => purgeMutation.mutate()}
                      isPending={purgeMutation.isPending}
                    >
                      <IconTrashBin className="w-4 h-4 mr-2" />
                      Purge Now
                    </Button>
                  </div>
                </Card>
              </div>
						</section>
					</div>
				</ScrollShadow>
			</div>
		</div>
	);
}
