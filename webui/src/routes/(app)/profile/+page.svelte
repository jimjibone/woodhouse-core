<script lang="ts">
	import { type User } from '$lib/api/v1/clients/user_service_pb';
	import { UsersStore as store } from '$lib/stores/users-stream';
	import * as Avatar from '$lib/components/ui/avatar/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import Button from '$lib/components/ui/button/button.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import * as RadioGroup from '$lib/components/ui/radio-group/index.js';
	import ProfileForm from './profile-form.svelte';
	import ChangePasswordForm from '$lib/components/change-password-form.svelte';
	import { onDestroy } from 'svelte';
	import { useConnectionContext } from '$lib/stores/connection-status.svelte';
	import { doLogout, userData } from '@/stores/auth-store';
	import { makeAcronym } from '@/tools/acronym';
	import { setMode, userPrefersMode } from 'mode-watcher';

	let users = $state<User[]>([]);

	const connStatus = useConnectionContext();
	onDestroy(
		store.subscribe((update) => {
			users = update.users;
			connStatus.set(update.connected, !update.connected && update.backoff > 0);
		})
	);
	onDestroy(() => connStatus.reset());

	// The JWT carries username and role but not the display name, so the record
	// has to come from the users stream. Non-admins only ever receive their own.
	const me = $derived(users.find((u) => u.username === $userData.username));

	// Before `me` resolves, fall back to what the JWT already gives us so the
	// header renders sensibly on first paint instead of showing a skeleton.
	const initials = $derived(makeAcronym(me?.fullname, $userData.username));
</script>

<main class="max-w-2xl mb-20 md:mb-0 grid gap-6">
	<div class="rounded-xl border bg-card/50 p-4 shadow-sm flex items-center gap-3">
		<Avatar.Root class="size-12 rounded-lg">
			<Avatar.Fallback>{initials}</Avatar.Fallback>
		</Avatar.Root>
		<div class="grid">
			<span class="font-semibold">{$userData.username}</span>
			{#if me?.fullname}
				<span class="text-muted-foreground text-sm">{me.fullname}</span>
			{/if}
		</div>
		<Badge variant="secondary" class="ml-auto">{$userData.role}</Badge>
	</div>

	<ProfileForm user={me} />

	<ChangePasswordForm
		description="Change the password you sign in with. Doing so signs out every other session on your account."
	/>

	<Field.Set>
		<Field.Legend>Appearance</Field.Legend>
		<Field.Description>
			Stored in this browser only - it does not sync across devices and does not touch anything on the server.
		</Field.Description>

		<RadioGroup.Root value={userPrefersMode.current} onValueChange={(v) => setMode(v as 'light' | 'dark' | 'system')}>
			<Field.Label>
				<Field.Field orientation="horizontal" class="cursor-pointer">
					<Field.Content>
						<Field.Title>Light</Field.Title>
					</Field.Content>
					<RadioGroup.Item value="light" />
				</Field.Field>
			</Field.Label>
			<Field.Label>
				<Field.Field orientation="horizontal" class="cursor-pointer">
					<Field.Content>
						<Field.Title>Dark</Field.Title>
					</Field.Content>
					<RadioGroup.Item value="dark" />
				</Field.Field>
			</Field.Label>
			<Field.Label>
				<Field.Field orientation="horizontal" class="cursor-pointer">
					<Field.Content>
						<Field.Title>System</Field.Title>
					</Field.Content>
					<RadioGroup.Item value="system" />
				</Field.Field>
			</Field.Label>
		</RadioGroup.Root>
	</Field.Set>

	<Field.Set>
		<Field.Legend>Session</Field.Legend>

		<Field.Field orientation="horizontal">
			<Field.Content>
				<Field.Title>Current session</Field.Title>
				<Field.Description>Signed in as {$userData.username}.</Field.Description>
			</Field.Content>
			<Button variant="outline" class="cursor-pointer" onclick={() => doLogout()}>Log out</Button>
		</Field.Field>
	</Field.Set>
</main>
