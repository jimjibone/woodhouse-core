<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input/index.js';

	let {
		name,
		value,
		onaction,
		class: className = ''
	}: {
		name: string;
		value: string | undefined;
		onaction: (value: string) => void;
		class?: string;
	} = $props();

	let draft = $state('');
	let editing = $state(false);

	// Follow the device's value while the user is not mid-edit, so an update
	// from elsewhere shows up without overwriting something being typed.
	$effect(() => {
		const incoming = value ?? '';
		if (!editing) {
			draft = incoming;
		}
	});

	function send() {
		editing = false;
		onaction(draft);
	}

	function onkeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			send();
		}
	}
</script>

<div class={className}>{name}</div>
<div class="col-span-2 flex flex-row gap-2">
	<Input
		type="text"
		bind:value={draft}
		{onkeydown}
		onfocus={() => (editing = true)}
		onblur={() => (editing = false)}
		class={className}
	/>
	<Button class={'cursor-pointer ' + className} variant="dark" onclick={send}>Send</Button>
</div>
