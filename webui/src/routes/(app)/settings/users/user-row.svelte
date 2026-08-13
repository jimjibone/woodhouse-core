<script lang="ts">
	import { type User, UserRole } from '$lib/api/v1/clients/user_service_pb';
	import * as Avatar from '$lib/components/ui/avatar/index.js';
	import Button from '$lib/components/ui/button/button.svelte';
	import { PencilIcon } from '@lucide/svelte';
	import { makeAcronym } from '$lib/tools/acronym';
	import Dialog from '$lib/components/wh/ui/dialog.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import Input from '$lib/components/ui/input/input.svelte';
	import * as RadioGroup from '$lib/components/ui/radio-group/index.js';
	import { ResetUserPassword, UpdateUserFullname, UpdateUserRole } from '@/stores/requests';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { toast } from 'svelte-sonner';
	import { doRefresh, userData } from '@/stores/auth-store';
	import type { ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '@/tools/headline-case';

	let { user }: { user: User } = $props();

	const id = $props.id();

	let initials = $derived(makeAcronym(user.fullname, user.username));

	let dialogOpen = $state(false);

	const roleOptions = [
		{ value: UserRole.UNDEFINED, label: 'Undefined' },
		{ value: UserRole.ADMIN, label: 'Admin' },
		{ value: UserRole.USER, label: 'User' }
	];

	function roleToString(role: UserRole): string {
		let res = roleOptions.find((r) => r.value === role);
		if (res) return res.label;
		return 'UNIMPLEMENTED';
	}

	function roleFromString(role: string): UserRole {
		let res = roleOptions.find((r) => r.label === role);
		if (res) return res.value;
		return UserRole.UNDEFINED;
	}

	let updateError: ConnectError | null = $state(null);

	// Local copies of these settings for the modal - supresses server updates
	// during edit.
	let fullname = $state('');
	let role = $state('');

	$effect(() => {
		if (dialogOpen) return;
		fullname = user.fullname;
		role = roleToString(user.role);
		newPassword = '';
		resetError = null;
	});

	// Mirrors passwordMinSize in cmd/woodhouse-core/core/user.go.
	const MIN_PASSWORD_LENGTH = 8;

	let newPassword = $state('');
	let resetError: string | null = $state(null);
	let resetting = $state(false);

	// An admin resetting their *own* password has to go through /profile
	// instead: the server routes a self-change through the path that demands
	// the current password, which this form has no field for and no business
	// asking an admin to bypass.
	const isSelf = $derived(user.username === $userData.username);

	async function handleResetPassword() {
		resetError = null;

		if (newPassword.length < MIN_PASSWORD_LENGTH) {
			resetError = `Password must be at least ${MIN_PASSWORD_LENGTH} characters`;
			return;
		}

		resetting = true;
		try {
			const err = await ResetUserPassword(user.username, newPassword);
			if (err) {
				resetError = err.rawMessage;
				return;
			}
			newPassword = '';
			toast.success(`Password reset for ${user.username}. They must choose a new one at next sign-in.`);
		} finally {
			resetting = false;
		}
	}

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		updateError = null;

		let saved = false;

		// Clearing the name is legitimate - the UI falls back to the username when
		// fullname is empty - so an empty field is a real edit, not a skip. Same as
		// the profile form.
		const newFullname = fullname.trim();

		if (newFullname !== user.fullname) {
			const err = await UpdateUserFullname(user.username, newFullname);
			if (err) updateError = err;
			else saved = true;
		}

		if (role !== roleToString(user.role)) {
			const err = await UpdateUserRole(user.username, roleFromString(role));
			// Keep the first failure - a later success must not blank out the
			// error from an earlier field.
			if (err) updateError ??= err;
			else saved = true;
		}

		// Fullname and role are both JWT claims, so an admin editing *their own*
		// row here would otherwise keep seeing the old name in the sidebar (and
		// the old role gating the admin-only nav) until the 60s refresh timer
		// caught up. Same trick as the profile form. Failures are swallowed on
		// purpose: the save already succeeded and the timer will catch up anyway,
		// so a network blip must not surface as a save error.
		//
		// Deliberately keyed off `saved` rather than "no error": if one field
		// saved and the other failed, the token's claims still changed and the
		// sidebar still needs refreshing. Doing it before the close below also
		// means the sidebar is already correct by the time the dialog goes away.
		if (saved && user.username === $userData.username) {
			await doRefresh().catch(() => {});
		}

		// Dismiss on success - closing is this dialog's only success feedback. A
		// failure keeps it open so the error stays visible next to the fields.
		if (!updateError) {
			dialogOpen = false;
		}
	}

	// Enter in a text field would otherwise implicitly submit the form, which
	// saves and closes the dialog out from under someone who was only editing.
	// Saving has to go through the Save button (Enter while it's focused still
	// works - that's a click, not an implicit submit).
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') event.preventDefault();
	}
