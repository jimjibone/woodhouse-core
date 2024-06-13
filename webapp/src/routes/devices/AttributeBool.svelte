<script lang="ts">
	import {
		BoolAttribute,
		BoolValue,
		Permissions,
		Value
	} from '$lib/api/v1/clients/client_service_pb';
	import { Switch } from '$lib/components/ui/switch/index.js';

	export let disabled: boolean;
	export let id: string;
	export let attr: BoolAttribute;
	export let onAction: (vals: Value[]) => Promise<void> | undefined;

	let action = async (val: boolean) => {
		if (onAction) {
			onAction([
				new Value({
					id: id,
					bool: new BoolValue({
						value: val
					})
				})
			]);
		}
	};
</script>

{#if disabled || attr.perms === Permissions.PERM_READONLY || attr.perms === Permissions.PERM_UNDEFINED}
	<Switch checked={attr.value} aria-readonly disabled />
{:else if attr.perms === Permissions.PERM_READWRITE}
	<Switch checked={attr.value} on:click={() => action(!attr.value)} />
{:else if attr.perms === Permissions.PERM_WRITEONLY}
	<p>WO: {attr.value ? 'true' : 'false'}</p>
{:else}
	<p>UNKNOWN {attr.value ? 'true' : 'false'}</p>
{/if}
