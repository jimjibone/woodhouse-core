<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input/index.js';

	// A number entry for a writable int, float or duration attribute that has
	// no min and max to build a slider from.
	let {
		name,
		value,
		min,
		max,
		units = '',
		onaction,
		class: className = ''
	}: {
		name: string;
		value: number | undefined;
		min?: number;
		max?: number;
		units?: string;
		onaction: (value: number) => void;
		class?: string;
	} = $props();

	let draft = $state('');
	let editing = $state(false);

	$effect(() => {
		const incoming = value === undefined ? '' : String(value);
		if (!editing) {
			draft = incoming;
		}
	});

	const parsed = $derived(draft.trim() === '' ? undefined : Number(draft));
	const valid = $derived(parsed !== undefined && Number.isFinite(parsed));

	function send() {
		if (!valid || parsed === undefined) return;
		editing = false;
		onaction(parsed);
	}

	function onkeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			send();
		}
	}
</script>

<div class={className}>{name}</div>
<div class="col-span-2 flex flex-row items-center gap-2">
	<Input
		type="text"
		inputmode="decimal"
		bind:value={draft}
		placeholder={min !== undefined || max !== undefined ? `${min ?? ''} to ${max ?? ''}` : ''}
		{onkeydown}
		onfocus={() => (editing = true)}
		onblur={() => (editing = false)}
		class={className}
	/>
	{#if units}
		<span class="text-muted-foreground">{units}</span>
	{/if}
	<Button class={'cursor-pointer ' + className} variant="dark" disabled={!valid} onclick={send}>Send</Button>
</div>
