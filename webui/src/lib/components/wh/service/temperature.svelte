<script lang="ts">
	import { type StandardProps } from './service-root.svelte';
	import SingleValue from './single-value.svelte';
	import { ThermometerIcon } from '@lucide/svelte';
	import { Temperature } from '$lib/schema/zz_services';

	let props: StandardProps = $props();
</script>

<SingleValue {...props} attrID={Temperature.attributes.temperature} icon={ThermometerIcon} />
