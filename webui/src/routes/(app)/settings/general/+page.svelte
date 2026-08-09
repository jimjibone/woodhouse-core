<script lang="ts">
	import { onMount } from 'svelte';
	import { loadSettings } from '$lib/stores/settings-store';
	import InstanceForm from './instance-form.svelte';

	onMount(() => {
		loadSettings();
	});
</script>

<main class="max-w-2xl">
	<div class="flex flex-col gap-8">
		<InstanceForm />
	</div>
</main>
