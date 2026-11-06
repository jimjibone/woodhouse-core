<script lang="ts">
	// Every attribute of a service not in `exclude`, rendered from its spec,
	// followed by the service's commands. Typed service components render the
	// attributes they know how to lay out themselves and hand the rest to this.
	import type { Service } from '$lib/api/v2/clients/client_service_pb';
	import type ServiceAction from '$lib/components/wh/service/service-action.svelte';
	import { isWritable } from '$lib/device';
	import AttributeRow from './attribute-row.svelte';
	import Commands from './commands.svelte';

	let {
		deviceID,
		endpointID,
		service,
		action,
		exclude = [],
		excludeCommands = [],
		separator = true
	}: {
		deviceID: string;
		endpointID: string;
		service: Service;
		action: ServiceAction;
		/** Attribute IDs the caller renders itself. */
		exclude?: string[];
		/** Command IDs the caller renders itself. */
		excludeCommands?: string[];
		/** Draw a rule above the attributes. */
		separator?: boolean;
	} = $props();

	// Attributes arrive in no particular order, so sort them to keep the
	// display stable: writable controls first, then read-only values, each
	// group ordered by ID.
	const others = $derived(
		service.attrs
			.filter((attr) => !exclude.includes(attr.id))
			.sort((a, b) => {
				const aw = isWritable(a) ? 0 : 1;
				const bw = isWritable(b) ? 0 : 1;
				if (aw !== bw) return aw - bw;
				return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
			})
	);
</script>

{#if others.length > 0}
	<div class={'grid grid-cols-[auto_1fr_auto] gap-4 items-center ' + (separator ? 'pt-4' : '')}>
		{#if separator}
			<div class="col-span-3 text-muted-foreground border-t-2 font-semibold"></div>
		{/if}
		{#each others as attr (attr.id)}
			<AttributeRow {attr} {deviceID} {endpointID} serviceID={service.id} {action} />
		{/each}
	</div>
{/if}
<Commands {service} {action} exclude={excludeCommands} />
