<script lang="ts">
	import NavUser from './nav-user.svelte';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import type { ComponentProps } from 'svelte';
	import { MoonIcon, SunIcon } from '@lucide/svelte';
	import { WoodhouseIcon } from '@/components/wh/icons';
	import { toggleMode, mode } from 'mode-watcher';
	import { page } from '$app/state';
	import { type Dashboards, isPathActive } from '$lib/nav';
	import type { Zone } from '$lib/api/v1/clients/zone_pb';
	import { zoneIcon } from '$lib/zone-icons';

	let {
		ref = $bindable(null),
		collapsible = 'icon',
		dashboards = [],
		zones = [],
		title = 'Woodhouse',
		...restProps
	}: ComponentProps<typeof Sidebar.Root> & {
		dashboards: Dashboards;
		zones?: Zone[];
		title?: string;
	} = $props();
</script>

<Sidebar.Root {collapsible} {...restProps}>
	<Sidebar.Header>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Sidebar.MenuButton size="lg">
					{#snippet child({ props })}
						<a href="/" {...props}>
							<div
								class="flex aspect-square size-8 items-center justify-center rounded-lg [[data-collapsed=true]_&]:bg-sidebar-primary [[data-collapsed=true]_&]:text-sidebar-primary-foreground transition-[background-color,color] duration-200 ease-linear"
							>
								<WoodhouseIcon class="size-5" />
							</div>
							<span class="truncate text-lg" {title}>{title}</span>
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
						<Sidebar.MenuButton isActive={isPathActive(page.url.pathname, item.url)}>
							{#snippet child({ props })}
								<a href={item.url} {...props}>
									<item.icon />
									<span>{item.name}</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				{/each}
			</Sidebar.Menu>
		</Sidebar.Group>

		<!--
			Zones are visible to every user - they are how you navigate to a room's
			devices. Only creating and editing them is admin-only, which lives under
			Settings. The group is omitted entirely until a zone exists so a fresh
			install does not show an empty heading.
		-->
		{#if zones.length > 0}
			<Sidebar.Group>
				<Sidebar.GroupLabel>Zones</Sidebar.GroupLabel>
				<Sidebar.Menu>
					{#each zones as zone (zone.id)}
						{@const ZoneIcon = zoneIcon(zone.icon)}
						<Sidebar.MenuItem>
							<Sidebar.MenuButton isActive={isPathActive(page.url.pathname, '/zones/' + zone.id)}>
								{#snippet child({ props })}
									<a href={'/zones/' + zone.id} {...props}>
										<ZoneIcon />
										<span>{zone.name}</span>
									</a>
								{/snippet}
							</Sidebar.MenuButton>
						</Sidebar.MenuItem>
					{/each}
				</Sidebar.Menu>
			</Sidebar.Group>
		{/if}
	</Sidebar.Content>
	<Sidebar.Separator />
	<Sidebar.Footer>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Sidebar.MenuButton onclick={toggleMode} class="cursor-pointer">
					{#if mode.current == 'dark'}
						<MoonIcon />
					{:else}
						<SunIcon />
					{/if}
					<span>Toggle Theme</span>
				</Sidebar.MenuButton>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
		<NavUser />
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
