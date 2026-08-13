<script lang="ts">
	import Button from '$lib/components/ui/button/button.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import Input from '$lib/components/ui/input/input.svelte';
	import { Switch } from '$lib/components/ui/switch/index.js';
	import { UpdateSettings } from '@/stores/requests';
	import { settings, applySettings, PRODUCT_NAME } from '@/stores/settings-store';
	import { ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '@/tools/headline-case';
	import { toast } from 'svelte-sonner';

	const id = $props.id();

	let updateError: string | null = $state(null);
	let submitting = $state(false);

	// Mirrors the stored preference until the user saves, so the toggle can be
	// moved and then reverted without touching the server.
	let showInstanceName = $state(false);
	$effect(() => {
		showInstanceName = $settings.showInstanceName;
	});

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		updateError = null;

		const form = event.target as HTMLFormElement;
		const data = new FormData(form);
		const instanceName = data.get(`instance-name-${id}`);

		if (!instanceName) {
			updateError = 'instance name must not be empty';
			return;
		}

		submitting = true;
		try {
			const res = await UpdateSettings({
				instanceName: instanceName.toString(),
				showInstanceName: showInstanceName
			});
			if (res instanceof ConnectError) {
				updateError = res.rawMessage;
				return;
			}
			// The server trims the name, so show what it actually stored.
			applySettings(res);
			toast.success('Settings saved');
		} finally {
			submitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit}>
	<Field.Set>
		<Field.Legend>General</Field.Legend>
		<Field.Description>Settings that apply to this Woodhouse server.</Field.Description>

		<Field.Group>
			<Field.Field>
				<Field.Label for="instance-name-{id}">Instance name</Field.Label>
				<Input
					id="instance-name-{id}"
					name="instance-name-{id}"
					type="text"
					placeholder="woodhouse"
					autocomplete="off"
					maxlength={63}
					required
					value={$settings.instanceName}
				/>
				<Field.Description>
					The name this server advertises on your local network. It is what you see when picking a server in the
					Woodhouse app. Changes apply immediately - no restart needed.
				</Field.Description>
			</Field.Field>

			<Field.Field orientation="horizontal">
				<Field.Content>
					<Field.Title>Show the instance name here</Field.Title>
					<Field.Description>
						Show the instance name in place of "{PRODUCT_NAME}" in the sidebar and the browser tab. Useful when you run
						more than one server and need to tell them apart.
					</Field.Description>
				</Field.Content>
				<Switch
					id="show-instance-name-{id}"
					bind:checked={showInstanceName}
					aria-label="Show the instance name in the interface"
				/>
			</Field.Field>
		</Field.Group>

		{#if updateError}
			<Field.Error>{toSentenceCase(updateError)}</Field.Error>
		{/if}

		<Field.Field>
			<Button type="submit" class="cursor-pointer" disabled={submitting || !$settings.loaded}>
				{submitting ? 'Saving…' : 'Save'}
			</Button>
		</Field.Field>
	</Field.Set>
</form>
