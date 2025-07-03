<script lang="ts">
	import type { BoolAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { Button } from "$lib/components/ui/button";

	let {
		name,
		attr,
		onaction
	}: {
		name: string,
		attr: BoolAttribute,
		onaction: (value: boolean)=>void
	} = $props();
</script>

<div>{name}</div>
<div class="col-span-2 flex flex-row gap-0">
	<Button class="cursor-pointer rounded-r-none w-16" variant={attr.value ? "light":"dark"} onclick={() => onaction(true)}>On</Button>
	<Button class="cursor-pointer rounded-l-none w-16" variant={attr.value ? "light":"dark"} onclick={() => onaction(false)}>Off</Button>
</div>
