<script lang="ts">
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import { type Dashboards } from "./app-sidebar.svelte";
	import { MobileNav, MobileNavItem } from "$lib/components/wh/nav";

	let {
		dashboards = [],
	} : {
		dashboards: Dashboards
	} = $props();
</script>

{#if Sidebar.useSidebar().isMobile}
	<MobileNav>
		{#each dashboards as item (item.name)}
			<MobileNavItem href={item.url}>
				<item.icon class="size-6"/>
			</MobileNavItem>
		{/each}
	</MobileNav>
{/if}
