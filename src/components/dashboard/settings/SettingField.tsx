import {
  FieldError,
  Input,
  Label,
  ListBox,
  Select,
  Switch,
  TextField,
  Tooltip,
} from "@heroui/react";
import React from "react";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconCircleInfo from "~icons/gravity-ui/circle-info";
import { cn } from "../../../lib/utils";

export interface OptionMetadata {
  name: string;
  category: string;
  type: string;
  default: string | null;
  description: string;
}

export const SettingField: React.FC<{
  opt: OptionMetadata;
  value: string;
  onUpdate: (name: string, value: string) => void;
  isReadOnly?: boolean;
}> = React.memo(({ opt, value, onUpdate, isReadOnly }) => {
  const isBoolean = opt.type === "true|false";
  const isEnum = opt.type.includes("|") && !isBoolean;
  const enumValues = isEnum ? opt.type.split("|") : [];

  // Validation logic
  const validate = (val: string) => {
    if (isReadOnly) return true;
    if (opt.type === "<NUM>" || opt.type === "<N>" || opt.type === "<SEC>") {
      if (val !== "" && isNaN(Number(val))) return "Must be a valid number";
    }
    if (opt.type === "<PORT>") {
      const port = Number(val);
      if (isNaN(port) || port < 1024 || port > 65535)
        return "Port must be between 1024 and 65535";
    }
    if (opt.type === "<SIZE>" || opt.type === "<SPEED>") {
      if (val !== "0" && !/^\d+[KMG]?$/i.test(val))
        return "Invalid format (e.g. 5M, 10K, 0)";
    }
    if (opt.type === "<IPADDRESS>" || opt.type === "<ADDR>") {
      if (val !== "" && !/^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test(val))
        return "Invalid IPv4 address";
    }
    return true;
  };

  return (
    <div
      className={cn(
        "flex flex-col gap-3 py-6 border-b border-border last:border-0",
        isReadOnly && "opacity-60 grayscale pointer-events-none",
      )}
    >
      <div className="flex items-start justify-between gap-6">
        <div className="flex flex-col flex-1">
          <div className="flex items-center gap-2 mb-1">
            <span className="font-mono text-sm font-bold text-accent tracking-tight">
              {opt.name}
            </span>
            {/*<Tooltip>
              <Tooltip.Trigger>
                <div className="cursor-help opacity-60 hover:opacity-100 transition-opacity">
                  <IconCircleInfo className="w-4 h-4 text-muted" />
                </div>
              </Tooltip.Trigger>
              <Tooltip.Content className="max-w-xs p-3 text-xs leading-relaxed">
                {opt.description}
              </Tooltip.Content>
            </Tooltip>*/}
          </div>
          <span className="text-[10px] text-muted uppercase font-black tracking-widest bg-default/10 w-fit px-2 py-0.5 rounded-md">
            {opt.name.replaceAll("-", " ")}
          </span>
        </div>

        <div className="w-1/2 flex justify-end shrink-0 pt-1">
          {isBoolean ? (
            <Switch
              isSelected={value === "true"}
              onChange={(selected) =>
                onUpdate(opt.name, selected ? "true" : "false")
              }
              size="sm"
              isDisabled={isReadOnly}
            >
              <Switch.Control>
                <Switch.Thumb />
              </Switch.Control>
            </Switch>
          ) : isEnum ? (
            <Select
              className="max-w-60 min-w-30"
              value={value || opt.default || enumValues[0]}
              onChange={(key) => onUpdate(opt.name, String(key))}
              isDisabled={isReadOnly}
            >
              <Select.Trigger>
                <Select.Value className="text-sm font-medium" />
                <Select.Indicator className="text-muted">
                  <IconChevronDown className="w-4 h-4" />
                </Select.Indicator>
              </Select.Trigger>
              <Select.Popover className="min-w-30 bg-background">
                <ListBox items={enumValues.map((v) => ({ id: v, name: v }))}>
                  {(item) => (
                    <ListBox.Item
                      id={item.id}
                      textValue={item.name}
                      className="px-3 py-2 rounded-lg text-sm"
                    >
                      <Label>{item.name}</Label>
                    </ListBox.Item>
                  )}
                </ListBox>
              </Select.Popover>
            </Select>
          ) : (
            <TextField
              className="max-w-60 w-full"
              defaultValue={value}
              onChange={(val) => {
                if (validate(val) === true) {
                  onUpdate(opt.name, val);
                }
              }}
              validate={validate}
              validationBehavior="aria"
              isReadOnly={isReadOnly}
            >
              <div className="relative group">
                <Input className="w-full h-10 px-4  text-sm  data-[invalid=true]:border-danger/50" />
                <FieldError className="absolute -bottom-5 right-0 text-[10px] text-danger font-bold uppercase tracking-tight animate-in fade-in slide-in-from-top-1" />
              </div>
            </TextField>
          )}
        </div>
      </div>
    </div>
  );
});
