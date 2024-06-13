<script lang="ts">
	import {
		FloatAttribute,
		FloatValue,
		Permissions,
		Value,
		Unit
	} from '$lib/api/v1/clients/client_service_pb';
	import { Input } from '$lib/components/ui/input/index.js';

	export let disabled: boolean;
	export let id: string;
	export let attr: FloatAttribute;
	export let onAction: (vals: Value[]) => Promise<void> | undefined;

	let editing = false;
	let editValue: number = attr.value;
	$: editInvalid = String(editValue) === '';

	$: {
		if (!editing) {
			editValue = attr.value;
		}
	}

	let startEditing = () => {
		editing = true;
	};

	let stopEditing = () => {
		editing = false;
		editValue = attr.value;
	};

	let submit = async (event: Event) => {
		event.preventDefault();
		action(editValue);
	};

	let keypress = (ev: Event) => {
		const event = ev as KeyboardEvent;
		if (event.code === 'Escape') {
			stopEditing();
		} else {
			startEditing();
		}
	};

	let action = async (val: number) => {
		if (onAction) {
			onAction([
				new Value({
					id: id,
					float: new FloatValue({
						value: val
					})
				})
			]);
		}
	};
</script>

{#if disabled || attr.perms === Permissions.PERM_READONLY || attr.perms === Permissions.PERM_UNDEFINED}
	{#if attr.unit === Unit.PERCENTAGE}
		<p>{attr.value} %</p>
	{:else if attr.unit === Unit.ARC_DEGREES}
		<p>{attr.value}°</p>
	{:else if attr.unit === Unit.CELSIUS}
		<p>{attr.value} °C</p>
	{:else if attr.unit === Unit.LUX}
		<p>{attr.value} lux</p>
	{:else if attr.unit === Unit.SECONDS}
		<p>{attr.value} seconds</p>
	{:else if attr.unit === Unit.PPM}
		<p>{attr.value} ppm</p>
	{:else if attr.unit === Unit.MICROGRAMS_PER_CUBIC_METER}
		<p>{attr.value} μg/m3</p>
	{:else if attr.unit === Unit.VOLTS}
		<p>{attr.value} V</p>
	{:else if attr.unit === Unit.AMPS}
		<p>{attr.value} A</p>
	{:else if attr.unit === Unit.WATTS}
		<p>{attr.value} W</p>
	{:else}
		<p>{attr.value}</p>
	{/if}
{:else if attr.perms === Permissions.PERM_READWRITE}
	<!-- <p>editing: {editing}, editValue: {editValue}, invalid: {editInvalid}</p> -->
	<form class="flex w-full max-w-sm items-center space-x-2" on:submit={submit}>
		<Input
			type="number"
			invalid={editInvalid}
			min={attr.min}
			max={attr.max}
			placeholder={attr.value}
			bind:value={editValue}
			on:focusin={startEditing}
			on:focusout={stopEditing}
			on:keypress={keypress}
		/>
	</form>
{:else if attr.perms === Permissions.PERM_WRITEONLY}
	<p>WO: {attr.value}</p>
{:else}
	<p>UNKNOWN {attr.value}</p>
{/if}
