<script lang="ts">
	import { Attribute, BoolValue, Permissions, Value } from '$lib/api/v1/clients/client_service_pb';
	// import { getDeviceInfo, getDeviceName } from '$lib/apitools';

	import * as Card from '$lib/components/ui/card';
	import Button from '@/components/ui/button/button.svelte';
	import { Label } from "$lib/components/ui/label/index.js";
	import { Switch } from "$lib/components/ui/switch/index.js";
	import AttributeBool from './AttributeBool.svelte';

	export let online: boolean;
	export let attr: Attribute;
	export let onAction: (val: Value) => Promise<void> | undefined

	let action = async (val: Value) => {
		if (onAction) {
			onAction(val);
		}
	}

	// $:info = getDeviceInfo(device);
</script>

<Card.Root class={online ? "" : "bg-muted"}>
	<Card.Header class="pb-3">
		<Card.Title>{attr.id}</Card.Title>
	</Card.Header>
	<Card.Content>
		{#if attr.bool !== undefined}
			<AttributeBool id={attr.id} attr={attr.bool} onAction={action} />
		{:else if attr.int !== undefined}
			{#if attr.int.perms === Permissions.PERM_READWRITE}
				<p>RW: {attr.int.value}</p>
			{:else if attr.int.perms === Permissions.PERM_WRITEONLY}
				<p>WO: {attr.int.value}</p>
			{:else} <!-- readonly, undefined -->
				<p>RO: {attr.int.value}</p>
			{/if}
		{:else if attr.float !== undefined}
			{#if attr.float.perms === Permissions.PERM_READWRITE}
				<p>RW: {attr.float.value}</p>
			{:else if attr.float.perms === Permissions.PERM_WRITEONLY}
				<p>WO: {attr.float.value}</p>
			{:else} <!-- readonly, undefined -->
				<p>RO: {attr.float.value}</p>
			{/if}
		{:else if attr.text !== undefined}
			{#if attr.text.perms === Permissions.PERM_READWRITE}
				<p>RW: {attr.text.value}</p>
			{:else if attr.text.perms === Permissions.PERM_WRITEONLY}
				<p>WO: {attr.text.value}</p>
			{:else} <!-- readonly, undefined -->
				<p>RO: {attr.text.value}</p>
			{/if}
		{:else if attr.duration !== undefined}
			{#if attr.duration.perms === Permissions.PERM_READWRITE}
				<p>RW: {attr.duration.value}ms</p>
			{:else if attr.duration.perms === Permissions.PERM_WRITEONLY}
				<p>WO: {attr.duration.value}ms</p>
			{:else} <!-- readonly, undefined -->
				<p>RO: {attr.duration.value}ms</p>
			{/if}
		{:else if attr.time !== undefined}
			{#if attr.time.perms === Permissions.PERM_READWRITE}
				<p>RW: {attr.time.seconds}s {attr.time.nanos}ns</p>
			{:else if attr.time.perms === Permissions.PERM_WRITEONLY}
				<p>WO: {attr.time.seconds}s {attr.time.nanos}ns</p>
			{:else} <!-- readonly, undefined -->
				<p>RO: {attr.time.seconds}s {attr.time.nanos}ns</p>
			{/if}
		{:else}
			<p>Unsupported type: {attr.toJsonString()}</p>
		{/if}
	</Card.Content>
</Card.Root>
