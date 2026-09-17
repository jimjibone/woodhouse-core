<script lang="ts">
	import { onMount } from 'svelte';
	import { loadSettings } from '$lib/stores/settings-store';
	import InstanceForm from './instance-form.svelte';
	import CertificateSection from './certificate-section.svelte';

	onMount(() => {
		loadSettings();
	});
</script>

<main class="max-w-2xl">
	<div class="flex flex-col gap-8">
		<InstanceForm />
		<CertificateSection />
	</div>
</main>
