<script lang="ts">
	import Button from '$lib/components/ui/button/button.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import Input from '$lib/components/ui/input/input.svelte';
	import { type User } from '$lib/api/v1/clients/user_service_pb';
	import { UpdateUserFullname } from '@/stores/requests';
	import { doRefresh, userData } from '@/stores/auth-store';
	import { ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '@/tools/headline-case';
	import { toast } from 'svelte-sonner';

	let { user }: { user: User | undefined } = $props();

	const id = $props.id();

	let updateError: string | null = $state(null);
	let submitting = $state(false);

	let fullname = $state('');

	// Mirrors user?.fullname into local state, but only when the incoming value
	// is actually new - not on every stream tick. The stream re-pushes this
	// record on unrelated events too (reconnects, other fields changing), and a
	// naive re-assignment on every run would blow away whatever the user is
	// mid-typing. Comparing against the last value we seeded (rather than the
	// current `fullname`) lets a save round-trip through the stream and land
	// here without that counting as "the user changed it".
	let lastSeenFullname: string | undefined = undefined;
	$effect(() => {
		if (user?.fullname !== lastSeenFullname) {
			lastSeenFullname = user?.fullname;
			fullname = user?.fullname ?? '';
		}
	});

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		updateError = null;

		// Clearing the name is legitimate - the UI falls back to the username
		// when fullname is empty - so no empty check here, just trim it.
		const value = fullname.trim();

		submitting = true;
		try {
			const err = await UpdateUserFullname($userData.username, value);
			if (err instanceof ConnectError) {
				updateError = err.rawMessage;
				return;
			}
			// The stream pushes the saved record back, so the $effect above will
			// pick it up - no local patch needed here.

			// The sidebar reads its display name off the JWT, not the users
			// stream, so without this it would lag behind the change here by
			// up to the 60s refresh timer. Failures are swallowed on purpose:
			// the save already succeeded, and the refresh timer will pick the
			// new name up shortly anyway - a network blip here must not cost
			// the user their success toast. (doRefresh resolves false on a bad
			// response, but still rejects if fetch itself fails.)
			await doRefresh().catch(() => {});
			toast.success('Profile saved');
		} finally {
			submitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit}>
	<Field.Set>
		<Field.Legend>Profile</Field.Legend>
		<Field.Description>How you appear elsewhere in the interface.</Field.Description>

		<Field.Group>
			<Field.Field>
				<Field.Label for="display-name-{id}">Display name</Field.Label>
				<Input
					id="display-name-{id}"
					name="display-name-{id}"
					type="text"
					autocomplete="name"
					maxlength={63}
					bind:value={fullname}
				/>
				<Field.Description>
					Shown instead of your username where there's room, such as the sidebar and this page's header.
				</Field.Description>
			</Field.Field>
		</Field.Group>

		{#if updateError}
			<Field.Error>{toSentenceCase(updateError)}</Field.Error>
		{/if}

		<Field.Field>
			<Button type="submit" class="cursor-pointer" disabled={submitting || user === undefined}>
				{submitting ? 'Saving…' : 'Save'}
			</Button>
		</Field.Field>
	</Field.Set>
</form>
