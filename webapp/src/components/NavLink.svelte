<script>
	import { Link } from 'svelte-routing';
	import classnames from 'classnames';

	export let to = '';

	function getProps({ location, href, isPartiallyCurrent, isCurrent }) {
		const isActive = href === '/' ? isCurrent : isPartiallyCurrent || isCurrent;

		const classes = classnames('navbar-item', {
			'font-medium': isActive,
			'text-slate-600': isActive,
			'dark:text-slate-300': isActive,
			'text-slate-500': !isActive,
			'dark:text-slate-400': !isActive
		});

		return { class: classes };
	}
</script>

<Link to="{to}" getProps="{getProps}">
	<slot />
</Link>

<!-- svelte-ignore css-unused-selector -->
<style>
	.is-active {
		font-weight: 700;
	}
</style>
