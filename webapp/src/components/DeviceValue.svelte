<script lang="ts">
	import uid from '../internal/uid';
	import type { DeviceValue } from '../api/device_pb';
	import BoolValue from './values/BoolValue.svelte';
	import NumberValue from './values/NumberValue.svelte';
	import { Label } from "$lib/components/ui/label";
	import { Input } from "$lib/components/ui/input";

	export let id: string;
	export let value: DeviceValue;
	export let writable: boolean = false;
	export let writer: ((value: DeviceValue) => void) | undefined;

	id = id || `wh-${uid(5)}`;

	function onRequest(v: DeviceValue) : void {
		if (writer) {
			v.setName(value.getName());
			writer(v);
		}
	}
</script>

<div class="flex flex-col  gap-1.5">
	<Label for="{id}">{value.getName()}</Label>
	{#if value.hasBool()}
		<BoolValue value={value.getBool()} writable={writable} writer={onRequest} />
	{:else if value.hasNumber()}
		<NumberValue value={value.getNumber()} writable={writable} writer={onRequest} />
	{:else if value.hasText()}
		<Input type="text" id="{id}" disabled value={value.getText().getValue()} />
	{:else if value.hasColor()}
		<p>Color: Hue: {value.getColor().getHue()}, Sat: {value.getColor().getSat()}</p>
	{/if}
</div>
