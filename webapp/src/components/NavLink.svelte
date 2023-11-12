<script>
	import { Link } from 'svelte-routing';
	import classnames from 'classnames';

	export let to = '';

	function getProps({ location, href, isPartiallyCurrent, isCurrent }) {
		const isActive = href === '/' ? isCurrent : isPartiallyCurrent || isCurrent;

		const classes = classnames('transition-colors', 'hover:text-foreground/80', {
			'text-foreground': isActive,
			'text-foreground/60': !isActive,
		});

		return { class: classes };
	}
</script>

<Link to="{to}" getProps="{getProps}">
	<slot />
</Link>
