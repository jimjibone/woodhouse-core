<script lang="ts">
	import { Router, Route } from 'svelte-routing';
	import classnames from 'classnames';
	import NavLink from './components/NavLink.svelte';
	import DevicesPage from './pages/DevicesPage.svelte';
	import BridgesPage from './pages/BridgesPage.svelte';

	export let url = "";

	let navOpen = false;
	$: navbarBurgerClasses = classnames('navbar-burger', {'is-active': navOpen});
	$: navbarMenuClasses = classnames('navbar-menu', {'is-active': navOpen});

	function toggleNavOpen() {
		navOpen = !navOpen;
	}

	// let deviceInfos: DeviceInfo[] = [];
	// let deviceInfosConnected: boolean = false;
	// const unsubscribeDeviceInfos = deviceInfosStream.subscribeData(value => { deviceInfos = value; });
	// const unsubscribeDeviceInfosConnected = deviceInfosStream.subscribeConnected(value => { deviceInfosConnected = value; });

	// onDestroy(() => {
	// 	unsubscribeDeviceInfos();
	// 	unsubscribeDeviceInfosConnected();
	// });
</script>

<main>
	<Router url="{url}">
		<nav class="navbar" aria-label="main navigation">
			<div class="navbar-brand">
				<a class="navbar-item" href="/">
					<p>Woodhouse</p>
				</a>

				<button class={navbarBurgerClasses} aria-label="menu" aria-expanded="false" data-target="navbarBasicExample" on:click={toggleNavOpen}>
					<span aria-hidden="true"></span>
					<span aria-hidden="true"></span>
					<span aria-hidden="true"></span>
				</button>
			</div>

			<div class={navbarMenuClasses}>
				<div class="navbar-start">
					<NavLink to="/">Home</NavLink>
					<NavLink to="/devices">Devices</NavLink>
					<NavLink to="/bridges">Bridges</NavLink>
				</div>
			</div>
		</nav>
		<div>
		<Route path="/">
			<section class="hero">
				<div class="hero-body">
					<p class="title">
						Woodhouse 4
					</p>
					<p class="subtitle">
						Hero subtitle
					</p>
				</div>
			</section>
		</Route>
		<Route path="/devices" component="{DevicesPage}" />
		<Route path="/bridges" component="{BridgesPage}" />
		</div>
	</Router>
</main>

<style>
</style>
