<script lang="ts">
	let { children } = $props();
</script>

<div class="fixed bottom-5 self-center shadow-lg rounded-full min-w-10 max-w-[90%] h-12 p-1 bg-card border flex flex-row items-center">
	{@render children()}
</div>
