<script lang="ts">
	import type { DeviceValue } from '../api/device_pb';

	export let value: DeviceValue = null;
</script>

{#if value.hasBool()}
<p>{value.getName()} Bool: {value.getBool() ? "true" : "false"}</p>
{:else if value.hasNumber()}
<p>{value.getName()} Number: {value.getNumber()}</p>
{:else if value.hasText()}
<p>{value.getName()} Text: {value.getText()}</p>
{/if}
