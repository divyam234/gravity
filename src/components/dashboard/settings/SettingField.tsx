import {
	FieldError,
	Input,
	Label,
	ListBox,
	Select,
	Switch,
	TextField,
} from "@heroui/react";
import React from "react";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import { cn } from "../../../lib/utils";

export interface OptionMetadata {
	name: string;
	type: string;
	defaultValue?: string;
	description?: string;
	options?: string[];
	min?: number;
	max?: number;
	pattern?: string;
	readonly?: boolean;
}

export const SettingField: React.FC<{
	opt: OptionMetadata;
	value: string;
	onUpdate: (name: string, value: string) => void;
	isReadOnly?: boolean;
}> = React.memo(({ opt, value, onUpdate, isReadOnly }) => {
	const isBoolean = opt.type === "boolean";
	const isOption = opt.type === "option";
	const isEnum = isOption || (opt.type.includes("|") && !isBoolean);

	const enumValues = isOption
		? opt.options || []
		: isEnum
			? opt.type.split("|")
			: [];

	// Validation logic
	const validate = (val: string) => {
		if (isReadOnly) return true;

		// Integer validation
		if (opt.type === "integer") {
			if (
				val !== "" &&
				(isNaN(Number(val)) || !Number.isInteger(Number(val)))
			) {
				return "Must be a valid integer";
			}
			const num = Number(val);
			if (opt.min !== undefined && num < opt.min)
				return `Minimum value is ${opt.min}`;
			if (opt.max !== undefined && num > opt.max)
				return `Maximum value is ${opt.max}`;
		}

		// Float validation
		if (opt.type === "float") {
			if (val !== "" && isNaN(Number(val))) {
				return "Must be a valid number";
			}
			const num = Number(val);
			if (opt.min !== undefined && num < opt.min)
				return `Minimum value is ${opt.min}`;
			if (opt.max !== undefined && num > opt.max)
				return `Maximum value is ${opt.max}`;
		}

		// Pattern validation
		if (opt.pattern) {
			const regex = new RegExp(opt.pattern);
			if (val !== "" && !regex.test(val)) {
				// Customize error messages for common patterns if possible
				if (opt.pattern.includes("K|k|M|m")) {
					return "Invalid format (e.g. 5M, 10K, 0)";
				}
				return "Invalid format";
			}
		}

		return true;
	};

	return (
		<div
			className={cn(
				"flex flex-col gap-3 py-6 border-b border-border/60 last:border-0",
				isReadOnly && "opacity-60 grayscale pointer-events-none",
			)}
		>
			<div className="flex items-start justify-between gap-6">
				<div className="flex flex-col flex-1">
					<span className="font-mono text-sm font-bold text-accent tracking-tight">
						{opt.name}
					</span>
					<p
						className="text-xs text-muted leading-relaxed line-clamp-2 mt-1"
						title={opt.description}
					>
						{opt.description}
					</p>
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
							className="max-w-[240px]"
							selectedKey={value || opt.defaultValue || enumValues[0]}
							onSelectionChange={(key) => onUpdate(opt.name, String(key))}
							isDisabled={isReadOnly}
						>
							<Select.Trigger className="h-10 px-4 bg-default/10 rounded-xl hover:bg-default/20 transition-colors border-none outline-none">
								<Select.Value className="text-sm font-medium" />
								<Select.Indicator className="text-muted">
									<IconChevronDown className="w-4 h-4" />
								</Select.Indicator>
							</Select.Trigger>
							<Select.Popover className="min-w-[240px] p-2 bg-background border border-border rounded-2xl shadow-xl">
								<ListBox items={enumValues.map((v) => ({ id: v, name: v }))}>
									{(item) => (
										<ListBox.Item
											id={item.id}
											textValue={item.name}
											className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 text-sm cursor-pointer outline-none"
										>
											<Label>{item.name}</Label>
										</ListBox.Item>
									)}
								</ListBox>
							</Select.Popover>
						</Select>
					) : (
						<TextField
							className="max-w-[240px] w-full"
							defaultValue={value}
							onChange={(val) => {
								// If the user clears the input, we can allow it temporarily or set to empty
								// but if we want to enforce validation on change, we check here.
								// For better UX, we might want to allow typing and only show error,
								// but the current pattern updates on change if valid.
								// We'll keep the pattern but loosen it slightly to allow empty strings if not required,
								// but since validation logic returns error strings, we need to handle that.
								// The TextField 'validate' prop handles the visual error state.

								// We always update the value to allow typing, but the parent might reject it?
								// The current implementation only calls onUpdate if valid.
								// This prevents typing invalid chars if they immediately invalidate.
								// Let's stick to the existing pattern: update if valid.
								// But for patterns like "10M", "1" is invalid? No.
								// "10" is valid. "10M" is valid.
								// "10X" is invalid.

								// However, for UX, it's often better to update local state and show error.
								// But preserving the existing "update on valid only" logic:
								if (validate(val) === true || val === "") {
									onUpdate(opt.name, val);
								}
							}}
							validate={validate}
							validationBehavior="aria"
							isReadOnly={isReadOnly}
						>
							<div className="relative group">
								<Input className="w-full h-10 px-4 bg-default/10 rounded-xl text-sm border border-transparent focus:bg-default/20 focus:border-accent/30 transition-all outline-none data-[invalid=true]:border-danger/50" />
								<FieldError className="absolute -bottom-5 right-0 text-[10px] text-danger font-bold uppercase tracking-tight animate-in fade-in slide-in-from-top-1" />
							</div>
						</TextField>
					)}
				</div>
			</div>
		</div>
	);
});
