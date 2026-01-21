import { Button, Card, Label, ScrollShadow, Spinner, Chip, Select, ListBox, Input, TextField, Checkbox, Tooltip } from "@heroui/react";
import type { Selection } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useEffect, useRef } from "react";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconMagnifyingGlass from "~icons/gravity-ui/magnifier";
import IconArrowsRotateRight from "~icons/gravity-ui/arrows-rotate-right";
import IconCircleCheckFill from "~icons/gravity-ui/circle-check-fill";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconGear from "~icons/gravity-ui/gear";
import { useSearch } from "../hooks/useSearch";
import { cn } from "../lib/utils";

export const Route = createFileRoute("/settings/search")({
  component: SearchSettingsPage,
});

const INTERVAL_OPTIONS = [
  { value: 0, label: "Disabled" },
  { value: 60, label: "Hourly" },
  { value: 360, label: "Every 6 Hours" },
  { value: 720, label: "Every 12 Hours" },
  { value: 1440, label: "Daily" },
  { value: 10080, label: "Weekly" },
];

function SearchSettingsPage() {
	const navigate = useNavigate();
	const { configs, isLoading, triggerIndex, updateConfigs } = useSearch();

	const [includedRemotes, setIncludedRemotes] = useState<Selection>(new Set());
	const [isExpanded, setIsExpanded] = useState(false);
	const initialized = useRef(false);

	const form = useForm({
		defaultValues: {
			interval: 1440,
			excludedPatterns: "",
			includedExtensions: "",
			minSizeBytes: 0,
		},
		onSubmit: async ({ value }) => {
			const batch: Record<string, any> = {};
			configs.forEach((config) => {
				const isIncluded =
					includedRemotes === "all" || includedRemotes.has(config.remote);
				batch[config.remote] = {
					...value,
					interval: isIncluded ? value.interval : 0,
				};
			});
			updateConfigs.mutate(batch);
		},
	});

	useEffect(() => {
		if (configs.length > 0 && !initialized.current) {
			const template = configs.find(c => c.autoIndexIntervalMin > 0 || c.excludedPatterns || c.includedExtensions || (c.minSizeBytes ?? 0) > 0) || configs[0];
			
			form.setFieldValue("interval", template.autoIndexIntervalMin || 1440);
			form.setFieldValue("excludedPatterns", template.excludedPatterns || "");
			form.setFieldValue("includedExtensions", template.includedExtensions || "");
			form.setFieldValue("minSizeBytes", template.minSizeBytes || 0);

			const included = new Set(
				configs
					.filter((c) => c.autoIndexIntervalMin > 0 || c.lastIndexedAt)
					.map((c) => c.remote),
			);
			
			if (configs.every((c) => c.autoIndexIntervalMin === 0 && !c.lastIndexedAt)) {
				setIncludedRemotes("all");
			} else {
				setIncludedRemotes(included);
			}
			
			initialized.current = true;
		}
	}, [configs, form]);

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
          <h2 className="text-2xl font-bold tracking-tight">Search Indexing</h2>
          <p className="text-xs text-muted">
            Global indexing rules & remote management
          </p>
        </div>
      </div>

      <div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
        <ScrollShadow className="h-full">
          <div className="max-w-4xl mx-auto p-8 space-y-10">
            {isLoading ? (
              <div className="flex justify-center py-24">
                <Spinner size="lg" />
              </div>
            ) : configs.length === 0 ? (
              <Card className="p-8 bg-background/50 border-border border-dashed text-center">
                <div className="w-16 h-16 bg-default/10 rounded-full flex items-center justify-center mb-4 mx-auto">
                  <IconMagnifyingGlass className="w-8 h-8 text-muted" />
                </div>
                <h4 className="font-bold text-lg mb-2">No remotes available</h4>
                <p className="text-sm text-muted mb-6">
                  Configure cloud remotes first to enable indexing.
                </p>
                <Button
                  variant="primary"
                  className="rounded-xl font-bold"
                  onPress={() => navigate({ to: "/settings/cloud" })}
                >
                  Go to Cloud Settings
                </Button>
              </Card>
            ) : (
              <>
                <section className="space-y-6">
                  <div className="flex items-center gap-3">
                    <div className="w-1.5 h-6 bg-accent rounded-full" />
                    <h3 className="text-lg font-bold">Global Configuration</h3>
                  </div>

                                    <Card className="bg-background/50 border-border overflow-hidden">

                                      <div className="p-6 flex flex-col md:flex-row md:items-center justify-between gap-6">

                                        <div className="flex-1">

                                          <h4 className="font-bold text-lg">Indexing Rules</h4>

                                          <p className="text-sm text-muted">

                                            Settings applied to all selected remotes below

                                          </p>

                                        </div>

                  

                                        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 shrink-0">

                                          <div className="flex flex-col gap-1.5">

                                            <Label className="text-[10px] font-black uppercase tracking-widest text-muted ml-1">

                                              Update Frequency

                                            </Label>

                                            <form.Field

                                              name="interval"

                                              children={(field) => (

                                                <Select

                                                  selectedKey={String(field.state.value)}

                                                  onSelectionChange={(key) => field.handleChange(parseInt(key as string))}

                                                >

                                                  <Select.Trigger className="h-[38px] px-3 bg-default/10 border border-border rounded-xl outline-none focus:ring-2 focus:ring-accent/50 transition-all min-w-[160px]">

                                                    <Select.Value className="text-sm font-bold" />

                                                    <Select.Indicator className="text-muted">

                                                      <IconChevronDown className="w-4 h-4" />

                                                    </Select.Indicator>

                                                  </Select.Trigger>

                                                  <Select.Popover className="min-w-[160px] p-2 bg-background border border-border rounded-2xl shadow-xl">

                                                    <ListBox>

                                                      {INTERVAL_OPTIONS.filter((o) => o.value > 0).map(

                                                        (opt) => (

                                                          <ListBox.Item

                                                            key={opt.value}

                                                            id={String(opt.value)}

                                                            className="rounded-xl py-2 px-3 data-[selected=true]:bg-accent/10 data-[selected=true]:text-accent font-bold text-sm cursor-pointer"

                                                          >

                                                            {opt.label}

                                                          </ListBox.Item>

                                                        ),

                                                      )}

                                                    </ListBox>

                                                  </Select.Popover>

                                                </Select>

                                              )}

                                            />

                                          </div>

                  

                                          <div className="flex items-end h-full pt-5 md:pt-0 gap-2">

                                            <Button

                                              variant="ghost"

                                              isIconOnly

                                              className={cn(

                                                "h-[38px] w-[38px] rounded-xl",

                                                isExpanded && "bg-accent/10 text-accent",

                                              )}

                                              onPress={() => setIsExpanded(!isExpanded)}

                                            >

                                              <IconGear className="w-4 h-4" />

                                            </Button>

                                            													<Button

                                            														variant="primary"

                                            														className="rounded-xl font-bold h-[38px] px-6 shadow-lg shadow-primary/20"

                                            														onPress={() => form.handleSubmit()}

                                            														isPending={updateConfigs.isPending}

                                            													>

                                            														Apply Rules

                                            													</Button>

                                            

                                          </div>

                                        </div>

                                      </div>

                  

                                      {isExpanded && (

                                        <div className="px-6 pb-8 pt-2 border-t border-border/50 animate-in slide-in-from-top-2 duration-200">

                                          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 mt-4">

                                        <div className="space-y-6">

                                              <form.Field

                                                name="excludedPatterns"

                                                children={(field) => (

                                                  <TextField className="w-full">

                                                    <Label className="text-sm font-bold mb-1.5 block text-foreground/80">

                                                      Exclude Patterns (Regex)

                                                    </Label>

                                                    <Input

                                                      value={field.state.value}

                                                      onChange={(e) => field.handleChange(e.target.value)}

                                                      placeholder="e.g. /node_modules/|/\.git/"

                                                      className="h-11 bg-default/10 rounded-2xl border-none font-mono text-xs"

                                                    />

                                                  </TextField>

                                                )}

                                              />

                                              <form.Field

                                                name="includedExtensions"

                                                children={(field) => (

                                                  <TextField className="w-full">

                                                    <Label className="text-sm font-bold mb-1.5 block text-foreground/80">

                                                      Include Extensions

                                                    </Label>

                                                    <Input

                                                      value={field.state.value}

                                                      onChange={(e) => field.handleChange(e.target.value)}

                                                      placeholder="e.g. mp4, mkv, iso"

                                                      className="h-11 bg-default/10 rounded-2xl border-none font-mono text-xs"

                                                    />

                                                  </TextField>

                                                )}

                                              />

                                            </div>

                                            <div className="space-y-6">

                                              <form.Field

                                                name="minSizeBytes"

                                                children={(field) => (

                                                  <TextField className="w-full">

                                                    <Label className="text-sm font-bold mb-1.5 block text-foreground/80">

                                                      Minimum File Size (MB)

                                                    </Label>

                                                    <Input

                                                      type="number"

                                                      value={String(Math.floor(field.state.value / (1024 * 1024)))}

                                                      onChange={(e) => field.handleChange(parseInt(e.target.value || "0") * 1024 * 1024)}

                                                      placeholder="0"

                                                      className="h-11 bg-default/10 rounded-2xl border-none font-bold text-center"

                                                    />

                                                  </TextField>

                                                )}

                                              />

                                              <div className="pt-7">

                                                <p className="text-xs text-muted leading-relaxed italic">

                                                  These filters will skip matching files during

                                                  the recursive cloud scan. Click "Apply Rules" to

                                                  update all selected remotes below.

                                                </p>

                                              </div>

                                            </div>

                                          </div>

                                        </div>

                                      )}

                                    </Card>

                  

                  
                </section>

                <section className="space-y-6">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-1.5 h-6 bg-accent rounded-full" />
                      <h3 className="text-lg font-bold">Target Remotes</h3>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        variant="ghost"
                        className="rounded-xl font-bold text-[10px] uppercase tracking-widest"
                        onPress={() => setIncludedRemotes("all")}
                      >
                        Select All
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="rounded-xl font-bold text-[10px] uppercase tracking-widest"
                        onPress={() => setIncludedRemotes(new Set())}
                      >
                        Deselect All
                      </Button>
                    </div>
                  </div>

									<ListBox
										aria-label="Target Remotes"
										items={configs}
										selectionMode="multiple"
										selectedKeys={includedRemotes}
										onSelectionChange={setIncludedRemotes}
										className="p-0 gap-3"
									>
										{(config) => (
											<ListBox.Item
												id={config.remote}
												textValue={config.remote}
												className={cn(
													"p-0 rounded-3xl transition-all duration-200 border border-border outline-none cursor-pointer",
													"bg-background/50 hover:bg-default/5",
													"data-[selected=true]:bg-accent/5 data-[selected=true]:border-accent/20",
												)}
											>
												<div className="p-5 flex items-center justify-between gap-4 w-full">
                          <div className="flex items-center gap-5 flex-1 min-w-0">
                            <ListBox.ItemIndicator>
                              {({ isSelected }) => (
                                <Checkbox
                                  isSelected={isSelected}
                                  isReadOnly
                                  className="pointer-events-none"
                                />
                              )}
                            </ListBox.ItemIndicator>
                            <div className="min-w-0">
                              <h4 className="font-bold text-base capitalize">
                                {config.remote}
                              </h4>
                              <div className="flex items-center gap-3 mt-1">
                                {config.status === "indexing" ? (
                                  <Chip
                                    color="accent"
                                    size="sm"
                                    variant="soft"
                                    className="animate-pulse h-5 text-[9px] font-black uppercase"
                                  >
                                    Indexing...
                                  </Chip>
                                ) : config.lastIndexedAt ? (
                                  <span className="text-[10px] text-muted flex items-center gap-1.5 whitespace-nowrap">
                                    <IconCircleCheckFill className="w-3 h-3 text-success" />
                                    Last indexed:{" "}
                                    {new Date(
                                      config.lastIndexedAt,
                                    ).toLocaleString()}
                                  </span>
                                ) : (
                                  <span className="text-[10px] text-warning font-bold uppercase tracking-tight">
                                    Never indexed
                                  </span>
                                )}
                                {config.status === "error" && (
                                  <Tooltip>
                                    <Tooltip.Trigger>
                                      <Chip
                                        color="danger"
                                        size="sm"
                                        variant="soft"
                                        className="h-5 text-[9px] font-black uppercase"
                                      >
                                        Error
                                      </Chip>
                                    </Tooltip.Trigger>
                                    <Tooltip.Content className="p-2 text-xs max-w-xs">
                                      {config.errorMsg}
                                    </Tooltip.Content>
                                  </Tooltip>
                                )}
                              </div>
                            </div>
                          </div>

                          <div className="shrink-0">
                            <Button
                              size="sm"
                              variant={
                                config.status === "indexing"
                                  ? "secondary"
                                  : "ghost"
                              }
                              className="rounded-xl font-bold h-10 px-5"
                              isDisabled={config.status === "indexing"}
                              isPending={
                                triggerIndex.isPending &&
                                triggerIndex.variables === config.remote
                              }
                              														onPress={() => {
                              															// Stop propagation doesn't work easily with HeroUI Button/ListBox combo
                              															// but we can try to use onPress properly
                              															triggerIndex.mutate(config.remote);
                              														}}
                              
                            >
                              <IconArrowsRotateRight
                                className={cn(
                                  "w-4 h-4 mr-2",
                                  config.status === "indexing" &&
                                    "animate-spin",
                                )}
                              />
                              {config.lastIndexedAt
                                ? "Rebuild Index"
                                : "Start Indexing"}
                            </Button>
                          </div>
                        </div>
                      </ListBox.Item>
                    )}
                  </ListBox>
                </section>
              </>
            )}
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}
