<script lang="ts">
	import uid from '../internal/uid';
	import type { DeviceValue } from '../api/device_pb';
	import BoolValue from './values/BoolValue.svelte';
	import NumberValue from './values/NumberValue.svelte';

	export let id = null;
	export let value: DeviceValue = null;
	export let writable: boolean = false;
	export let writer: (value: DeviceValue) => void = null;

	id = id || `wh-${uid(5)}`;

	function onRequest(v: DeviceValue) : void {
		if (writer) {
			v.setName(value.getName());
			writer(v);
		}
	}
</script>

<div class="field">
	<label class="label text-slate-500 dark:text-slate-400" for="{id}">{value.getName()}</label>
	<div class="control">
		{#if value.hasBool()}
		<BoolValue value={value.getBool()} writable={writable} writer={onRequest} />
		{:else if value.hasNumber()}
		<NumberValue value={value.getNumber()} writable={writable} writer={onRequest} />
		{:else if value.hasText()}
		<p>Text: {value.getText().getValue()}</p>
		{:else if value.hasColor()}
		<p>Color: Hue: {value.getColor().getHue()}, Sat: {value.getColor().getSat()}</p>
		{/if}
	</div>
</div>
