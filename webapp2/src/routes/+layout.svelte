<script lang="ts">
	import '../app.pcss';
	import { ModeWatcher } from "mode-watcher";
	import { buttonVariants } from "$lib/components/ui/button";
	import { page } from '$app/stores';
	import { Button } from '$lib/components/ui/button';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as Tooltip from '$lib/components/ui/tooltip/index.js';

	import {
		Sun,
		Moon,
		Settings,
		LayoutDashboard,
		Lamp,
		Bug,
	} from 'lucide-svelte';

	import { resetMode, setMode } from 'mode-watcher';
</script>

<ModeWatcher />

<div class="grid h-screen w-full pl-[53px]">
	<aside class="inset-y fixed left-0 z-20 flex h-full flex-col border-r">
		<div class="border-b p-2">
			<Button variant="outline" size="icon" aria-label="Home" href="/">
				<!-- <Home class="size-5" /> -->
				<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="2,8 12,2 22,8"></polyline>
					<polyline points="17,5 17,3 18,3 18,5"></polyline>
					<polyline points="5,12 7,21 12,18 17,21 19,12"></polyline>
				</svg>
			</Button>
		</div>
		<nav class="grid gap-1 p-2">
			<Tooltip.Root>
				<Tooltip.Trigger>
					<a
						href="/"
						aria-label="Dashboard"
						class={buttonVariants({ variant: "ghost", size: "icon", class: "rounded-lg" })}
						class:bg-muted={$page.url.pathname === '/'}
					>
						<LayoutDashboard class="size-5" />
					</a>
				</Tooltip.Trigger>
				<Tooltip.Content side="right" sideOffset={5}>Dashboard</Tooltip.Content>
			</Tooltip.Root>
			<Tooltip.Root>
				<Tooltip.Trigger>
					<a
						href="/devices"
						aria-label="Devices"
						class={buttonVariants({ variant: "ghost", size: "icon", class: "rounded-lg" })}
						class:bg-muted={$page.url.pathname === '/devices'}
					>
						<Lamp class="size-5" />
					</a>
				</Tooltip.Trigger>
				<Tooltip.Content side="right" sideOffset={5}>Devices</Tooltip.Content>
			</Tooltip.Root>
			<Tooltip.Root>
				<Tooltip.Trigger>
					<a
						href="/debug"
						aria-label="Debug"
						class={buttonVariants({ variant: "ghost", size: "icon", class: "rounded-lg" })}
						class:bg-muted={$page.url.pathname === '/debug'}
					>
						<Bug class="size-5" />
					</a>
				</Tooltip.Trigger>
				<Tooltip.Content side="right" sideOffset={5}>Debug</Tooltip.Content>
			</Tooltip.Root>
		</nav>
		<nav class="mt-auto grid gap-1 p-2">
			<Tooltip.Root>
				<Tooltip.Trigger>
					<DropdownMenu.Root>
						<DropdownMenu.Trigger asChild let:builder>
							<Button
								variant="ghost"
								size="icon"
								class="mt-auto rounded-lg"
								aria-label="Toggle Theme"
								builders={[builder]}
							>
								<Sun
									class="h-[1.2rem] w-[1.2rem] rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0"
								/>
								<Moon
									class="absolute h-[1.2rem] w-[1.2rem] rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100"
								/>
							</Button>
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="end">
							<DropdownMenu.Item on:click={() => setMode('light')}>Light</DropdownMenu.Item>
							<DropdownMenu.Item on:click={() => setMode('dark')}>Dark</DropdownMenu.Item>
							<DropdownMenu.Item on:click={() => resetMode()}>System</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</Tooltip.Trigger>
				<Tooltip.Content side="right" sideOffset={5}>Toggle Theme</Tooltip.Content>
			</Tooltip.Root>
			<Tooltip.Root>
				<Tooltip.Trigger>
					<a
						href="/settings"
						aria-label="Settings"
						class={buttonVariants({ variant: "ghost", size: "icon", class: "rounded-lg" })}
						class:bg-muted={$page.url.pathname === '/settings'}
					>
						<Settings class="size-5" />
					</a>
				</Tooltip.Trigger>
				<Tooltip.Content side="right" sideOffset={5}>Settings</Tooltip.Content>
			</Tooltip.Root>
		</nav>
	</aside>
	<div class="flex flex-col">
		<slot></slot>
	</div>
</div>
