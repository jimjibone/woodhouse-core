<script lang="ts">
	import { type StandardProps } from './service-root.svelte';
	import SingleValue from './single-value.svelte';
	import { DropletsIcon } from '@lucide/svelte';
	import { Humidity } from '$lib/schema/zz_services';

	let props: StandardProps = $props();
</script>

<SingleValue {...props} attrID={Humidity.attributes.humidity} icon={DropletsIcon} />
