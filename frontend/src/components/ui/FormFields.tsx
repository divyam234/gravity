import {
  Label,
  TextField,
  Select,
  ListBox,
  InputGroup,
  FieldError,
  Switch,
  Card,
  type TextFieldProps,
  type SelectRootProps,
  type SwitchProps,
  cn,
} from "@heroui/react";
import type { DeepKeys, DeepValue } from "@tanstack/react-form";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import React from "react";
import type { ZodType } from "zod";

// --- Simplified Form API ---
interface AppFormApi<_TData> {
  Field: any;
  handleSubmit: () => void;
  setFieldValue: (name: any, value: any) => void;
  reset: () => void;
}

// --- Dynamic Settings Types ---

export type SettingFieldType =
  | "text"
  | "number"
  | "switch"
  | "select"
  | "password";

export interface SettingFieldOption {
  label: string;
  value: string | number;
}

export interface SettingFieldConfig<TData> {
  name: DeepKeys<TData>;
  type: SettingFieldType;
  label: string;
  description?: string;
  placeholder?: string;
  options?: SettingFieldOption[]; // For select
  schema?: ZodType<any>; // Optional Zod schema for validation
  format?: (value: any) => string;
  parse?: (value: string) => any;
  startContent?: React.ReactNode;
  endContent?: React.ReactNode;
  className?: string;
  colSpan?: number; // Grid column span (default 1)
}

export interface SettingDividerConfig {
  type: "divider";
}

export type SettingItemConfig<TData> =
  | SettingFieldConfig<TData>
  | SettingDividerConfig;

export interface SettingGroupConfig<TData> {
  id: string;
  title: string;
  description?: string;
  fields: SettingItemConfig<TData>[];
}

export interface DynamicSettingsProps<TData> {
  form: AppFormApi<TData>;
  groups: SettingGroupConfig<TData>[];
  className?: string;
}

// --- Dynamic Settings Component ---

export function DynamicSettings<TData>({
  form,
  groups,
  className,
}: DynamicSettingsProps<TData>) {
  return (
    <div className={cn("space-y-10", className)}>
      {groups.map((group) => (
        <section key={group.id} className="space-y-6">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-1.5 h-6 bg-accent rounded-full" />
            <h3 className="text-lg font-bold">{group.title}</h3>
          </div>

          <Card className="bg-background/50 border-border overflow-hidden">
            <Card.Content className="p-6">
              {group.description && (
                <div className="p-4 bg-default/10 rounded-xl border border-dashed border-border mb-6">
                  <p className="text-sm text-muted text-center">
                    {group.description}
                  </p>
                </div>
              )}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {group.fields.map((item, index) => {
                  if ("type" in item && item.type === "divider") {
                    return (
                      <div
                        key={`div-${group.id}-${index}`}
                        className="col-span-full h-px bg-border"
                      />
                    );
                  }

                  const config = item as SettingFieldConfig<TData>;
                  return (
                    <div
                      key={String(config.name)}
                      className={cn(
                        config.colSpan &&
                          `col-span-${config.colSpan} md:col-span-${config.colSpan}`,
                      )}
                      style={
                        config.colSpan === 2
                          ? { gridColumn: "1 / -1" }
                          : undefined
                      }
                    >
                      <RenderField form={form} config={config} />
                    </div>
                  );
                })}
              </div>
            </Card.Content>
          </Card>
        </section>
      ))}
    </div>
  );
}

// --- Field Renderer ---

