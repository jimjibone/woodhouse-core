<script lang="ts">
	// An input for a value of any Spec kind, used to fill in command
	// parameters. The initial value is read once; after that the component
	// holds the draft and reports each change through onchange (undefined
	// while the input is empty or does not parse).
	import type { Spec, Value } from '$lib/api/v2/clients/client_service_pb';
	import {
		asBool,
		asColor,
		asDuration,
		asEnum,
		asList,
		asNumber,
		asStruct,
		asText,
		asTime,
		boolValue,
		colorHueSatValue,
		colorXYValue,
		durationValue,
		enumValue,
		field,
		floatValue,
		intValue,
		listValue,
		specEnumOptions,
		specKind,
		specListItem,
		specRange,
		specStructFields,
		specUnit,
		structValue,
		textValue,
		timeValue
	} from '$lib/device';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import { unitLabel } from '$lib/tools/units';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Switch } from '$lib/components/ui/switch';
	import { PlusIcon, XIcon } from '@lucide/svelte';
	import Self from './field-input.svelte';

	let {
		spec,
		value = undefined,
		onchange,
		id = undefined
	}: {
		spec: Spec | undefined;
		value?: Value;
		onchange: (value: Value | undefined) => void;
		id?: string;
	} = $props();

	const kind = specKind(spec);
	const range = specRange(spec);
	const units = unitLabel(specUnit(spec));

	const selectClass =
		'border-input bg-background dark:bg-input/30 shadow-xs h-9 w-full min-w-0 rounded-md border px-3 py-1 text-base outline-none md:text-sm focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]';

	// Numbers (int, float, duration in seconds) are held as text so a partly
	// typed number is not rewritten under the cursor.
	const initialNumber = () => {
		const n = asNumber(value);
		if (n === undefined) return '';
		return kind === 'duration' ? String(n / 1000) : String(n);
	};
	let numText = $state(initialNumber());

	const onNumber = () => {
		const t = numText.trim();
		const n = Number(t);
		if (t === '' || !Number.isFinite(n)) {
			onchange(undefined);
		} else if (kind === 'int') {
			onchange(Number.isInteger(n) ? intValue(n) : undefined);
		} else if (kind === 'duration') {
			onchange(durationValue(n * 1000));
		} else {
			onchange(floatValue(n));
		}
	};

	let textDraft = $state(asText(value) ?? '');

	const toLocalInput = (d: Date | undefined) => {
		if (d === undefined) return '';
		const p = (n: number) => String(n).padStart(2, '0');
		return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
	};
	let timeText = $state(toLocalInput(asTime(value)));
	const onTime = () => {
		const d = new Date(timeText);
		onchange(timeText === '' || Number.isNaN(d.getTime()) ? undefined : timeValue(d));
	};

	let enumDraft = $state(asEnum(value) ?? '');

	// Color: hue/saturation when the spec accepts it, else x/y.
	const useHueSat = spec?.kind.case === 'color' ? spec.kind.value.hueSat || !spec.kind.value.xy : true;
	let c1 = $state(
		String((useHueSat ? asColor(value)?.hueSat?.hue : asColor(value)?.xy?.x) ?? '')
	);
	let c2 = $state(
		String((useHueSat ? asColor(value)?.hueSat?.sat : asColor(value)?.xy?.y) ?? '')
	);
	const onColor = () => {
		const a = Number(c1);
		const b = Number(c2);
		if (c1.trim() === '' || c2.trim() === '' || !Number.isFinite(a) || !Number.isFinite(b)) {
			onchange(undefined);
		} else {
			onchange(useHueSat ? colorHueSatValue(a, b) : colorXYValue(a, b));
		}
	};

	// List: one draft per item, with a stable key so removing an item does
	// not hand its draft to the next one.
	let nextKey = 0;
	type Item = { key: number; v: Value | undefined };
	let items = $state.raw<Item[]>((asList(value)?.items ?? []).map((v) => ({ key: nextKey++, v })));
	const emitList = () =>
		onchange(listValue(items.flatMap((i) => (i.v === undefined ? [] : [i.v]))));

	// Struct: one draft per field.
	const structFields = specStructFields(spec);
	let fields = $state.raw<Record<string, Value | undefined>>(
		Object.fromEntries((asStruct(value)?.fields ?? []).map((f) => [f.id, f.value]))
	);
	const emitStruct = () =>
		onchange(
			structValue(
				structFields.flatMap((f) => {
					const v = fields[f.id];
					return v === undefined ? [] : [field(f.id, v)];
				})
			)
		);
