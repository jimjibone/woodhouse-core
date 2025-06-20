<script lang="ts">
	import NavUser from "./nav-user.svelte";
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import type { ComponentProps } from "svelte";
	import { MoonIcon, SunIcon } from "@lucide/svelte";
	import { WoodhouseIcon } from "@/components/wh/icons";
	import { toggleMode, mode } from "mode-watcher";
	import { page } from "$app/state";

	let {
		ref = $bindable(null),
		collapsible = "icon",
		dashboards = [],
		...restProps
	}: ComponentProps<typeof Sidebar.Root> & { dashboards: Dashboards } = $props();

	const user = {
		name: "shadcn",
		email: "m@example.com",
		avatar: "/avatars/shadcn.jpg",
	};

	export type Dashboards = {
		name: string;
		url: string;
		icon: any;
	}[];
	// const dashboards: Dashboards = [
	// 	{
	// 		name: "Favorites",
	// 		url: "/favorites",
	// 		icon: HeartIcon,
	// 	},
	// 	{
	// 		name: "Services",
	// 		url: "/services",
	// 		icon: Rows3Icon,
	// 	},
	// 	{
	// 		name: "Devices",
	// 		url: "/devices",
	// 		icon: LampIcon,
	// 	},
	// 	{
	// 		name: "Debug",
	// 		url: "/debug",
	// 		icon: BugIcon,
	// 	},
	// ];
</script>

<Sidebar.Root {collapsible} {...restProps}>
	<Sidebar.Header>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Sidebar.MenuButton size="lg">
					{#snippet child({ props })}
						<a href="/" {...props}>
							<div class="flex aspect-square size-8 items-center justify-center rounded-lg [[data-collapsed=true]_&]:bg-sidebar-primary [[data-collapsed=true]_&]:text-sidebar-primary-foreground transition-[background-color,color] duration-200 ease-linear">
								<WoodhouseIcon class="size-5" />
							</div>
							<span class="text-lg">Woodhouse</span>
						</a>
					{/snippet}
				</Sidebar.MenuButton>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
	</Sidebar.Header>
	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.GroupLabel>Dashboards</Sidebar.GroupLabel>
			<Sidebar.Menu>
				{#each dashboards as item (item.name)}
					<Sidebar.MenuItem>
						<Sidebar.MenuButton isActive={page.url.pathname === item.url}>
							{#snippet child({ props })}
								<a href={item.url} {...props}>
									<item.icon/>
									<span>{item.name}</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				{/each}
			</Sidebar.Menu>
		</Sidebar.Group>
	</Sidebar.Content>
	<Sidebar.Separator />
	<Sidebar.Footer>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Sidebar.MenuButton onclick={toggleMode} class="cursor-pointer">
					{#if mode.current == "dark"}
					<MoonIcon/>
					{:else}
					<SunIcon/>
					{/if}
					<span>Toggle Theme</span>
				</Sidebar.MenuButton>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
		<NavUser user={user} />
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
