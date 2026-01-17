import { Accordion, Input } from "@heroui/react";
import React from "react";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconMagnifier from "~icons/gravity-ui/magnifier";
import { useAria2Actions } from "../../../hooks/useAria2";
import aria2Options from "../../../lib/aria2-options.json";
import { cn } from "../../../lib/utils";
import { SettingField } from "./SettingField";

export const Aria2FullOptionsSettings: React.FC<{
  options: Record<string, string>;
}> = ({ options: currentOptions }) => {
  const [search, setSearch] = React.useState("");
  const [expandedKeys, setExpandedKeys] = React.useState<Set<React.Key>>(
    new Set(["Basic Options"]),
  );
  const { changeGlobalOption } = useAria2Actions();

  // Debounced update handler
  const timeoutRef = React.useRef<
    Record<string, ReturnType<typeof setTimeout>>
  >({});
  const handleUpdate = React.useCallback(
    (name: string, value: string) => {
      if (timeoutRef.current[name]) {
        clearTimeout(timeoutRef.current[name]);
      }
      timeoutRef.current[name] = setTimeout(() => {
        changeGlobalOption.mutate({ [name]: value });
      }, 500);
    },
    [changeGlobalOption],
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

  return (
    <div className="flex flex-col h-full space-y-6">
      <div className="flex items-center gap-4 border-b border-border pb-6 shrink-0">
        <div className="relative flex-1">
          <IconMagnifier className="absolute left-4 top-1/2 -translate-y-1/2 text-muted z-10 w-4.5 h-4.5" />
          <Input
            placeholder="Search all 180+ options..."
            className="w-full h-11 pl-11 pr-4 bg-default/10 rounded-2xl text-sm outline-none transition-all focus:bg-default/30 focus:ring-2 focus:ring-accent/20"
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
        className="px-0 gap-5 flex flex-col"
      >
        {filteredCategories.map((cat) => (
          <Accordion.Item
            key={cat}
            id={cat}
            className={cn("group/item rounded-[32px] border border-border")}
          >
            <Accordion.Heading>
              <Accordion.Trigger className="m-2 px-6 py-5 rounded-[24px] hover:bg-default/10 transition-all group-data-[expanded=true]/item:text-accent">
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
                    .map((opt) => (
                      <SettingField
                        key={opt.name}
                        opt={opt as any}
                        value={currentOptions[opt.name] ?? opt.default ?? ""}
                        onUpdate={handleUpdate}
                      />
                    ))}
              </Accordion.Body>
            </Accordion.Panel>
          </Accordion.Item>
        ))}
      </Accordion>
    </div>
  );
};
