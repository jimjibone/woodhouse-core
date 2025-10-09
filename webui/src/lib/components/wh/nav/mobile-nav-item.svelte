<script lang="ts">
	import { cn } from "$lib/utils";
	import { page } from "$app/state";
	import type { Snippet } from "svelte";

	let {
		children,
		href
	} : {
		children: Snippet<[]>,
		href: string
	} = $props();
</script>

<a class={cn("h-12 rounded-full aspect-square hover:bg-muted/50 hover:backdrop-blur-md cursor-pointer flex items-center justify-center", page.url.pathname === href && "bg-muted/70 backdrop-blur-md")} href={href}>
	{@render children()}
</a>
