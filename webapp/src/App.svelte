<script lang="ts">
	import { Router, Route, Link } from "svelte-routing";
	import classnames from "classnames";
	import NavLink from "./components/NavLink.svelte";
	import Toast from "./components/Toast.svelte";
	import DevicesPage from "./pages/DevicesPage.svelte";
	import BridgesPage from "./pages/BridgesPage.svelte";
	import FavouritesPage from "./pages/FavouritesPage.svelte";

	import { Button } from "$lib/components/ui/button";
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
	import { Sun, Moon } from 'lucide-svelte';

	export let url = "";

	let navOpen = false;
	$: navbarBurgerClasses = classnames("navbar-burger", {
		"is-active": navOpen,
	});
	$: navbarMenuClasses = classnames("navbar-menu", { "is-active": navOpen });

	function toggleNavOpen() {
		navOpen = !navOpen;
	}

	function toggleDarkMode() {
		if (localStorage.theme == undefined) {
			localStorage.theme = 'dark';
		} else if (localStorage.theme === 'dark') {
			localStorage.theme = 'light';
		} else if (localStorage.theme === 'light') {
			localStorage.removeItem('theme');
		}
		updateLightDarkMode();
	}

	let darkMode = (localStorage.theme === "dark" || (!("theme" in localStorage) && window.matchMedia("(prefers-color-scheme: dark)").matches));

	function setDarkMode() {
		localStorage.theme = 'dark';
		updateLightDarkMode();
	}
	function setLightMode() {
		localStorage.theme = 'light';
		updateLightDarkMode();
	}
	function setAutoMode() {
		localStorage.removeItem('theme');
		updateLightDarkMode();
	}
	function updateLightDarkMode() {
		if (localStorage.theme === 'dark' || (!('theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
			document.documentElement.classList.add('dark');
			darkMode = true;
		} else {
			document.documentElement.classList.remove('dark');
			darkMode = false;
		}
	}
</script>

<main>
	<Router {url}>
		<header class="supports-[backdrop-filter]:bg-background/60 sticky top-0 z-50 w-full border-b bg-background/95 shadow-sm backdrop-blur">
			<div class="container flex h-14 items-center">
				<div class="mr-4 md:flex">
					<Link to="/" class="mr-6 flex items-center space-x-2">
						<span class="font-bold sm:inline-block text-[15px] lg:text-base">Woodhouse</span>
					</Link>
					<nav class="flex items-center space-x-6 text-sm font-medium">
						<NavLink to="/">Favourites</NavLink>
						<NavLink to="/devices">Devices</NavLink>
						<NavLink to="/bridges">Bridges</NavLink>
					</nav>
				</div>
				<div class="flex flex-1 items-center justify-between space-x-2 sm:space-x-4 md:justify-end">
					<nav class="flex items-center">
						<DropdownMenu.Root preventScroll={false}>
							<DropdownMenu.Trigger asChild let:builder>
								<Button builders={[builder]} variant="ghost" size="sm">
									{#if darkMode}
										<Moon size={18} />
									{:else}
										<Sun size={18} />
									{/if}
								</Button>
							</DropdownMenu.Trigger>
							<DropdownMenu.Content class="w-56">
								<DropdownMenu.Item on:click={setLightMode}>
									<span>Light</span>
								</DropdownMenu.Item>
								<DropdownMenu.Item on:click={setDarkMode}>
									<span>Dark</span>
								</DropdownMenu.Item>
								<DropdownMenu.Item on:click={setAutoMode}>
									<span>System</span>
								</DropdownMenu.Item>
							</DropdownMenu.Content>
						</DropdownMenu.Root>
					</nav>
				</div>
			</div>
		</header>

		<div>
			<Route path="/" component="{FavouritesPage}" />
			<Route path="/devices" component="{DevicesPage}" />
			<Route path="/bridges" component="{BridgesPage}" />
		</div>
	</Router>
	<Toast/>
</main>

<style>
</style>
