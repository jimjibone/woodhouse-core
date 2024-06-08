<script lang="ts">
	import { Attribute, BoolValue, Permissions, Value } from '$lib/api/v1/clients/client_service_pb';
	// import { getDeviceInfo, getDeviceName } from '$lib/apitools';

	import * as Card from '$lib/components/ui/card';
	import AttributeBool from './AttributeBool.svelte';
	import AttributeFloat from './AttributeFloat.svelte';
	import AttributeInt from './AttributeInt.svelte';
	import AttributeColor from './AttributeColor.svelte';

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

<div>
	{#if attr.bool !== undefined}
		<AttributeBool id={attr.id} attr={attr.bool} onAction={action} disabled={!online}/>
	{:else if attr.int !== undefined}
		<AttributeInt id={attr.id} attr={attr.int} onAction={action} disabled={!online}/>
	{:else if attr.float !== undefined}
		<AttributeFloat id={attr.id} attr={attr.float} onAction={action} disabled={!online}/>
	{:else if attr.color !== undefined}
		<AttributeColor id={attr.id} attr={attr.color} onAction={action} disabled={!online}/>
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
</div>
