<script lang="ts">
	import { Permissions, type TextAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input/index.js';

	let {
		name,
		attr,
		onaction,
		class: className = ''
	}: {
		name: string;
		attr: TextAttribute;
		onaction: (value: string) => void;
		class?: string;
	} = $props();

	// A write-only attribute has no value to read back, so the field starts
	// empty and empties again once sent. That suits what write-only text is
	// for: a one-shot entry such as a pairing PIN, which should not linger.
	let writeOnly = $derived(attr.perms === Permissions.PERM_WRITEONLY);

	let draft = $state('');
	let editing = $state(false);

	// Follow the device's value while the user is not mid-edit, so an update
	// from elsewhere shows up without overwriting something being typed.
	$effect(() => {
		const incoming = attr.value;
		if (!writeOnly && !editing) {
			draft = incoming;
		}
	});

	function send() {
		const value = draft;
		if (writeOnly) {
			draft = '';
		}
		editing = false;
		onaction(value);
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
		placeholder={writeOnly ? `Enter ${name.toLowerCase()}` : ''}
		class={className}
	/>
	<Button class={'cursor-pointer ' + className} variant="dark" onclick={send}>Send</Button>
</div>
