<script lang="ts">
	import { type StandardProps } from './service-root.svelte';
	import SingleValue from './single-value.svelte';
	import { CpuIcon } from '@lucide/svelte';
	import { DeviceTemperature } from '$lib/schema/zz_services';

	let props: StandardProps = $props();
</script>

<SingleValue {...props} attrID={DeviceTemperature.attributes.temperature} label="Device Temperature" icon={CpuIcon} />
