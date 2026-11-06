<script lang="ts">
	import { type StandardProps } from './service-root.svelte';
	import SingleValue from './single-value.svelte';
	import { ActivityIcon } from '@lucide/svelte';
	import { AnalogInput } from '$lib/schema/zz_services';

	let props: StandardProps = $props();
</script>

<!-- The reading in percent. The mapped value and range_map are shown by the
	 generic attribute rendering. -->
<SingleValue {...props} attrID={AnalogInput.attributes.value} icon={ActivityIcon} />