function RenderField<TData>({
  form,
  config,
}: {
  form: AppFormApi<TData>;
  config: SettingFieldConfig<TData>;
}) {
  switch (config.type) {
    case "switch":
      return (
        <FormSwitch
          form={form}
          name={config.name}
          label={config.label}
          description={config.description}
          className={config.className}
        />
      );
    case "select":
      return (
        <FormSelect
          form={form}
          name={config.name}
          label={config.label}
          description={config.description}
          items={config.options || []}
          className={config.className}
        />
      );
    case "number":
      return (
        <FormTextField
          form={form}
          name={config.name}
          label={config.label}
          placeholder={config.placeholder}
          description={config.description}
          type="number"
          parse={config.parse || Number}
          format={config.format}
          startContent={config.startContent}
          endContent={config.endContent}
          className={config.className}
        />
      );
    case "text":
    case "password":
    default:
      return (
        <FormTextField
          form={form}
          name={config.name}
          label={config.label}
          placeholder={config.placeholder}
          description={config.description}
          type={config.type === "password" ? "password" : "text"}
          parse={config.parse}
          format={config.format}
          startContent={config.startContent}
          endContent={config.endContent}
          className={config.className}
        />
      );
  }
}

// --- Base Components ---

interface FormSwitchProps<
  TFormData,
  TName extends DeepKeys<TFormData>,
> extends Omit<
  SwitchProps,
  "name" | "form" | "children" | "onChange" | "isSelected" | "defaultSelected"
> {
  form: AppFormApi<TFormData>;
  name: TName;
  label?: React.ReactNode;
  description?: string;
}

export function FormSwitch<TFormData, TName extends DeepKeys<TFormData>>({
  form,
  name,
  label,
  description,
  className,
  ...props
}: FormSwitchProps<TFormData, TName>) {
  return (
    <form.Field name={name}>
      {(field: any) => (
        <div
          className={cn("flex items-center justify-between h-full", className)}
        >
          <div className="flex flex-col gap-0.5 mr-4">
            {label && (
              <Label className="text-[10px] font-black uppercase tracking-widest text-muted mb-1.5 block">
                {label}
              </Label>
            )}
            {description && (
              <p className="text-[10px] text-muted font-medium leading-relaxed">
                {description}
              </p>
            )}
          </div>
          <Switch
            isSelected={!!field.state.value}
            onChange={(isSelected) =>
              field.handleChange(isSelected as DeepValue<TFormData, TName>)
            }
            {...props}
          >
            <Switch.Control>
              <Switch.Thumb />
            </Switch.Control>
          </Switch>
        </div>
      )}
    </form.Field>
  );
}

interface FormTextFieldProps<
  TFormData,
  TName extends DeepKeys<TFormData>,
> extends Omit<
  TextFieldProps,
  "name" | "form" | "children" | "onChange" | "value" | "defaultValue"
> {
  form: AppFormApi<TFormData>;
  name: TName;
  label?: React.ReactNode;
  placeholder?: string;
  description?: string;
  format?: (value: DeepValue<TFormData, TName>) => string;
  parse?: (value: string) => DeepValue<TFormData, TName>;
  startContent?: React.ReactNode;
  endContent?: React.ReactNode;
}

export function FormTextField<TFormData, TName extends DeepKeys<TFormData>>({
  form,
  name,
  label,
  className,
  placeholder,
  format,
  parse,
  startContent,
  endContent,
  ...props
}: FormTextFieldProps<TFormData, TName>) {
  return (
    <form.Field name={name}>
      {(field: any) => {
        const displayValue = format
          ? format(field.state.value)
          : String(field.state.value ?? "");
        return (
          <TextField
            className={cn("w-full group", className)}
            isInvalid={field.state.meta.isTouched && !field.state.meta.isValid}
            validationBehavior="aria"
            {...props}
          >
            {label && (
              <Label className="text-[10px] font-black uppercase tracking-widest text-muted mb-1.5 block">
                {label}
              </Label>
            )}
            {props.description && (
              <p className="text-[10px] text-muted mb-2 -mt-1 font-medium leading-relaxed">
                {props.description}
              </p>
            )}
            <InputGroup className="relative">
              {startContent && (
                <InputGroup.Prefix>{startContent}</InputGroup.Prefix>
              )}
              <InputGroup.Input
                value={displayValue}
                onChange={(e) => {
                  const val = e.target.value;
                  field.handleChange(
                    parse ? parse(val) : (val as DeepValue<TFormData, TName>),
                  );
                }}
                onBlur={field.handleBlur}
                placeholder={placeholder}
                className="h-11 bg-default/10 rounded-2xl border-none text-xs w-full"
              />
              {endContent && (
                <InputGroup.Suffix>{endContent}</InputGroup.Suffix>
              )}
              {field.state.meta.isTouched && !field.state.meta.isValid ? (
                <FieldError className="absolute -bottom-5 right-0 text-[10px] text-danger font-bold uppercase tracking-tight animate-in fade-in slide-in-from-top-1">
                  {field.state.meta.errors[0]?.message ||
                    field.state.meta.errors[0]}
                </FieldError>
              ) : null}
            </InputGroup>
          </TextField>
        );
      }}
    </form.Field>
  );
}

