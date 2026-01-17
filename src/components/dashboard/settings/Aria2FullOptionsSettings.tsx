import { Accordion, Input, ScrollShadow, Switch, Tooltip } from "@heroui/react";
import React from "react";
import IconCircleInfo from "~icons/gravity-ui/circle-info";
import IconMagnifier from "~icons/gravity-ui/magnifier";
import aria2Options from "../../../lib/aria2-options.json";

export const Aria2FullOptionsSettings: React.FC<{
  options: Record<string, string>;
}> = ({ options: currentOptions }) => {
  const [search, setSearch] = React.useState("");

  // Group options by category
  const groupedOptions = aria2Options.reduce(
    (acc, opt) => {
      if (!acc[opt.category]) {
        acc[opt.category] = [];
      }
      acc[opt.category].push(opt);
      return acc;
    },
    {} as Record<string, typeof aria2Options>,
  );

  const filteredCategories = Object.keys(groupedOptions).filter((cat) => {
    const hasMatch = groupedOptions[cat].some(
      (opt) =>
        opt.name.toLowerCase().includes(search.toLowerCase()) ||
        opt.description.toLowerCase().includes(search.toLowerCase()),
    );
    return hasMatch;
  });

  const renderOption = (opt: (typeof aria2Options)[0]) => {
    const value = currentOptions[opt.name] ?? opt.default ?? "";
    const isBoolean = opt.type === "true|false";

    return (
      <div
        key={opt.name}
        className="flex flex-col gap-2 py-4 border-b border-default-100 last:border-0"
      >
        <div className="flex items-center justify-between gap-4">
          <div className="flex flex-col flex-1">
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm font-bold text-accent">
                {opt.name}
              </span>
              <Tooltip>
                <Tooltip.Trigger>
                  <div className="cursor-help">
                    <IconCircleInfo className="w-3.5 h-3.5 text-default-400" />
                  </div>
                </Tooltip.Trigger>
                <Tooltip.Content className="max-w-xs p-2 text-xs">
                  {opt.description}
                </Tooltip.Content>
              </Tooltip>
            </div>
            <span className="text-xs text-default-500 uppercase">
              {opt.type}
            </span>
          </div>

          <div className="w-1/2 flex justify-end">
            {isBoolean ? (
              <Switch defaultSelected={value === "true"}>
                <Switch.Control>
                  <Switch.Thumb />
                </Switch.Control>
              </Switch>
            ) : (
              <Input
                className="max-w-200"
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
    <div className="flex flex-col h-full space-y-4">
      <div className="flex items-center gap-4 border-b border-default-100 pb-4">
        <div className="relative flex-1">
          <IconMagnifier className="absolute left-3 top-1/2 -translate-y-1/2 text-default-400 z-10 w-4.5 h-4.5" />
          <Input
            placeholder="Search all 180+ options..."
            className="pl-10"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="text-sm text-default-500 whitespace-nowrap">
          {aria2Options.length} options
        </div>
      </div>

      <ScrollShadow className="flex-1 pr-2 h-450">
        <Accordion
          allowsMultipleExpanded
          defaultExpandedKeys={["Basic Options"]}
        >
          {filteredCategories.map((cat) => (
            <Accordion.Item key={cat} id={cat}>
              <Accordion.Heading>
                <Accordion.Trigger className="py-3 text-base font-bold">
                  {cat}
                </Accordion.Trigger>
              </Accordion.Heading>
              <Accordion.Panel>
                <Accordion.Body className="pb-4">
                  {groupedOptions[cat]
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
      </ScrollShadow>
    </div>
  );
};
