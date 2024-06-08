<script lang="ts">
	import { Label } from "$lib/components/ui/label/index.js";

	export let id: string;
	export let alias: string;
</script>

{#if alias === ""}
<Label>{id}</Label>
{:else}
<Label class="max-sm:hidden">{alias} ({id})</Label>
<Label class="sm:hidden">{alias}</Label>
{/if}
