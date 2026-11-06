<script lang="ts">
	import { Button } from '$lib/components/ui/button';

	let {
		name,
		value,
		onaction,
		class: className = ''
	}: {
		name: string;
		value: boolean | undefined;
		onaction: (value: boolean) => void;
		class?: string;
	} = $props();
</script>

<div class={className}>{name}</div>
<div class="col-span-2 flex flex-row gap-0">
	<Button
		class={'cursor-pointer rounded-r-none w-16 ' + className}
		variant={value ? 'light' : 'dark'}
		onclick={() => onaction(true)}>On</Button
	>
	<Button
		class={'cursor-pointer rounded-l-none w-16 ' + className}
		variant={value ? 'light' : 'dark'}
		onclick={() => onaction(false)}>Off</Button
	>
</div>
