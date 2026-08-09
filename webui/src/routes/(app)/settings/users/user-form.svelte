<script lang="ts">
	import { UserRole } from '$lib/api/v1/clients/user_service_pb';
	import Button from '$lib/components/ui/button/button.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import Input from '$lib/components/ui/input/input.svelte';
	import * as RadioGroup from '$lib/components/ui/radio-group/index.js';
	import { AddUser, UpdateUserFullname, UpdateUserRole } from '@/stores/requests';
	import type { ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '@/tools/headline-case';

	let { onSuccess }: { onSuccess: () => void } = $props();

	const id = $props.id();

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

	let updateError: string | null = $state(null);

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		const form = event.target as HTMLFormElement;
		const data = new FormData(form);
		const username = data.get(`username-${id}`);
		const fullname = data.get(`fullname-${id}`);
		const role = data.get(`role-group-${id}`);
		const initialPassword = data.get(`initial-password-${id}`);

		if (username && fullname && role && initialPassword) {
			const res = await AddUser(
				username.toString(),
				fullname.toString(),
				roleFromString(role.toString()),
				initialPassword.toString()
			);
			if (res) {
				updateError = res.rawMessage;
			} else {
				onSuccess();
			}
		} else {
			updateError = 'missing fields';
		}
	}
</script>

<form onsubmit={handleSubmit}>
	<Field.Set>
		<Field.Legend>New User</Field.Legend>
		<Field.Description>Create a new user for Woodhouse.</Field.Description>

		<Field.Group>
			<Field.Field>
				<Field.Label for="username-{id}">Username</Field.Label>
				<Input name="username-{id}" type="text" placeholder="crashoverride" autocomplete="off" required />
				<Field.Description>A unique username. This cannot be changed.</Field.Description>
			</Field.Field>
			<Field.Field>
				<Field.Label for="fullname-{id}">Full name</Field.Label>
				<Input name="fullname-{id}" type="text" placeholder="Dade Murphy" autocomplete="off" required />
				<Field.Description>This appears in the user interface.</Field.Description>
			</Field.Field>
			<Field.Field>
				<Field.Label for="initial-password-{id}">Initial password</Field.Label>
				<Input name="initial-password-{id}" type="password" placeholder="********" autocomplete="off" required />
				<Field.Description
					>A temporary password which the user will be prompted to change on first login.</Field.Description
				>
			</Field.Field>
		</Field.Group>

		<Field.Group>
			<Field.Set>
				<Field.Label for="role-group-{id}">Role</Field.Label>
				<Field.Description>Select the role for this user.</Field.Description>
				<RadioGroup.Root value={roleToString(UserRole.USER)} name="role-group-{id}">
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
			<Field.Error>{toSentenceCase(updateError)}</Field.Error>
		{/if}

		<Field.Field>
			<Button type="submit" class="cursor-pointer">Create</Button>
		</Field.Field>
	</Field.Set>
</form>
