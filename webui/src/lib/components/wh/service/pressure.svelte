<script lang="ts">
	import { type StandardProps } from './service-root.svelte';
	import SingleValue from './single-value.svelte';
	import { GaugeIcon } from '@lucide/svelte';
	import { Pressure } from '$lib/schema/zz_services';

	let props: StandardProps = $props();
</script>

<SingleValue {...props} attrID={Pressure.attributes.pressure} icon={GaugeIcon} />