</script>

{#if kind === 'bool'}
	<Switch checked={asBool(value) ?? false} onCheckedChange={(c) => onchange(boolValue(c))} {id} />
{:else if kind === 'int' || kind === 'float' || kind === 'duration'}
	<div class="flex items-center gap-2">
		<Input
			type="text"
			inputmode="decimal"
			{id}
			bind:value={numText}
			oninput={onNumber}
			placeholder={range.min !== undefined || range.max !== undefined
				? `${kind === 'duration' && range.min !== undefined ? range.min / 1000 : (range.min ?? '')} to ${kind === 'duration' && range.max !== undefined ? range.max / 1000 : (range.max ?? '')}`
				: ''}
		/>
		{#if kind === 'duration'}
			<span class="text-muted-foreground text-sm">s</span>
		{:else if units}
			<span class="text-muted-foreground text-sm whitespace-pre">{units}</span>
		{/if}
	</div>
{:else if kind === 'text'}
	<Input
		type="text"
		{id}
		bind:value={textDraft}
		oninput={() => onchange(textValue(textDraft))}
	/>
{:else if kind === 'time'}
	<Input type="datetime-local" {id} bind:value={timeText} oninput={onTime} />
{:else if kind === 'enum'}
	<select
		class={selectClass}
		{id}
		bind:value={enumDraft}
		onchange={() => onchange(enumDraft === '' ? undefined : enumValue(enumDraft))}
	>
		<option value="">Select...</option>
		{#each specEnumOptions(spec) as option (option)}
			<option value={option}>{option}</option>
		{/each}
	</select>
{:else if kind === 'color'}
	<div class="grid grid-cols-2 gap-2">
		<Input type="text" inputmode="decimal" bind:value={c1} oninput={onColor} placeholder={useHueSat ? 'Hue (0-360)' : 'x'} />
		<Input type="text" inputmode="decimal" bind:value={c2} oninput={onColor} placeholder={useHueSat ? 'Saturation (0-100)' : 'y'} />
	</div>
{:else if kind === 'list'}
	<div class="grid gap-2">
		{#each items as item (item.key)}
			<div class="flex items-start gap-2">
				<div class="grow">
					<Self
						spec={specListItem(spec)}
						value={item.v}
						onchange={(v) => {
							items = items.map((it) => (it.key === item.key ? { key: it.key, v } : it));
							emitList();
						}}
					/>
				</div>
				<Button
					type="button"
					variant="ghost"
					size="icon-sm"
					class="cursor-pointer"
					aria-label="Remove item"
					onclick={() => {
						items = items.filter((it) => it.key !== item.key);
						emitList();
					}}
				>
					<XIcon />
				</Button>
			</div>
		{/each}
		<div>
			<Button
				type="button"
				variant="secondary"
				size="sm"
				class="cursor-pointer"
				onclick={() => {
					items = [...items, { key: nextKey++, v: undefined }];
				}}
			>
				<PlusIcon /> Add item
			</Button>
		</div>
	</div>
{:else if kind === 'struct'}
	<div class="grid gap-2 rounded-md border p-2">
		{#each structFields as f (f.id)}
			<div class="grid gap-1">
				<span class="text-sm text-muted-foreground">
					{toHeadlineCase(f.id)}{f.required ? ' *' : ''}
				</span>
				<Self
					spec={f.spec}
					value={fields[f.id]}
					onchange={(v) => {
						fields = { ...fields, [f.id]: v };
						emitStruct();
					}}
				/>
			</div>
		{/each}
	</div>
{:else}
	<span class="text-sm text-muted-foreground">This type cannot be entered here.</span>
{/if}
