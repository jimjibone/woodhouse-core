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
	import { UpdateUserFullname, UpdateUserRole } from '@/stores/requests';
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

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		const form = event.target as HTMLFormElement;
		const data = new FormData(form);
		const fullname = data.get(`fullname-${id}`);
		const role = data.get(`role-group-${id}`);
		updateError = null;

		if (fullname && fullname !== user.fullname) {
			updateError = await UpdateUserFullname(user.username, fullname.toString());
		}

		if (role && role !== roleToString(user.role)) {
			updateError = await UpdateUserRole(user.username, roleFromString(role.toString()));
		}
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
	<div class="grow flex flex-row items-center justify-center">
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
			<!-- <Field.Legend>Profile</Field.Legend>
    		<Field.Description>This appears on invoices and emails.</Field.Description> -->
			<Field.Group>
				<Field.Field>
					<Field.Label for="username">Username</Field.Label>
					<Input id="username" disabled value={user.username} />
				</Field.Field>
				<Field.Field>
					<Field.Label for="fullname-{id}">Full name</Field.Label>
					<Input name="fullname-{id}" type="text" placeholder="Dade Murphy" autocomplete="off" value={user.fullname} />
					<Field.Description>This appears in the user interface.</Field.Description>
				</Field.Field>
			</Field.Group>

			<Field.Group>
				<Field.Set>
					<Field.Label for="role-group-{id}">Role</Field.Label>
					<Field.Description>Select the role for this user.</Field.Description>
					<RadioGroup.Root
						value={roleToString(user.role)}
						onValueChange={(v) => console.log('v:', v)}
						name="role-group-{id}"
					>
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
		</Field.Set>
	</form>
</Dialog>
