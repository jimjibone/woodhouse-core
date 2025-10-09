<script lang="ts">
	import { cn } from "$lib/utils";
	import { page } from "$app/state";
	import type { Snippet } from "svelte";

	let {
		onclick,
		children
	} : {
		onclick: ()=>void,
		children: Snippet<[]>
	} = $props();
</script>

<button {onclick} class={cn("h-12 rounded-full aspect-square hover:bg-muted/50 hover:backdrop-blur-md cursor-pointer flex items-center justify-center")}>
	{@render children()}
</button>
