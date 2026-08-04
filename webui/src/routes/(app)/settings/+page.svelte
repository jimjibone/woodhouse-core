<script lang="ts">
	import { onMount } from 'svelte';
	import { userData } from '$lib/stores/auth-store';
	import { loadSettings } from '$lib/stores/settings-store';
	import InstanceForm from './instance-form.svelte';

	const isAdmin = $derived($userData.role === 'admin');

	onMount(() => {
		loadSettings();
	});
</script>

<main class="max-w-2xl">
	{#if isAdmin}
		<div class="flex flex-col gap-8">
			<InstanceForm />
		</div>
	{:else}
		<p class="text-muted-foreground text-sm">You need to be an admin to change server settings.</p>
	{/if}
</main>
