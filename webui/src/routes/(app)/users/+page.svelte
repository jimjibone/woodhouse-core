<script lang="ts">
	import { type User } from '$lib/api/v1/clients/user_service_pb';
	import { UsersStore as store } from '$lib/stores/users-stream';
	import Button from '$lib/components/ui/button/button.svelte';
	import { onDestroy } from 'svelte';
	import { useConnectionContext } from '$lib/stores/connection-status.svelte';
	import { UserPlusIcon } from '@lucide/svelte';
	import UserRow from './user-row.svelte';
	import { IsMobile } from '$lib/hooks/is-mobile.svelte.js';
	import Dialog from '$lib/components/wh/ui/dialog.svelte';
	import UserForm from './user-form.svelte';

	const isMobile = new IsMobile();
	let dialogOpen = $state(false);

	let users = $state<User[]>([]);

	const connStatus = useConnectionContext();
	onDestroy(
		store.subscribe((update) => {
			users = update.users;
			connStatus.set(update.connected, !update.connected && update.backoff > 0);
		})
	);
	onDestroy(() => connStatus.reset());
</script>

<main>
	<div class="pb-4">
		<Button class="cursor-pointer" onclick={() => (dialogOpen = true)}>
			<UserPlusIcon />
			Add User
		</Button>
	</div>
	<div class="flex flex-col gap-4">
		{#each users as user (user.username)}
			<UserRow {user} />
		{/each}
	</div>
</main>

<Dialog bind:open={dialogOpen}>
	<UserForm onSuccess={() => (dialogOpen = false)} />
</Dialog>
