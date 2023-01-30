<script lang="ts">
	import { Router, Route, Link } from 'svelte-routing';
	import classnames from 'classnames';
	import NavLink from './components/NavLink.svelte';
	import Toast from './components/Toast.svelte';
	import DevicesPage from './pages/DevicesPage.svelte';
	import BridgesPage from './pages/BridgesPage.svelte';

	export let url = "";

	let navOpen = false;
	$: navbarBurgerClasses = classnames('navbar-burger', {'is-active': navOpen});
	$: navbarMenuClasses = classnames('navbar-menu', {'is-active': navOpen});

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

		if (localStorage.theme === 'dark' || (!('theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
			document.documentElement.classList.add('dark');
		} else {
			document.documentElement.classList.remove('dark');
		}
	}
</script>

<main>
	<Router url="{url}">
		<nav class="navbar is-light" aria-label="main navigation">
			<div class="navbar-brand">
				<Link to="/" class="navbar-item">
					Woodhouse
				</Link>

				<button class={navbarBurgerClasses} aria-label="menu" aria-expanded="false" data-target="navbarBasicExample" on:click={toggleNavOpen}>
					<span aria-hidden="true"></span>
					<span aria-hidden="true"></span>
					<span aria-hidden="true"></span>
				</button>
			</div>

			<div class={navbarMenuClasses}>
				<div class="navbar-start">
					<NavLink to="/">Devices</NavLink>
					<NavLink to="/bridges">Bridges</NavLink>
					<!-- <button on:click={toggleDarkMode} class="text-gray-300 hover:bg-gray-700 hover:text-white px-3 py-2 rounded-md text-sm font-medium">
						Dark Mode
					</button> -->
				</div>
			</div>
		</nav>
		<div>
			<Route path="/" component="{DevicesPage}" />
			<Route path="/bridges" component="{BridgesPage}" />
		</div>
	</Router>
	<Toast/>
</main>

<style>
</style>