</script>

{#snippet Role(role: UserRole)}
	{#if role === UserRole.UNDEFINED}
		Undefined
	{:else if role === UserRole.ADMIN}
		Admin
	{:else if role === UserRole.USER}
		User
	{/if}
{/snippet}

<div class="rounded-lg border bg-card/50 p-2 text-card-foreground shadow-sm flex flex-row gap-2">
	<div class="shrink pl-0">
		<Avatar.Root class="size-12 rounded-full">
			<Avatar.Image src={''} alt={user.username} />
			<Avatar.Fallback>{initials}</Avatar.Fallback>
		</Avatar.Root>
	</div>
	<div class="shrink flex flex-col">
		<span class="font-semibold">{user.fullname ? user.fullname : 'No Name'}</span>
		<span class="text-muted-foreground">{user.username}</span>
	</div>
	<div class="grow flex flex-row items-center justify-center gap-2">
		{#if user.resetPassword}
			<!-- Either a brand new account or one that has just been reset:
			they are still on a password somebody else chose. -->
			<Badge variant="secondary">Password reset pending</Badge>
		{/if}
		<span class="text-muted-foreground">{roleToString(user.role)}</span>
	</div>
	<div class="shrink flex flex-row pr-2 gap-2 items-center">
		<!-- <Combobox options={roleOptions} value={user.role} /> -->
		<Button variant="secondary" size="icon" class="size-8 cursor-pointer" onclick={() => (dialogOpen = true)}
			><PencilIcon /></Button
		>
	</div>
</div>
<Dialog bind:open={dialogOpen}>
	<form onsubmit={handleSubmit}>
		<Field.Set>
			<Field.Group>
				<Field.Set>
					<Field.Legend>Profile</Field.Legend>
					<Field.Field>
						<Field.Label for="username-{id}">Username</Field.Label>
						<Input id="username-{id}" disabled value={user.username} />
					</Field.Field>
					<Field.Field>
						<Field.Label for="fullname-{id}">Full name</Field.Label>
						<Input
							id="fullname-{id}"
							name="fullname-{id}"
							type="text"
							placeholder="Dade Murphy"
							autocomplete="off"
							bind:value={fullname}
							onkeydown={handleKeydown}
						/>
						<Field.Description>This appears in the user interface.</Field.Description>
					</Field.Field>
				</Field.Set>
			</Field.Group>

			<Field.Group>
				<Field.Set>
					<Field.Legend>Role</Field.Legend>
					<Field.Description>Select the role for this user.</Field.Description>
					<RadioGroup.Root bind:value={role} name="role-group-{id}">
						<Field.Label>
							<Field.Field orientation="horizontal" class="cursor-pointer">
								<Field.Content>
									<Field.Title>Admin</Field.Title>
									<Field.Description>Allows full access to all features.</Field.Description>
								</Field.Content>
								<RadioGroup.Item value="Admin" />
							</Field.Field>
						</Field.Label>
						<Field.Label>
							<Field.Field orientation="horizontal" class="cursor-pointer">
								<Field.Content>
									<Field.Title>User</Field.Title>
									<Field.Description>Only allowed to view and control devices.</Field.Description>
								</Field.Content>
								<RadioGroup.Item value="User" />
							</Field.Field>
						</Field.Label>
					</RadioGroup.Root>
				</Field.Set>
			</Field.Group>

			{#if updateError}
				<Field.Error>{toSentenceCase(updateError.rawMessage)}</Field.Error>
			{/if}
			<Field.Field>
				<Button type="submit" class="cursor-pointer">Save</Button>
			</Field.Field>

			<Field.Group>
				<Field.Set>
					<Field.Legend>Password</Field.Legend>
					{#if isSelf}
						<Field.Description>
							Change your own password from your <a href="/profile" class="underline">profile</a>, where it can be
							confirmed against your current one.
						</Field.Description>
					{:else}
						<Field.Description>
							Set a temporary password for {user.username}. They must choose their own before they can use Woodhouse
							again, and all of their existing sessions are signed out.
						</Field.Description>
						<Input
							id="reset-password-{id}"
							name="reset-password-{id}"
							type="password"
							placeholder="********"
							autocomplete="off"
							bind:value={newPassword}
							onkeydown={handleKeydown}
						/>
						{#if resetError}
							<Field.Error>{toSentenceCase(resetError)}</Field.Error>
						{/if}
						<Field.Field>
							<!-- Deliberately type="button": resetting a password is
							its own action, not part of saving name and role. -->
							<Button
								type="button"
								variant="outline"
								class="cursor-pointer"
								disabled={resetting || newPassword.length === 0}
								onclick={handleResetPassword}
							>
								{resetting ? 'Resetting…' : 'Reset password'}
							</Button>
						</Field.Field>
					{/if}
				</Field.Set>
			</Field.Group>
		</Field.Set>
	</form>
</Dialog>
