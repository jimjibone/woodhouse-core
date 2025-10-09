<script lang="ts">
	let { children } = $props();
</script>

<div class="fixed bottom-5 self-center shadow-lg rounded-full min-w-12 max-w-[90%] p-1 backdrop-blur-md border">
	<div class="h-12 flex flex-row gap-1 items-center">
		{@render children()}
	</div>
</div>
