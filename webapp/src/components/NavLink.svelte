<script>
	import { Link } from 'svelte-routing';
	import classnames from 'classnames';

	export let to = '';

	function getProps({ location, href, isPartiallyCurrent, isCurrent }) {
		const isActive = href === '/' ? isCurrent : isPartiallyCurrent || isCurrent;

		const classes = classnames('navbar-item', {
			'is-active': isActive
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
