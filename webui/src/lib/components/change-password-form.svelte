<script lang="ts">
	import Button from '$lib/components/ui/button/button.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import Input from '$lib/components/ui/input/input.svelte';
	import { UpdateUserPassword } from '@/stores/requests';
	import { doRefresh, userData } from '@/stores/auth-store';
	import { ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '@/tools/headline-case';
	import { toast } from 'svelte-sonner';

	// Mirrors passwordMinSize in cmd/woodhouse-core/core/user.go. Checking it
	// here only saves a round trip - the server is still the one enforcing it.
	const MIN_LENGTH = 8;

	// legend and description are droppable (pass "") for callers that already
	// headline the form themselves - the forced-reset screen does, and
	// repeating it there just says the same thing twice. Note "" rather than
	// undefined: undefined is exactly what makes Svelte fall back to the
	// default below, so it would render the headings after all.
	let {
		legend = 'Password',
		description = 'Change the password you sign in with.',
		submitLabel = 'Change password',
		currentLabel = 'Current password',
		currentHelp = 'Confirms it is really you at the keyboard, so a signed-in session that is not yours cannot take the account over.',
		onSuccess
	}: {
		legend?: string;
		description?: string;
		submitLabel?: string;
		currentLabel?: string;
		currentHelp?: string;
		onSuccess?: () => void;
	} = $props();

	const id = $props.id();

	let updateError: string | null = $state(null);
	let submitting = $state(false);

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		updateError = null;

		if (newPassword.length < MIN_LENGTH) {
			updateError = `New password must be at least ${MIN_LENGTH} characters`;
			return;
		}
		// Caught here rather than server-side: the server only ever sees one
		// new password, so a typo in it would otherwise silently become the
		// password and lock the user out.
		if (newPassword !== confirmPassword) {
			updateError = 'New passwords do not match';
			return;
		}

		submitting = true;
		try {
			const err = await UpdateUserPassword($userData.username, currentPassword, newPassword);
			if (err instanceof ConnectError) {
				updateError = err.rawMessage;
				return;
			}

			// Clear the fields before anything else - on the forced-reset path
			// this form stays mounted until the refresh below swaps it out.
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';

			// The reset_password claim lives in the access token, so the app
			// stays gated behind this form until a refresh picks up the
			// cleared flag. Unlike the profile form's fire-and-forget refresh
			// this one is awaited and its failure is surfaced: without a fresh
			// token the user would be stuck looking at this form for up to the
			// 60s timer with no sign the save worked.
			const refreshed = await doRefresh().catch(() => false);
			if (!refreshed) {
				updateError = 'Password changed, but the session could not be refreshed. Try reloading the page.';
				return;
			}

			toast.success('Password changed. Any other signed-in sessions have been signed out.');
			onSuccess?.();
		} finally {
			submitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit}>
	<Field.Set>
		{#if legend}
			<Field.Legend>{legend}</Field.Legend>
		{/if}
		{#if description}
			<Field.Description>{description}</Field.Description>
		{/if}

		<Field.Group>
			<Field.Field>
				<Field.Label for="current-password-{id}">{currentLabel}</Field.Label>
				<Input
					id="current-password-{id}"
					name="current-password-{id}"
					type="password"
					autocomplete="current-password"
					required
					bind:value={currentPassword}
				/>
				<Field.Description>{currentHelp}</Field.Description>
			</Field.Field>

			<Field.Field>
				<Field.Label for="new-password-{id}">New password</Field.Label>
				<Input
					id="new-password-{id}"
					name="new-password-{id}"
					type="password"
					autocomplete="new-password"
					required
					bind:value={newPassword}
				/>
				<Field.Description>At least {MIN_LENGTH} characters.</Field.Description>
			</Field.Field>

			<Field.Field>
				<Field.Label for="confirm-password-{id}">Confirm new password</Field.Label>
				<Input
					id="confirm-password-{id}"
					name="confirm-password-{id}"
					type="password"
					autocomplete="new-password"
					required
					bind:value={confirmPassword}
				/>
			</Field.Field>
		</Field.Group>

		{#if updateError}
			<Field.Error>{toSentenceCase(updateError)}</Field.Error>
		{/if}

		<Field.Field>
			<Button type="submit" class="cursor-pointer" disabled={submitting}>
				{submitting ? 'Saving…' : submitLabel}
			</Button>
		</Field.Field>
	</Field.Set>
</form>