interface FormSelectOption {
  value: string | number;
  label: string;
}

interface FormSelectProps<
  TFormData,
  TName extends DeepKeys<TFormData>,
> extends Omit<
  SelectRootProps<any>,
  | "name"
  | "form"
  | "children"
  | "onChange"
  | "selectedKey"
  | "onSelectionChange"
  | "items"
> {
  form: AppFormApi<TFormData>;
  name: TName;
  label?: React.ReactNode;
  description?: string;
  items: FormSelectOption[];
}

export function FormSelect<TFormData, TName extends DeepKeys<TFormData>>({
  form,
  name,
  label,
  description,
  items,
  className,
  ...props
}: FormSelectProps<TFormData, TName>) {
  return (
    <form.Field name={name}>
      {(field: any) => (
        <Select
          className={cn("w-full group", className)}
          value={
            field.state.value !== undefined && field.state.value !== null
              ? String(field.state.value)
              : undefined
          }
          onChange={(key) => {
            const val = key as string;
            const isNumber =
              items.length > 0 && typeof items[0]?.value === "number";
            field.handleChange(
              (isNumber ? Number(val) : val) as DeepValue<TFormData, TName>,
            );
          }}
          isInvalid={field.state.meta.errors.length > 0}
          validationBehavior="aria"
          {...props}
        >
          {label && (
            <Label className="text-[10px] font-black uppercase tracking-widest text-muted mb-1.5 block">
              {label}
            </Label>
          )}
          {description && (
            <p className="text-[10px] text-muted mb-2 -mt-1 font-medium leading-relaxed">
              {description}
            </p>
          )}
          <Select.Trigger className="h-11 px-3 bg-default/10 border-none rounded-2xl outline-none data-[focus=true]:ring-2 data-[focus=true]:ring-accent/50 transition-all w-full flex items-center justify-between">
            <Select.Value className="text-xs font-mono" />
            <Select.Indicator className="text-muted">
              <IconChevronDown className="w-4 h-4" />
            </Select.Indicator>
          </Select.Trigger>
          <Select.Popover className="min-w-40 p-2 bg-background border border-border rounded-2xl shadow-xl">
            <ListBox>
              {items.map((opt) => (
                <ListBox.Item
                  key={opt.value}
                  id={String(opt.value)}
                  textValue={opt.label}
                  className="rounded-xl py-2 px-3 data-[selected=true]:bg-accent/10 data-[selected=true]:text-accent font-bold text-sm cursor-pointer"
                >
                  {opt.label}
                </ListBox.Item>
              ))}
            </ListBox>
          </Select.Popover>
          {field.state.meta.isTouched && !field.state.meta.isValid && (
            <FieldError className="absolute -bottom-5 right-0 text-[10px] text-danger font-bold uppercase tracking-tight animate-in fade-in slide-in-from-top-1">
              {field.state.meta.errors[0]?.message || field.state.meta.errors[0]}
            </FieldError>
          )}
        </Select>
      )}
    </form.Field>
  );
}
