import { Label, TextField, Select, ListBox, InputGroup, FieldError, Switch, type TextFieldProps, type SelectRootProps, type SwitchProps, cn } from "@heroui/react";
import type { DeepKeys, DeepValue } from "@tanstack/react-form";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import React from "react";

// Simplified form API interface to avoid library-specific type mismatch errors
interface AppFormApi {
  Field: any;
  handleSubmit: () => void;
  setFieldValue: (name: any, value: any) => void;
  reset: () => void;
}

// Generic FormSwitch component
interface FormSwitchProps<TFormData, TName extends DeepKeys<TFormData>> extends Omit<SwitchProps, "name" | "form" | "children" | "onChange" | "isSelected" | "defaultSelected"> {
  form: AppFormApi;
  name: TName;
  label?: React.ReactNode;
  description?: string;
  validators?: {
    onChange?: (params: { value: DeepValue<TFormData, TName> }) => string | undefined;
  };
}

export function FormSwitch<TFormData, TName extends DeepKeys<TFormData>>({
  form,
  name,
  label,
  description,
  validators,
  className,
  ...props
}: FormSwitchProps<TFormData, TName>) {
  return (
    <form.Field
      name={name}
      validators={validators}
    >
      {(field: any) => (
        <div className={cn("flex items-center justify-between", className)}>
          <div className="flex flex-col gap-0.5">
            {label && (
              <Label className="text-sm font-bold tracking-tight">
                {label}
              </Label>
            )}
            {description && (
              <p className="text-xs text-muted">
                {description}
              </p>
            )}
          </div>
          <Switch
            isSelected={!!field.state.value}
            onChange={(isSelected) => field.handleChange(isSelected as DeepValue<TFormData, TName>)}
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

// Generic FormTextField component
interface FormTextFieldProps<TFormData, TName extends DeepKeys<TFormData>> extends Omit<TextFieldProps, "name" | "form" | "children" | "onChange" | "value" | "defaultValue"> {
  form: AppFormApi;
  name: TName;
  label?: React.ReactNode;
  placeholder?: string;
  description?: string;
  validators?: {
    onChange?: (params: { value: DeepValue<TFormData, TName> }) => string | undefined;
  };
  // transform for display (e.g. bytes -> MB)
  format?: (value: DeepValue<TFormData, TName>) => string;
  // transform for storage (e.g. MB -> bytes)
  parse?: (value: string) => DeepValue<TFormData, TName>;
  startContent?: React.ReactNode;
  endContent?: React.ReactNode;
}

export function FormTextField<TFormData, TName extends DeepKeys<TFormData>>({
  form,
  name,
  label,
  validators,
  className,
  placeholder,
  format,
  parse,
  startContent,
  endContent,
  ...props
}: FormTextFieldProps<TFormData, TName>) {
  return (
    <form.Field
      name={name}
      validators={validators}
    >
      {(field: any) => {
        // Handle value conversion
        const displayValue = format ? format(field.state.value) : String(field.state.value ?? "");
        
        return (
          <TextField
            className={cn("w-full group", className)}
            isInvalid={field.state.meta.errors.length > 0}
            validationBehavior="aria"
            {...props}
          >
            {label && (
              <Label className="text-sm font-bold mb-1.5 block text-foreground/80">
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
                <InputGroup.Prefix>
                  {startContent}
                </InputGroup.Prefix>
              )}
              <InputGroup.Input
                value={displayValue}
                onChange={(e) => {
                  const val = e.target.value;
                  field.handleChange(parse ? parse(val) : val as DeepValue<TFormData, TName>);
                }}
                onBlur={field.handleBlur}
                placeholder={placeholder}
                className="h-11 bg-default/10 rounded-2xl border-none font-mono text-xs w-full"
              />
              {endContent && (
                <InputGroup.Suffix>
                  {endContent}
                </InputGroup.Suffix>
              )}
              <FieldError className="absolute -bottom-5 right-0 text-[10px] text-danger font-bold uppercase tracking-tight animate-in fade-in slide-in-from-top-1" />
            </InputGroup>
          </TextField>
        );
      }}
    </form.Field>
  );
}

// Generic FormSelect component
interface FormSelectOption {
  value: string | number;
  label: string;
}

interface FormSelectProps<TFormData, TName extends DeepKeys<TFormData>> extends Omit<SelectRootProps<any>, "name" | "form" | "children" | "onChange" | "selectedKey" | "onSelectionChange" | "items"> {
  form: AppFormApi;
  name: TName;
  label?: React.ReactNode;
  items: FormSelectOption[];
  validators?: {
    onChange?: (params: { value: DeepValue<TFormData, TName> }) => string | undefined;
  };
}

export function FormSelect<TFormData, TName extends DeepKeys<TFormData>>({
  form,
  name,
  label,
  items,
  validators,
  className,
  ...props
}: FormSelectProps<TFormData, TName>) {
  return (
    <form.Field
      name={name}
      validators={validators}
    >
      {(field: any) => (
        <Select
          className={cn("w-full", className)}
          selectedKey={String(field.state.value)}
          onSelectionChange={(key) => {
            const val = key as string;
            // Infer type from the first item value if possible, default to string
            const isNumber = typeof items[0]?.value === 'number';
            field.handleChange((isNumber ? Number(val) : val) as DeepValue<TFormData, TName>);
          }}
          isInvalid={field.state.meta.errors.length > 0}
          validationBehavior="aria"
          {...props}
        >
          {label && (
            <Label className="text-[10px] font-black uppercase tracking-widest text-muted ml-1 mb-1.5 block">
              {label}
            </Label>
          )}
          <Select.Trigger className="h-[38px] px-3 bg-default/10 border border-border rounded-xl outline-none focus:ring-2 focus:ring-accent/50 transition-all min-w-[160px]">
            <Select.Value className="text-sm font-bold" />
            <Select.Indicator className="text-muted">
              <IconChevronDown className="w-4 h-4" />
            </Select.Indicator>
          </Select.Trigger>
          <Select.Popover className="min-w-[160px] p-2 bg-background border border-border rounded-2xl shadow-xl">
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
          <FieldError className="absolute -bottom-5 right-0 text-[10px] text-danger font-bold uppercase tracking-tight animate-in fade-in slide-in-from-top-1" />
        </Select>
      )}
    </form.Field>
  );
}