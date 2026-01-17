import { Accordion, Input, Switch, Tooltip } from "@heroui/react";
import React from "react";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconCircleInfo from "~icons/gravity-ui/circle-info";
import IconMagnifier from "~icons/gravity-ui/magnifier";
import aria2Options from "../../../lib/aria2-options.json";
import { cn } from "../../../lib/utils";

export const Aria2FullOptionsSettings: React.FC<{
  options: Record<string, string>;
}> = ({ options: currentOptions }) => {
  const [search, setSearch] = React.useState("");
  const [expandedKeys, setExpandedKeys] = React.useState<Set<React.Key>>(
    new Set(["Basic Options"]),
  );

  // Group options by category
  const groupedOptions = React.useMemo(() => {
    return aria2Options.reduce(
      (acc, opt) => {
        if (!acc[opt.category]) {
          acc[opt.category] = [];
        }
        acc[opt.category].push(opt);
        return acc;
      },
      {} as Record<string, typeof aria2Options>,
    );
  }, []);

  const filteredCategories = React.useMemo(() => {
    return Object.keys(groupedOptions).filter((cat) => {
      const hasMatch = groupedOptions[cat].some(
        (opt) =>
          opt.name.toLowerCase().includes(search.toLowerCase()) ||
          opt.description.toLowerCase().includes(search.toLowerCase()),
      );
      return hasMatch;
    });
  }, [groupedOptions, search]);

  // Auto-expand all matching categories when searching
  React.useEffect(() => {
    if (search.trim()) {
      setExpandedKeys(new Set(filteredCategories));
    }
  }, [search, filteredCategories]);

  const renderOption = (opt: (typeof aria2Options)[0]) => {
    const value = currentOptions[opt.name] ?? opt.default ?? "";
    const isBoolean = opt.type === "true|false";

    return (
      <div
        key={opt.name}
        className="flex flex-col gap-3 py-6 border-b border-border last:border-0"
      >
        <div className="flex items-center justify-between gap-6">
          <div className="flex flex-col flex-1">
            <div className="flex items-center gap-2 mb-1">
              <span className="font-mono text-sm font-bold text-accent tracking-tight">
                {opt.name}
              </span>
              <Tooltip>
                <Tooltip.Trigger>
                  <div className="cursor-help opacity-60 hover:opacity-100 transition-opacity">
                    <IconCircleInfo className="w-4 h-4 text-muted" />
                  </div>
                </Tooltip.Trigger>
                <Tooltip.Content className="max-w-xs p-3 text-xs leading-relaxed">
                  {opt.description}
                </Tooltip.Content>
              </Tooltip>
            </div>
            <span className="text-[10px] text-muted uppercase font-black tracking-widest bg-default/10 w-fit px-2 py-0.5 rounded-md">
              {opt.type}
            </span>
          </div>

          <div className="w-1/2 flex justify-end shrink-0">
            {isBoolean ? (
              <Switch defaultSelected={value === "true"} size="sm">
                <Switch.Control>
                  <Switch.Thumb />
                </Switch.Control>
              </Switch>
            ) : (
              <Input
                className="max-w-240 h-10 px-4 rounded-xl text-sm"
                defaultValue={value}
                placeholder={opt.default ?? ""}
              />
            )}
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="flex flex-col h-full space-y-6">
      <div className="flex items-center gap-4 border-b border-border pb-6 shrink-0">
        <div className="relative flex-1">
          <IconMagnifier className="absolute left-4 top-1/2 -translate-y-1/2 text-muted z-10 w-4.5 h-4.5" />
          <Input
            placeholder="Search all 180+ options..."
            className="w-full h-11 pl-11 pr-4 rounded-2xl text-sm outline-none"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="text-[10px] font-black uppercase text-muted tracking-widest bg-default/30 px-3 py-1.5 rounded-full">
          {aria2Options.length} total
        </div>
      </div>

      <Accordion
        allowsMultipleExpanded
        expandedKeys={expandedKeys as any}
        onExpandedChange={(keys) => setExpandedKeys(keys as any)}
        // variant="surface"
        className="px-0 gap-5 flex flex-col"
      >
        {filteredCategories.map((cat) => (
          <Accordion.Item
            key={cat}
            id={cat}
            className="group/item rounded-[32px] border border-border"
          >
            <Accordion.Heading>
              <Accordion.Trigger className="m-2 px-6 py-5 rounded-[24px] hover:bg-default/20 transition-all group-data-[expanded=true]/item:text-accent">
                <span className="text-base font-bold tracking-tight text-foreground/90 group-data-[expanded=true]/item:text-accent">
                  {cat}
                </span>
                <Accordion.Indicator className="text-muted group-data-[expanded=true]/item:text-accent">
                  <IconChevronDown className="w-5 h-5 transition-transform duration-300" />
                </Accordion.Indicator>
              </Accordion.Trigger>
            </Accordion.Heading>
            <Accordion.Panel>
              <Accordion.Body className="px-8 pb-8 pt-2">
                {expandedKeys.has(cat) &&
                  groupedOptions[cat]
                    .filter(
                      (opt) =>
                        opt.name.toLowerCase().includes(search.toLowerCase()) ||
                        opt.description
                          .toLowerCase()
                          .includes(search.toLowerCase()),
                    )
                    .map(renderOption)}
              </Accordion.Body>
            </Accordion.Panel>
          </Accordion.Item>
        ))}
      </Accordion>
    </div>
  );
};
