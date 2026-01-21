import { Input, Label, TextField, Select, ListBox, InputGroup, FieldError, type TextFieldProps, type SelectProps, cn } from "@heroui/react";
import type { FormApi, Validator } from "@tanstack/react-form";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import React from "react";

// Generic FormTextField component
interface FormTextFieldProps<TData, TName> extends Omit<TextFieldProps, "name" | "form" | "children" | "onChange" | "value" | "defaultValue"> {
  form: FormApi<TData, any, any, any>;
  name: TName;
  label?: React.ReactNode;
  placeholder?: string;
  validators?: {
    onChange?: Validator<any, any>;
    onBlur?: Validator<any, any>;
  };
  // transform for display (e.g. bytes -> MB)
  format?: (value: any) => string;
  // transform for storage (e.g. MB -> bytes)
  parse?: (value: string) => any;
  startContent?: React.ReactNode;
  endContent?: React.ReactNode;
}

export function FormTextField<TData, TName>({
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
}: FormTextFieldProps<TData, TName>) {
  return (
    <form.Field
      name={name as any}
      validators={validators}
    >
      {(field) => {
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
                  field.handleChange(parse ? parse(val) : val as any);
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

interface FormSelectProps<TData, TName> extends Omit<SelectProps, "name" | "form" | "children" | "onChange" | "selectedKey" | "onSelectionChange" | "items"> {
  form: FormApi<TData, any, any, any>;
  name: TName;
  label?: React.ReactNode;
  items: FormSelectOption[];
  validators?: {
    onChange?: Validator<any, any>;
    onBlur?: Validator<any, any>;
  };
}

export function FormSelect<TData, TName>({
  form,
  name,
  label,
  items,
  validators,
  className,
  ...props
}: FormSelectProps<TData, TName>) {
  return (
    <form.Field
      name={name as any}
      validators={validators}
    >
      {(field) => (
        <Select
          className={cn("w-full", className)}
          selectedKey={String(field.state.value)}
          onSelectionChange={(key) => {
            const val = key as string;
            // Infer type from the first item value if possible, default to string
            const isNumber = typeof items[0]?.value === 'number';
            field.handleChange(isNumber ? Number(val) : val as any);
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
