<script lang="ts">
	import { IntAttribute, IntValue, Permissions, Value } from '$lib/api/v1/clients/client_service_pb';
	import { Input } from "$lib/components/ui/input/index.js";

	export let disabled: boolean;
	export let id: string;
	export let attr: IntAttribute;
	export let onAction: (val: Value) => Promise<void> | undefined

	let editing = false;
	let editValue: bigint = attr.value;
	$: editInvalid = String(editValue) === "";

	$: {
		if (!editing) {
			editValue = attr.value;
		}
	}

	let startEditing = () => {
		editing = true;
	}

	let stopEditing = () => {
		editing = false;
		editValue = attr.value;
	}

	let submit = async (event: Event) => {
		event.preventDefault();
		action(editValue);
	}

	let keypress = (ev: Event) => {
		const event = ev as KeyboardEvent;
		if (event.code === "Escape") {
			stopEditing();
		} else {
			startEditing();
		}
	}

	let action = async (val: bigint) => {
		if (onAction) {
			onAction(
				new Value({
					id: id,
					int: new IntValue({
						value: val
					})
				})
			);
		}
	}
</script>

{#if disabled || attr.perms === Permissions.PERM_READONLY || attr.perms === Permissions.PERM_UNDEFINED}
	<p>{attr.value}</p>
{:else if attr.perms === Permissions.PERM_READWRITE}
	<!-- <p>editing: {editing}, editValue: {editValue}, invalid: {editInvalid}</p> -->
	<form class="flex w-full max-w-sm items-center space-x-2" on:submit={submit}>
		<Input type="number" invalid={editInvalid} min={attr.min} max={attr.max} placeholder={attr.value} bind:value={editValue} on:focusin={startEditing} on:focusout={stopEditing} on:keypress={keypress}/>
	</form>
{:else if attr.perms === Permissions.PERM_WRITEONLY}
	<p>WO: {attr.value}</p>
{:else}
	<p>UNKNOWN {attr.value}</p>
{/if}
